package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"agent-metadata-action/internal/client"
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/loader"
	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/oci"

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

	if err := run(nrApp); err != nil {
		logging.Errorf(ctx, "%v", err)
		os.Exit(1)
	}

	if nrApp != nil {
		logging.Notice(ctx, "Shutting down New Relic - waiting up to 15 seconds to send data...")
		nrApp.Shutdown(15 * time.Second)
		logging.Notice(ctx, "New Relic shutdown complete")
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
	// Force failure for testing
	return "", "", fmt.Errorf("INTENTIONAL TEST FAILURE - checking error handling")

	workspace = config.GetWorkspace()
	if workspace == "" {
		return "", "", fmt.Errorf("GITHUB_WORKSPACE is required but not set")
	}

	if _, err := os.Stat(workspace); err != nil {
		return "", "", fmt.Errorf("workspace directory does not exist: %s", workspace)
	}

	token = config.GetToken()
	if token == "" {
		return "", "", fmt.Errorf("NEWRELIC_TOKEN is required but not set")
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
		return fmt.Errorf("failed to read configuration definitions: %w", err)
	}
	logging.Noticef(ctx, "Loaded %d configuration definitions", len(configs))

	// Load agent control definitions (optional)
	agentControl, err := loader.ReadAgentControlDefinitions(ctx, workspace)
	if err != nil {
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

	// Send to service
	if err := client.SendMetadata(ctx, agentType, agentVersion, &metadata); err != nil {
		return fmt.Errorf("failed to send metadata for %s: %w", agentType, err)
	}

	// Handle OCI binary uploads (optional)
	ociConfig, err := oci.LoadConfig()
	if err != nil {
		return fmt.Errorf("error loading OCI config: %w", err)
	}

	if err := oci.HandleUploads(ctx, &ociConfig, workspace, agentType, agentVersion); err != nil {
		return fmt.Errorf("binary upload failed: %w", err)
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
