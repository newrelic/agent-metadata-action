package config

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
// running in the action's own repository (for testing).
// This prevents users from redirecting requests to steal tokens.
func GetMetadataURL() string {
	// Only allow override in the action's own repository for testing
	if url := GetMetadataServiceUrl(); url != "" {
		// GITHUB_ACTION_REPOSITORY = repo where the action code lives
		// GITHUB_REPOSITORY = repo where the action is being used
		// If they match, we're running in the action's own repo (for testing)
		if GetActionRepository() != "" && GetActionRepository() == GetRepository() {
			return url
		}
		// Silently ignore override attempts from other repositories
		// This prevents token theft attacks
	}
	return MetadataURL
}
