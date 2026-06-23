package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMetadataURL_DefaultsToProduction(t *testing.T) {
	t.Setenv("METADATA_SERVICE_URL", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITHUB_REPOSITORY", "")

	assert.Equal(t, MetadataURL, GetMetadataURL())
}

func TestGetMetadataURL_OverrideAllowedOutsideGitHubActions(t *testing.T) {
	// GITHUB_ACTIONS is not set — simulates local CLI run
	t.Setenv("METADATA_SERVICE_URL", "https://staging.example.com")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITHUB_REPOSITORY", "newrelic/dotnet-agent")

	assert.Equal(t, "https://staging.example.com", GetMetadataURL())
}

func TestGetMetadataURL_OverrideAllowedInActionOwnRepo(t *testing.T) {
	t.Setenv("METADATA_SERVICE_URL", "https://staging.example.com")
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_REPOSITORY", "newrelic/agent-metadata-action")

	assert.Equal(t, "https://staging.example.com", GetMetadataURL())
}

func TestGetMetadataURL_OverrideBlockedForExternalRepoInGitHubActions(t *testing.T) {
	t.Setenv("METADATA_SERVICE_URL", "https://attacker.example.com")
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_REPOSITORY", "newrelic/dotnet-agent")

	assert.Equal(t, MetadataURL, GetMetadataURL())
}

func TestGetSigningURL_DefaultsToProduction(t *testing.T) {
	t.Setenv("SIGNING_SERVICE_URL", "")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITHUB_REPOSITORY", "")

	assert.Equal(t, SigningURL, GetSigningURL())
}

func TestGetSigningURL_OverrideAllowedOutsideGitHubActions(t *testing.T) {
	t.Setenv("SIGNING_SERVICE_URL", "https://staging-signer.example.com")
	t.Setenv("GITHUB_ACTIONS", "")
	t.Setenv("GITHUB_REPOSITORY", "newrelic/dotnet-agent")

	assert.Equal(t, "https://staging-signer.example.com", GetSigningURL())
}

func TestGetSigningURL_OverrideBlockedForExternalRepoInGitHubActions(t *testing.T) {
	t.Setenv("SIGNING_SERVICE_URL", "https://attacker.example.com")
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_REPOSITORY", "newrelic/dotnet-agent")

	assert.Equal(t, SigningURL, GetSigningURL())
}

func TestIsURLOverrideAllowed(t *testing.T) {
	tests := []struct {
		name          string
		repo          string
		githubActions string
		expected      bool
	}{
		{"own repo in GH Actions", "newrelic/agent-metadata-action", "true", true},
		{"own repo outside GH Actions", "newrelic/agent-metadata-action", "", true},
		{"external repo outside GH Actions", "newrelic/dotnet-agent", "", true},
		{"external repo in GH Actions", "newrelic/dotnet-agent", "true", false},
		{"no repo outside GH Actions", "", "", true},
		{"no repo in GH Actions", "", "true", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("GITHUB_REPOSITORY", tt.repo)
			os.Setenv("GITHUB_ACTIONS", tt.githubActions)
			t.Cleanup(func() {
				os.Unsetenv("GITHUB_REPOSITORY")
				os.Unsetenv("GITHUB_ACTIONS")
			})
			assert.Equal(t, tt.expected, isURLOverrideAllowed())
		})
	}
}
