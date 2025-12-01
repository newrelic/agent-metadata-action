package config

import (
	"os"
)

// GetWorkspace loads the workspace path from environment variables
func GetWorkspace() string {
	return os.Getenv("GITHUB_WORKSPACE")
}
