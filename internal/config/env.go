package config

import (
	"os"
	"strings"
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
}

// GetTags loads the tags JSON from environment variables
func GetTags() string {
	return os.Getenv("INPUT_TAGS")
}

// GetBindingsOverride loads the bindings override JSON from environment variables
func GetBindingsOverride() string {
	return os.Getenv("INPUT_BINDINGS_OVERRIDE")
}

// GetNRAgentLicenseKey gets the license key to use the go agent and monitor this app
func GetNRAgentLicenseKey() string {
	return os.Getenv("APM_CONTROL_NR_LICENSE_KEY")
}

// GetConfigDirectory loads the config directory from environment variables
// Returns the directory where configuration files are located (relative to workspace)
func GetConfigDirectory() string {
	return os.Getenv("INPUT_CONFIG_DIRECTORY")
}

// GetMonitoringType loads the monitoring type from environment variables
func GetMonitoringType() string {
	return os.Getenv("INPUT_MONITORING_TYPE")
}

// GetDisplayName loads the display name from environment variables
func GetDisplayName() string {
	return os.Getenv("INPUT_DISPLAY_NAME")
}

// GetValidateAgentType reports whether agent type definitions should be validated via
// newrelic-agent-control-cli before being sent to the instrumentation service.
func GetValidateAgentType() bool {
	return strings.EqualFold(os.Getenv("INPUT_VALIDATE_AGENT_TYPE"), "true")
}

// GetAgentControlCLIImageTag loads the Docker image tag for newrelic-agent-control-cli
// from environment variables. Returns "" if not set, letting the caller fall back to a default.
func GetAgentControlCLIImageTag() string {
	return os.Getenv("INPUT_AGENT_CONTROL_CLI_TAG")
}

// GetChannel loads the release channel to promote to from environment variables.
func GetChannel() string {
	return os.Getenv("INPUT_CHANNEL")
}

// GetPlatform loads the target platform for a release channel promotion from environment variables.
func GetPlatform() string {
	return os.Getenv("INPUT_PLATFORM")
}

// GetOperatingSystem loads the target operating system for a release channel promotion
// from environment variables.
func GetOperatingSystem() string {
	return os.Getenv("INPUT_OPERATING_SYSTEM")
}

// GetNote loads the optional audit-trail note for a release channel promotion from
// environment variables.
func GetNote() string {
	return os.Getenv("INPUT_NOTE")
}

// SetNRAgentHost sets the host to use for the go agent that will be used to monitor this app
func SetNRAgentHost() error {
	err := os.Setenv("NEW_RELIC_HOST", "staging-collector.newrelic.com")
	if err != nil {
		return err
	}
	return nil
}
