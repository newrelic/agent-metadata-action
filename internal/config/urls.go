package config

import "os"

// Service URL configuration - hardcoded for security
const (
	// MetadataURL is the instrumentation metadata service endpoint
	MetadataURL = "https://instrumentation-metadata.service.newrelic.com"
)

// ServiceURLs holds all service endpoint URLs
type ServiceURLs struct {
	MetadataURL    string
	OCIRegistryURL string
	SigningURL     string
}

// GetMetadataURL returns the metadata service URL.
// Can be overridden with METADATA_SERVICE_URL environment variable ONLY when
// GITHUB_REPOSITORY matches the action's own repository (for testing).
// This prevents users from redirecting requests to steal tokens.
func GetMetadataURL() string {
	// Only allow override in the action's own repository for testing
	if url := os.Getenv("METADATA_SERVICE_URL"); url != "" {
		repo := GetRepo()
		if repo == "newrelic/agent-metadata-action" {
			return url
		}
		// Silently ignore override attempts from other repositories
		// This prevents token theft attacks
	}
	return MetadataURL
}
