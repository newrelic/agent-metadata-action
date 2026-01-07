package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMetadataURL(t *testing.T) {
	tests := []struct {
		name               string
		actionRepository   string // GITHUB_ACTION_REPOSITORY (where action code lives)
		repository         string // GITHUB_REPOSITORY (where action is being used)
		metadataServiceURL string
		expectedURL        string
		description        string
	}{
		{
			name:               "running in action's own repo with override",
			actionRepository:   "newrelic/agent-metadata-action",
			repository:         "newrelic/agent-metadata-action",
			metadataServiceURL: "https://test-override.example.com",
			expectedURL:        "https://test-override.example.com",
			description:        "Should allow URL override when running in action's own repository",
		},
		{
			name:               "running in different repo attempts override",
			actionRepository:   "newrelic/agent-metadata-action",
			repository:         "attacker/malicious-repo",
			metadataServiceURL: "https://attacker-site.example.com",
			expectedURL:        MetadataURL,
			description:        "Should ignore URL override when repositories don't match (security)",
		},
		{
			name:               "action repository env not set",
			actionRepository:   "",
			repository:         "newrelic/agent-metadata-action",
			metadataServiceURL: "https://test.example.com",
			expectedURL:        MetadataURL,
			description:        "Should use default URL when GITHUB_ACTION_REPOSITORY is not set",
		},
		{
			name:               "no override env var set",
			actionRepository:   "newrelic/agent-metadata-action",
			repository:         "newrelic/agent-metadata-action",
			metadataServiceURL: "",
			expectedURL:        MetadataURL,
			description:        "Should use default URL when no override is set",
		},
		{
			name:               "renamed repo still works",
			actionRepository:   "newrelic/new-action-name",
			repository:         "newrelic/new-action-name",
			metadataServiceURL: "https://test-override.example.com",
			expectedURL:        "https://test-override.example.com",
			description:        "Should allow override even if action repo is renamed (no hardcoding)",
		},
		{
			name:               "different organization",
			actionRepository:   "different-org/agent-metadata-action",
			repository:         "different-org/agent-metadata-action",
			metadataServiceURL: "https://test-override.example.com",
			expectedURL:        "https://test-override.example.com",
			description:        "Should allow override if forked to different organization (for testing forks)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			t.Setenv("GITHUB_ACTION_REPOSITORY", tt.actionRepository)
			t.Setenv("GITHUB_REPOSITORY", tt.repository)
			t.Setenv("METADATA_SERVICE_URL", tt.metadataServiceURL)

			// Call the function
			result := GetMetadataURL()

			// Assert the result
			assert.Equal(t, tt.expectedURL, result, tt.description)
		})
	}
}
