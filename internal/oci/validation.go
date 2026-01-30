package oci

import (
	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func ValidateBinaryPath(workspacePath, binaryPath string) error {
	// Reject paths with directory traversal
	if strings.Contains(binaryPath, "..") {
		return fmt.Errorf("invalid binary path: contains directory traversal")
	}

	// Resolve to absolute path
	var fullPath string
	if filepath.IsAbs(binaryPath) {
		fullPath = binaryPath
	} else {
		fullPath = filepath.Join(workspacePath, binaryPath)
	}

	resolvedPath, err := filepath.Abs(fullPath)
	if err != nil {
		return fmt.Errorf("failed to resolve binary path: %w", err)
	}

	// Check file exists and is readable
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return fmt.Errorf("binary file not found or not readable: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("binary path is a directory, not a file")
	}

	if info.Size() == 0 {
		return fmt.Errorf("binary file is empty")
	}

	return nil
}

func ValidateAllArtifacts(ctx context.Context, workspacePath string, config *models.OCIConfig) error {
	for _, artifact := range config.Artifacts {
		if err := ValidateBinaryPath(workspacePath, artifact.Path); err != nil {
			return fmt.Errorf("validation failed for artifact '%s': %w", artifact.Name, err)
		}
	}
	logging.Debug(ctx, "All artifact validations passed")
	return nil
}

func ResolveArtifactPath(workspacePath, artifactPath string) (string, error) {
	if filepath.IsAbs(artifactPath) {
		return artifactPath, nil
	}
	return filepath.Join(workspacePath, artifactPath), nil
}
