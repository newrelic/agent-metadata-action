package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"agent-metadata-action/internal/client"
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/loader"
	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/oci"
	"agent-metadata-action/internal/sign"

	"github.com/newrelic/go-agent/v3/newrelic"
)

// metadataClient interface for testing
type metadataClient interface {
	SendMetadata(ctx context.Context, agentType string, agentVersion string, metadata *models.AgentMetadata) error
}

// createMetadataClientFunc is a variable that holds the function to create a metadata client
// This allows tests to override the implementation
var createMetadataClientFunc = func(baseURL, token string) metadataClient {
	return client.NewInstrumentationClient(baseURL, token)
}

// ociHandleUploadsFunc is a variable that holds the function to handle OCI uploads
// This allows tests to override the implementation
var ociHandleUploadsFunc = func(ctx context.Context, ociConfig *models.OCIConfig, workspace, version string) (string, error) {
	return oci.HandleUploads(ctx, ociConfig, workspace, version)
}

// initNewRelic initializes the New Relic application
// Returns nil if APM_CONTROL_NR_LICENSE_KEY is not set (silent no-op mode)
func initNewRelic(ctx context.Context) *newrelic.Application {
	licenseKey := config.GetNRAgentLicenseKey()
	if licenseKey == "" {
		logging.Warn(ctx, "Failed to init New Relic - missing license key")
		return nil
	}

	// Hardcode staging environment
	err := config.SetNRAgentHost()
	if err != nil {
		logging.Warnf(ctx, "Failed to init New Relic, missing host: %v", err)
		return nil
	}
	logging.Notice(ctx, "Using New Relic staging environment")

	app, err := newrelic.NewApplication(
		newrelic.ConfigAppName("agent-metadata-action"),
		newrelic.ConfigLicense(licenseKey),
		newrelic.ConfigDebugLogger(os.Stdout),
		newrelic.ConfigDistributedTracerEnabled(true),
		newrelic.ConfigAppLogForwardingEnabled(true),
		newrelic.ConfigFromEnvironment(), // This reads NEW_RELIC_HOST
		newrelic.ConfigLabels(map[string]string{
			"team": "APM Control Team",
		}),
	)

	if err != nil {
		logging.Warnf(ctx, "Failed to init New Relic: %v", err)
		return nil
	}

	logging.Notice(ctx, "New Relic APM enabled - waiting for connection...")

	// Wait for the app to connect (max 10 seconds)
	if err := app.WaitForConnection(10 * time.Second); err != nil {
		logging.Warnf(ctx, "New Relic connection timeout: %v - will try to send data anyway", err)
	} else {
		logging.Notice(ctx, "New Relic connected successfully")
	}

	return app
}

func main() {
	// Create base context for early logging
	ctx := context.Background()

	nrApp := initNewRelic(ctx)

	// Run the action
	err := run(nrApp)

	// Ensure New Relic shuts down gracefully (even on error)
	// Must happen BEFORE os.Exit() since os.Exit bypasses defers
	if nrApp != nil {
		logging.Notice(ctx, "Shutting down New Relic - waiting up to 15 seconds to send data...")
		nrApp.Shutdown(15 * time.Second)
		logging.Notice(ctx, "New Relic shutdown complete")
	}

	// Exit with appropriate code
	if err != nil {
		logging.Errorf(ctx, "%v", err)
		os.Exit(1)
	}
}

func run(nrApp *newrelic.Application) error {
	// Create context
	ctx := context.Background()

	// Start transaction if New Relic is enabled
	if nrApp != nil {
		txn := nrApp.StartTransaction("agent-metadata-action")
		defer txn.End()

		// Add transaction to context for logging
		ctx = newrelic.NewContext(ctx, txn)
		logging.Debug(ctx, "New Relic transaction started")
		defer logging.Debug(ctx, "New Relic transaction ended")
	}

	// Validate required environment and setup
	workspace, token, err := validateEnvironment(ctx)
	if err != nil {
		return err
	}

	// Create metadataClient
	metadataClient := createMetadataClientFunc(config.GetMetadataURL(), token)

	// Determine which flow to execute
	agentType := config.GetAgentType()
	agentVersion := config.GetVersion()

	if agentType != "" && agentVersion != "" {
		return runAgentFlow(ctx, metadataClient, workspace, agentType, agentVersion)
	}

	return runDocsFlow(ctx, metadataClient)
}

// validateEnvironment checks required environment variables and workspace
func validateEnvironment(ctx context.Context) (workspace string, token string, err error) {
	workspace = config.GetWorkspace()
	if workspace == "" {
		err := fmt.Errorf("GITHUB_WORKSPACE is required but not set")
		logging.NoticeErrorWithCategory(ctx, err, "environment.validation", map[string]interface{}{
			"error.operation": "validate_workspace",
			"error.field":     "GITHUB_WORKSPACE",
		})
		return "", "", err
	}

	if _, err := os.Stat(workspace); err != nil {
		noticeErr := fmt.Errorf("workspace directory does not exist: %s", workspace)
		logging.NoticeErrorWithCategory(ctx, noticeErr, "environment.validation", map[string]interface{}{
			"error.operation": "validate_workspace",
			"workspace.path":  workspace,
		})
		return "", "", noticeErr
	}

	token = config.GetToken()
	if token == "" {
		err := fmt.Errorf("NEWRELIC_TOKEN is required but not set")
		logging.NoticeErrorWithCategory(ctx, err, "environment.validation", map[string]interface{}{
			"error.operation": "validate_token",
			"error.field":     "NEWRELIC_TOKEN",
		})
		return "", "", err
	}

	logging.Notice(ctx, "Environment validated successfully")
	return workspace, token, nil
}

// runAgentFlow handles the agent repository workflow
func runAgentFlow(ctx context.Context, client metadataClient, workspace, agentType, agentVersion string) error {
	logging.Debugf(ctx, "Running agent repository flow for %s version %s", agentType, agentVersion)

	// Check for .fleetControl directory
	fleetControlPath := filepath.Join(workspace, config.GetRootFolderForAgentRepo())
	if _, err := os.Stat(fleetControlPath); err != nil {
		return fmt.Errorf("%s directory does not exist: %s", config.GetRootFolderForAgentRepo(), fleetControlPath)
	}

	// Load configuration definitions (required)
	configs, err := loader.ReadConfigurationDefinitions(ctx, workspace)
	if err != nil {
		logging.NoticeErrorWithCategory(ctx, err, "configuration.load", map[string]interface{}{
			"error.operation": "load_configuration_definitions",
			"agent.type":      agentType,
			"agent.version":   agentVersion,
			"workflow.type":   "agent",
		})
		return fmt.Errorf("failed to read configuration definitions: %w", err)
	}
	logging.Noticef(ctx, "Loaded %d configuration definitions", len(configs))

	// Load agent control definitions (optional)
	agentControl, err := loader.ReadAgentControlDefinitions(ctx, workspace)
	if err != nil {
		logging.NoticeErrorWithCategory(ctx, err, "configuration.load", map[string]interface{}{
			"error.operation": "load_agent_control_definitions",
			"agent.type":      agentType,
			"agent.version":   agentVersion,
			"workflow.type":   "agent",
			"error.severity":  "warning", // Graceful error
		})
		logging.Warnf(ctx, "Unable to load agent control definitions: %v - continuing without them", err)
		agentControl = nil
	} else {
		logging.Noticef(ctx, "Loaded %d agent control definitions", len(agentControl))
	}

	// Build metadata
	metadata := models.AgentMetadata{
		ConfigurationDefinitions: configs,
		Metadata:                 loader.LoadMetadataForAgents(agentVersion),
		AgentControlDefinitions:  agentControl,
	}

	printJSON(ctx, "Agent Metadata", metadata)

	// Step 1: Send to metadata service
	if err := client.SendMetadata(ctx, agentType, agentVersion, &metadata); err != nil {
		return fmt.Errorf("failed to send metadata for %s: %w", agentType, err)
	}

	ociConfig, err := oci.LoadConfig()
	if err != nil {
		logging.NoticeErrorWithCategory(ctx, err, "oci.configuration", map[string]interface{}{
			"error.operation": "load_oci_config",
			"agent.type":      agentType,
			"agent.version":   agentVersion,
		})
		return fmt.Errorf("error loading OCI config: %w", err)
	}

	if ociConfig.IsEnabled() {
		// Step 2: Upload binaries
		indexDigest, err := ociHandleUploadsFunc(ctx, &ociConfig, workspace, agentVersion)
		if err != nil {
			return fmt.Errorf("binary upload failed: %w", err)
		}

		// Step 3: Sign the manifest index
		githubRepo := config.GetRepo()
		if githubRepo == "" {
			return fmt.Errorf("GITHUB_REPOSITORY environment variable is required for artifact signing")
		}

		// Extract repository name from full path (e.g., "agent-metadata-action" from "newrelic/agent-metadata-action")
		repoParts := strings.Split(githubRepo, "/")
		repoName := repoParts[len(repoParts)-1]

		token := config.GetToken()
		if token == "" {
			return fmt.Errorf("NEWRELIC_TOKEN is required for artifact signing")
		}

		if err := sign.SignIndex(ctx, ociConfig.Registry, indexDigest, agentVersion, token, repoName); err != nil {
			return fmt.Errorf("artifact signing failed: %w", err)
		}
	}

	logging.Noticef(ctx, "Successfully sent metadata for %s version %s", agentType, agentVersion)
	return nil
}

// runDocsFlow handles the documentation repository workflow
func runDocsFlow(ctx context.Context, client metadataClient) error {
	logging.Debug(ctx, "Running documentation flow")

	// Load metadata from changed MDX files
	metadataList, err := loader.LoadMetadataForDocs(ctx)
	if err != nil {
		return fmt.Errorf("failed to load metadata from docs: %w", err)
	}

	if len(metadataList) == 0 {
		logging.Notice(ctx, "No metadata changes detected")
		return nil
	}

	logging.Noticef(ctx, "Processing %d metadata entries", len(metadataList))

	// Send each metadata entry separately
	successCount := 0
	for _, entry := range metadataList {
		if err := sendDocsMetadata(ctx, client, entry); err != nil {
			logging.Errorf(ctx, "Failed to send metadata for %s: %v", entry.AgentType, err)
			continue
		}
		successCount++
	}

	logging.Noticef(ctx, "Successfully sent %d of %d metadata entries", successCount, len(metadataList))
	return nil
}

// sendDocsMetadata sends a single docs metadata entry to the service
func sendDocsMetadata(ctx context.Context, client metadataClient, entry loader.MetadataForDocs) error {
	version, _ := entry.AgentMetadataFromDocs["version"].(string)

	metadata := models.AgentMetadata{
		Metadata: entry.AgentMetadataFromDocs,
	}

	printJSON(ctx, fmt.Sprintf("Docs Metadata (%s %s)", entry.AgentType, version), entry.AgentMetadataFromDocs)

	if err := client.SendMetadata(ctx, entry.AgentType, version, &metadata); err != nil {
		return err
	}

	logging.Noticef(ctx, "Sent metadata for %s version %s", entry.AgentType, version)
	return nil
}

// printJSON marshals data to JSON and prints it with a debug annotation
func printJSON(ctx context.Context, label string, data any) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		logging.Debugf(ctx, "Failed to marshal %s: %v", label, err)
		return
	}
	logging.Debugf(ctx, "%s: %s", label, string(jsonData))
}
