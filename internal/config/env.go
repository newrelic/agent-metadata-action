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

// GetToken loads the newrelic token from the environment variables
func GetToken() string {
	return os.Getenv("NEWRELIC_TOKEN")
}

// GetOCIRegistry loads the OCI registry from environment variables
func GetOCIRegistry() string {
	return os.Getenv("INPUT_OCI_REGISTRY")
}

// GetOCIUsername loads the OCI username from environment variables
func GetOCIUsername() string {
	return os.Getenv("INPUT_OCI_USERNAME")
}

// GetOCIPassword loads the OCI password from environment variables
func GetOCIPassword() string {
	return os.Getenv("INPUT_OCI_PASSWORD")
}

// GetBinaries loads the binaries JSON from environment variables
func GetBinaries() string {
	return os.Getenv("INPUT_BINARIES")

// GetNRAgentLicenseKey gets the license key to use the go agent and monitor this app
func GetNRAgentLicenseKey() string {
	return os.Getenv("APM_CONTROL_NR_LICENSE_KEY")
}

// SetNRAgentHost sets the host to use for the go agent that will be used to monitor this app
func SetNRAgentHost() error {
	err := os.Setenv("NEW_RELIC_HOST", "staging-collector.newrelic.com")
	if err != nil {
		return err
	}
	return nil
}
