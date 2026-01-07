package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"agent-metadata-action/internal/client"
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/loader"
	"agent-metadata-action/internal/models"
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

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "::error::%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Validate required environment and setup
	workspace, token, err := validateEnvironment()
	if err != nil {
		return err
	}

	// Create client
	ctx := context.Background()
	client := createMetadataClientFunc(config.GetMetadataURL(), token)

	// Determine which flow to execute
	agentType := config.GetAgentType()
	agentVersion := config.GetVersion()

	if agentType != "" && agentVersion != "" {
		return runAgentFlow(ctx, client, workspace, agentType, agentVersion)
	}

	return runDocsFlow(ctx, client)
}

// validateEnvironment checks required environment variables and workspace
func validateEnvironment() (workspace string, token string, err error) {
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

	fmt.Println("::notice::Environment validated successfully")
	return workspace, token, nil
}

// runAgentFlow handles the agent repository workflow
func runAgentFlow(ctx context.Context, client metadataClient, workspace, agentType, agentVersion string) error {
	fmt.Println("::debug::Running agent repository flow")

	// Check for .fleetControl directory
	fleetControlPath := filepath.Join(workspace, loader.FleetControlDir)
	if _, err := os.Stat(fleetControlPath); err != nil {
		return fmt.Errorf("%s directory does not exist: %s", loader.FleetControlDir, fleetControlPath)
	}

	// Load configuration definitions (required)
	configs, err := loader.ReadConfigurationDefinitions(workspace)
	if err != nil {
		return fmt.Errorf("failed to read configuration definitions: %w", err)
	}
	fmt.Printf("::notice::Loaded %d configuration definitions\n", len(configs))

	// Load agent control (optional)
	agentControl, err := loader.LoadAndEncodeAgentControl(workspace)
	if err != nil {
		fmt.Println("::warn::Unable to load agent control file - continuing without it")
	} else {
		fmt.Println("::notice::Loaded agent control file")
	}

	// Build metadata
	metadata := models.AgentMetadata{
		ConfigurationDefinitions: configs,
		Metadata:                 loader.LoadMetadataForAgents(agentVersion),
		AgentControl:             agentControl,
	}

	printJSON("Agent Metadata", metadata)

	// Send to service
	if err := client.SendMetadata(ctx, agentType, agentVersion, &metadata); err != nil {
		return fmt.Errorf("failed to send metadata for %s: %w", agentType, err)
	}

	fmt.Printf("::notice::Successfully sent metadata for %s version %s\n", agentType, agentVersion)
	return nil
}

// runDocsFlow handles the documentation repository workflow
func runDocsFlow(ctx context.Context, client metadataClient) error {
	fmt.Println("::debug::Running documentation flow")

	// Load metadata from changed MDX files
	metadataList, err := loader.LoadMetadataForDocs()
	if err != nil {
		return fmt.Errorf("failed to load metadata from docs: %w", err)
	}

	if len(metadataList) == 0 {
		fmt.Println("::notice::No metadata changes detected")
		return nil
	}

	fmt.Printf("::notice::Processing %d metadata entries\n", len(metadataList))

	// Send each metadata entry separately
	successCount := 0
	for _, entry := range metadataList {
		if err := sendDocsMetadata(ctx, client, entry); err != nil {
			fmt.Printf("::warn::Failed to send metadata for %s: %v\n", entry.AgentType, err)
			continue
		}
		successCount++
	}

	fmt.Printf("::notice::Successfully sent %d of %d metadata entries\n", successCount, len(metadataList))
	return nil
}

// sendDocsMetadata sends a single docs metadata entry to the service
func sendDocsMetadata(ctx context.Context, client metadataClient, entry loader.MetadataForDocs) error {
	version, _ := entry.AgentMetadataFromDocs["version"].(string)

	metadata := models.AgentMetadata{
		Metadata: entry.AgentMetadataFromDocs,
	}

	printJSON(fmt.Sprintf("Docs Metadata (%s %s)", entry.AgentType, version), entry.AgentMetadataFromDocs)

	if err := client.SendMetadata(ctx, entry.AgentType, version, &metadata); err != nil {
		return err
	}

	fmt.Printf("::notice::Sent metadata for %s version %s\n", entry.AgentType, version)
	return nil
}

// printJSON marshals data to JSON and prints it with a debug annotation
func printJSON(label string, data any) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Printf("::debug::Failed to marshal %s: %v\n", label, err)
		return
	}
	fmt.Printf("::debug::%s: %s\n", label, string(jsonData))
}
