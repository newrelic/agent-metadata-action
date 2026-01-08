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
		fleetControlPath := filepath.Join(workspace, config.GetRootFolderForAgentRepo())
		if _, err := os.Stat(fleetControlPath); err != nil {
			return fmt.Errorf("error expected root folder does not exist: %s", fleetControlPath)
		} else {
			fmt.Printf("::debug::Reading config from workspace: %s\n", workspace)

			configs, err := loader.ReadConfigurationDefinitions(workspace)
			if err != nil {
				return fmt.Errorf("error reading configs: %w", err)
			}

			fmt.Println("::notice::Successfully read configs file")
			fmt.Printf("::debug::Found %d configs\n", len(configs))

			agentControl, err := loader.ReadAgentControlDefinitions(workspace)
			if err != nil {
				fmt.Println("::debug::Unable to read agent control files")
			} else {
				fmt.Println("::notice::Successfully read agent control files")
			}

			metadata := loader.LoadMetadataForAgents(agentVersion)

			// @todo will need to add agentRequirements here in a future PR

			agentMetadata := models.AgentMetadata{
				ConfigurationDefinitions: configs,
				Metadata:                 metadata,
				AgentControlDefinitions:  agentControl,
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
			// warn but don't fail the docs push - this data is useful but not required at this time
			fmt.Printf("::warn::Error reading metadata %s \n", err)
		} else {
			for _, currMetadata := range metadata {
				fmt.Printf("::debug::Found metadata for %s %s \n", currMetadata.AgentType, currMetadata.AgentMetadataFromDocs["version"])
				printJSON("Docs Metadata", currMetadata.AgentMetadataFromDocs)

				currAgentMetadata := models.AgentMetadata{
					Metadata: currMetadata.AgentMetadataFromDocs,
				}

				if err := metadataClient.SendMetadata(ctx, currMetadata.AgentType, &currAgentMetadata); err != nil {
					fmt.Printf("::warn::Failed to send docs metadata to instrumentation service for agent type: %s \n", currMetadata.AgentType)
				} else {
					fmt.Printf("::notice::Successfully sent docs metadata to instrumentation service for agent type:  %s \n", currMetadata.AgentType)
				}
			}
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
