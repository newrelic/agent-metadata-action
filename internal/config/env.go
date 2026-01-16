package config

import (
	"os"
)

// GetWorkspace loads the GH workspace path from environment variables
func GetWorkspace() string {
	return os.Getenv("GITHUB_WORKSPACE")
}

// GetRepo loads the GH repo from environment variables
func GetRepo() string {
	return os.Getenv("GITHUB_REPOSITORY")
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

func GetToken() string {
	return os.Getenv("NEWRELIC_TOKEN")
}

func SetNRHost() error {
	err := os.Setenv("NEW_RELIC_HOST", "staging-collector.newrelic.com")
	if err != nil {
		return err
	}
	return nil
}
