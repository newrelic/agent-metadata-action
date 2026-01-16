package oci

import (
	"encoding/json"
	"fmt"
	"strings"

	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/models"
)

func LoadConfig() (models.OCIConfig, error) {
	registry := config.GetOCIRegistry()
	username := config.GetOCIUsername()
	password := config.GetOCIPassword()
	binariesJSON := config.GetBinaries()

	config := models.OCIConfig{
		Registry:  strings.TrimSpace(registry),
		Username:  strings.TrimSpace(username),
		Password:  password,
		Artifacts: []models.ArtifactDefinition{},
	}

	if binariesJSON != "" {
		if err := json.Unmarshal([]byte(binariesJSON), &config.Artifacts); err != nil {
			return config, fmt.Errorf("failed to parse binaries JSON: %w", err)
		}
	}

	if err := config.Validate(); err != nil {
		return config, err
	}

	return config, nil
}
