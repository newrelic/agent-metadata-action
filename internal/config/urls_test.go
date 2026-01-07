package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMetadataURL(t *testing.T) {
	tests := []struct {
		name                   string
		repository             string
		metadataServiceURL     string
		expectedURL            string
		description            string
	}{
		{
			name:                   "correct repo with override",
			repository:             "newrelic/agent-metadata-action",
			metadataServiceURL:     "https://test-override.example.com",
			expectedURL:            "https://test-override.example.com",
			description:            "Should allow URL override in the action's own repository",
		},
		{
			name:                   "incorrect repo attempts override",
			repository:             "attacker/malicious-repo",
			metadataServiceURL:     "https://attacker-site.example.com",
			expectedURL:            MetadataURL,
			description:            "Should ignore URL override from unauthorized repository (security)",
		},
		{
			name:                   "incorrect repo different owner",
			repository:             "random-user/some-repo",
			metadataServiceURL:     "https://steal-tokens.example.com",
			expectedURL:            MetadataURL,
			description:            "Should ignore URL override from different repository owner",
		},
		{
			name:                   "no override env var set",
			repository:             "newrelic/agent-metadata-action",
			metadataServiceURL:     "",
			expectedURL:            MetadataURL,
			description:            "Should use default URL when no override is set",
		},
		{
			name:                   "no repository env var",
			repository:             "",
			metadataServiceURL:     "https://test.example.com",
			expectedURL:            MetadataURL,
			description:            "Should use default URL when repository is not set",
		},
		{
			name:                   "similar but incorrect repo name",
			repository:             "newrelic/agent-metadata-action-fork",
			metadataServiceURL:     "https://fork-attempt.example.com",
			expectedURL:            MetadataURL,
			description:            "Should reject similar but non-exact repository names",
		},
		{
			name:                   "case sensitivity check",
			repository:             "NewRelic/Agent-Metadata-Action",
			metadataServiceURL:     "https://case-test.example.com",
			expectedURL:            MetadataURL,
			description:            "Should reject repository name with different casing",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables
			t.Setenv("GITHUB_REPOSITORY", tt.repository)
			t.Setenv("METADATA_SERVICE_URL", tt.metadataServiceURL)

			// Call the function
			result := GetMetadataURL()

			// Assert the result
			assert.Equal(t, tt.expectedURL, result, tt.description)
		})
	}
}

func TestGetMetadataURL_DefaultValue(t *testing.T) {
	// Test with no environment variables set at all
	result := GetMetadataURL()
	assert.Equal(t, MetadataURL, result, "Should return default URL when no environment variables are set")
	assert.Equal(t, "https://instrumentation-metadata.service.newrelic.com", result, "Default URL should match expected production endpoint")
}
