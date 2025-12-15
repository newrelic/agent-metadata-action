package config

import (
	"os"
)

// GetWorkspace loads the workspace path from environment variables
func GetWorkspace() string {
	return os.Getenv("GITHUB_WORKSPACE")
}

// GetAgentType loads the agent type from environment variables
func GetAgentType() string {
	return os.Getenv("INPUT_AGENT_TYPE")
}

// GetVersion loads the version from environment variables
func GetVersion() string {
	return os.Getenv("INPUT_VERSION")
}

// GetEventPath loads the GitHub event path from environment variables
func GetEventPath() string {
	return os.Getenv("GITHUB_EVENT_PATH")
}
