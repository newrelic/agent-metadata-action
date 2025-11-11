package main

import (
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

	configs, err := config.ReadConfigs(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "::error::Error reading configs: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("::notice::Successfully fetched configs file")
	printConfigs(configs)

}

func printConfigs(configs []models.ConfigJson) {
	for i, configJson := range configs {
		fmt.Printf("Config %d:\n", i+1)
		fmt.Printf("  Name: %s\n", configJson.Name)
		fmt.Printf("  Slug: %s\n", configJson.Slug)
		fmt.Printf("  Platform: %s\n", configJson.Platform)
		fmt.Printf("  Description: %s\n", configJson.Description)
		fmt.Printf("  Type: %s\n", configJson.Type)
		fmt.Printf("  Version: %s\n", configJson.Version)
		fmt.Println()
	}
}
