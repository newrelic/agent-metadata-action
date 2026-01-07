package config

import (
	"os"
)

// GetWorkspace loads the workspace path from environment variables
func GetWorkspace() string {
	return os.Getenv("GITHUB_WORKSPACE")
}

// GetRepository loads the repo from environment variables
// This is the repository that is USING the action
func GetRepository() string {
	return os.Getenv("GITHUB_REPOSITORY")
}

// GetActionRepository loads the repository where the action code lives
// This is used to verify the action is running in its own repository (for testing overrides)
func GetActionRepository() string {
	return os.Getenv("GITHUB_ACTION_REPOSITORY")
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

// GetToken loads the NR token from environment variables
func GetToken() string {
	return os.Getenv("NEWRELIC_TOKEN")
}

// GetMetadataServiceUrl loads the url for the metadata service from environment variables
func GetMetadataServiceUrl() string {
	return os.Getenv("METADATA_SERVICE_URL")
}
