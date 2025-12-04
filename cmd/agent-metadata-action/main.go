package main

import (
	"encoding/json"
	"fmt"
	"os"

	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/models"
)

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
		return fmt.Errorf("Error: GITHUB_WORKSPACE is required but not set")
	}

	// Validate workspace directory exists
	if _, err := os.Stat(workspace); err != nil {
		return fmt.Errorf("Error reading configs: workspace directory does not exist: %s", workspace)
	}

	agentType := os.Getenv("INPUT_AGENT_TYPE")
	agentVersion, err := config.LoadVersion()
	if err != nil {
		return err
	}

	if agentType != "" && agentVersion != "" { // Scenario 1: Agent repo flow
		fmt.Println("::debug::Agent scenario")
		fleetControlPath := workspace + "/.fleetControl"
		if _, err := os.Stat(fleetControlPath); err == nil {
			fmt.Printf("::debug::Reading config from workspace: %s\n", workspace)

			configs, err := config.ReadConfigurationDefinitions(workspace)
			if err != nil {
				return fmt.Errorf("Error reading configs: %w", err)
			}

			agentControl, err := config.LoadAndEncodeAgentControl(workspace)
			if err != nil {
				return fmt.Errorf("Error reading agent control: %w", err)
			}

			fmt.Println("::notice::Successfully read configs file")
			fmt.Printf("::debug::Found %d configs\n", len(configs))

			metadata := config.LoadMetadataForAgents(agentVersion)

			agentMetadata := models.AgentMetadata{
				ConfigurationDefinitions: configs,
				Metadata:                 metadata,
				AgentControl:             agentControl,
			}

			// @todo use the AgentMetadata object to call the InstrumentationMetadata service to add/update the agent in NGEP
			printJSON("Agent Metadata", agentMetadata)
		}
	} else { // Scenario 2: Docs repo flow
		fmt.Println("::debug::Docs scenario")
		metadata, err := config.LoadMetadataForDocs()
		if err != nil {
			return fmt.Errorf("Error reading metadata: %w", err)
		}
		for _, currMetadata := range metadata {
			agentMetadata := models.AgentMetadata{
				Metadata: currMetadata,
			}
			printJSON("Agent Metadata", agentMetadata)
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
