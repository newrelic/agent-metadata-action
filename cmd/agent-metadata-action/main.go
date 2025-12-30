package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"agent-metadata-action/internal/client"
	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/loader"
	"agent-metadata-action/internal/models"
)

// metadataClient interface for testing
type metadataClient interface {
	SendMetadata(ctx context.Context, agentType string, metadata *models.AgentMetadata) error
}

// createMetadataClientFunc is a variable that holds the function to create a metadata client
// This allows tests to override the implementation
var createMetadataClientFunc = func(baseURL, token string) metadataClient {
	return client.NewInstrumentationClient(baseURL, token)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "::error::%v\n", err)
		os.Exit(1)
	}
}

func run() error {
	workspace := config.GetWorkspace()

	// Workspace is required
	if workspace == "" {
		return fmt.Errorf("GITHUB_WORKSPACE is required but not set")
	}

	// Validate workspace directory exists
	if _, err := os.Stat(workspace); err != nil {
		return fmt.Errorf("error reading configs: workspace directory does not exist: %s", workspace)
	}

	// Get OAuth token from environment (set by action.yml authentication step)
	token := config.GetToken()
	if token == "" {
		return fmt.Errorf("NEWRELIC_TOKEN is required but not set")
	}
	fmt.Println("::notice::OAuth token loaded from environment")

	// Create instrumentation client
	ctx := context.Background()
	metadataClient := createMetadataClientFunc(config.GetMetadataURL(), token)

	agentType := config.GetAgentType()
	agentVersion := config.GetVersion()

	if agentType != "" && agentVersion != "" { // Scenario 1: Agent repo flow
		fmt.Println("::debug::Agent scenario")
		fleetControlPath := workspace + "/.fleetControl"
		if _, err := os.Stat(fleetControlPath); err != nil {
			return fmt.Errorf("error ./fleetControl folder does not exist: %s", fleetControlPath)
		} else {
			fmt.Printf("::debug::Reading config from workspace: %s\n", workspace)

			configs, err := loader.ReadConfigurationDefinitions(workspace)
			if err != nil {
				return fmt.Errorf("error reading configs: %w", err)
			}

			agentControl, err := loader.LoadAndEncodeAgentControl(workspace)
			if err != nil {
				return fmt.Errorf("error reading agent control: %w", err)
			}

			fmt.Println("::notice::Successfully read configs file")
			fmt.Printf("::debug::Found %d configs\n", len(configs))

			metadata := loader.LoadMetadataForAgents(agentVersion)

			agentMetadata := models.AgentMetadata{
				ConfigurationDefinitions: configs,
				Metadata:                 metadata,
				AgentControl:             agentControl,
			}

			printJSON("Agent Metadata", agentMetadata)

			// Send metadata to instrumentation service
			fmt.Println("::debug::Sending metadata to instrumentation service...")
			if err := metadataClient.SendMetadata(ctx, agentType, &agentMetadata); err != nil {
				return fmt.Errorf("failed to send metadata: %w", err)
			}
			fmt.Println("::notice::Successfully sent metadata to instrumentation service")
		}
	} else { // Scenario 2: Docs flow
		fmt.Println("::debug::Docs scenario")

		metadata, err := loader.LoadMetadataForDocs()

		if err != nil {
			return fmt.Errorf("error reading metadata: %w", err)
		}

		for _, currMetadata := range metadata {
			fmt.Printf("::debug::Found metadata for %s %s", currMetadata.AgentType, currMetadata.AgentMetadataFromDocs.Version)
			printJSON("Agent Metadata", currMetadata.AgentMetadataFromDocs)

			// TODO: Implement metadata service call for docs workflow
		}
	}

	return nil
}

func printJSON(label string, data any) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "::error::Failed to marshal %s: %v\n", label, err)
		os.Exit(1)
	}
	fmt.Printf("::debug::%s: %s\n", label, string(jsonData))
}
