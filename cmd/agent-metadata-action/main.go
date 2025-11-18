package main

import (
	"encoding/json"
	"fmt"
	"os"

	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/models"
)

func main() {
	workspace := config.LoadEnv()

	metadata, err := config.LoadMetadata()
	if err != nil {
		fmt.Fprintf(os.Stderr, "::error::Error loading metadata: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("::debug::Agent version: %s\n", metadata.Version)
	fmt.Printf("::debug::Features: %v\n", metadata.Features)
	fmt.Printf("::debug::Bugs: %v\n", metadata.Bugs)
	fmt.Printf("::debug::Security: %v\n", metadata.Security)

	// If GITHUB_WORKSPACE is set, read configuration definitions (agent repo flow)
	if workspace != "" {
		fmt.Printf("::debug::Reading config from workspace: %s\n", workspace)

		configs, err := config.ReadConfigurationDefinitions(workspace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "::error::Error reading configs: %v\n", err)
			os.Exit(1)
		}

		agentControl, err := config.LoadAndEncodeAgentControl(workspace)
		if err != nil {
			fmt.Fprintf(os.Stderr, "::error::Error reading agent control: %v\n", err)
			os.Exit(1)
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
		// Docs workflow: only output metadata
		fmt.Println("::notice::Running in metadata-only mode (no workspace provided)")
		// @todo use the INPUT_AGENT_TYPE along with the metadata to call the InstrumentationMetadata service for updating the agent in NGEP with extra metadata
		printJSON("Metadata", metadata)
	}
}

func printJSON(label string, data any) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "::error::Failed to marshal %s: %v\n", label, err)
		os.Exit(1)
	}
	fmt.Printf("::debug::%s: %s\n", label, string(jsonData))
}
