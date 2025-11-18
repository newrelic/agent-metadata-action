package config

import (
	"os"
)

// LoadEnv loads the workspace path from environment variables
// Returns empty string if GITHUB_WORKSPACE is not set (optional for docs workflow)
func LoadEnv() string {
	return os.Getenv("GITHUB_WORKSPACE")
}
