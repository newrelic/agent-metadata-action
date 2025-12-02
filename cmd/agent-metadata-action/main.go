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
	// Validate agent type here for now - may move once there is code to call InstrumentationMetadata service
	if err := validateAgentType(); err != nil {
		return err
	}

	workspace := config.GetWorkspace()

	// Workspace is required
	if workspace == "" {
		return fmt.Errorf("Error: GITHUB_WORKSPACE is required but not set")
	}

	// Validate workspace directory exists
	if _, err := os.Stat(workspace); err != nil {
		return fmt.Errorf("Error reading configs: workspace directory does not exist: %s", workspace)
	}

	metadata, err := config.LoadMetadata()
	if err != nil {
		return fmt.Errorf("Error loading metadata: %w", err)
	}
	fmt.Printf("::debug::Agent version: %s\n", metadata.Version)
	fmt.Printf("::debug::Features: %v\n", metadata.Features)
	fmt.Printf("::debug::Bugs: %v\n", metadata.Bugs)
	fmt.Printf("::debug::Security: %v\n", metadata.Security)

	// Check if .fleetControl directory exists to determine flow (agent repo vs docs)
	fleetControlPath := workspace + "/.fleetControl"
	if _, err := os.Stat(fleetControlPath); err == nil {
		// Scenario 1: Agent repo flow - .fleetControl directory exists
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

		agentMetadata := models.AgentMetadata{
			ConfigurationDefinitions: configs,
			Metadata:                 metadata,
			AgentControl:             agentControl,
		}

		// @todo use the AgentMetadata object to call the InstrumentationMetadata service to add/update the agent in NGEP
		printJSON("Agent Metadata", agentMetadata)
	} else {
		// Scenario 2: Docs workflow
		fmt.Println("::notice::Running in metadata-only mode (.fleetControl not found, using MDX files)")
		// @todo use the INPUT_AGENT_TYPE along with the metadata to call the InstrumentationMetadata service for updating the agent in NGEP with extra metadata
		printJSON("Metadata", metadata)
	}

	return nil
}

func validateAgentType() error {
	agentType := os.Getenv("INPUT_AGENT_TYPE")
	if agentType == "" {
		return fmt.Errorf("agent-type is required: INPUT_AGENT_TYPE not set")
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
