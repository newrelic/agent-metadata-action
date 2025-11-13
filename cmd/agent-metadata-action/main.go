package main

import (
	"encoding/json"
	"fmt"
	"os"

	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/models"
)

func main() {
	cfg, err := config.LoadEnv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "::error::Error loading config: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("::debug::Reading config from workspace: %s\n", cfg)

	configs, err := config.ReadConfigurationDefinitions(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "::error::Error reading configs: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("::notice::Successfully read configs file")
	fmt.Printf("::debug::Found %d configs\n", len(configs))
	printConfigs(configs)
}

func printConfigs(configs []models.ConfigurationDefinition) {
	for _, cfg := range configs {
		jsonData, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "::warning::Failed to marshal config: %v\n", err)
			continue
		}
		fmt.Printf("::debug::Config: %s\n", string(jsonData))
	}
}
