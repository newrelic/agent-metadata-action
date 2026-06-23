package config

import "os"

// Service URL configuration - hardcoded for security
const (
	// MetadataURL is the instrumentation metadata service endpoint
	MetadataURL = "https://instrumentation-metadata.service.newrelic.com"

	// SigningURL is the OCI artifact signing service endpoint
	SigningURL = "https://oci-signer.service.newrelic.com"
)

// ServiceURLs holds all service endpoint URLs
type ServiceURLs struct {
	MetadataURL    string
	OCIRegistryURL string
	SigningURL     string
}

// isURLOverrideAllowed returns true when it is safe to honour a service URL override.
// Override is allowed in two cases:
//  1. Running inside the action's own repository (CI self-testing).
//  2. Running outside of GitHub Actions entirely (local CLI runs).
//
// It is never allowed in external repositories during a real GitHub Actions run,
// which is the scenario that could be used to redirect requests and steal tokens.
func isURLOverrideAllowed() bool {
	return GetRepo() == "newrelic/agent-metadata-action" || os.Getenv("GITHUB_ACTIONS") != "true"
}

// GetMetadataURL returns the metadata service URL.
// Can be overridden with METADATA_SERVICE_URL when running locally or in the
// action's own repository. Override is blocked for external repositories during
// real GitHub Actions runs to prevent token theft.
func GetMetadataURL() string {
	if url := os.Getenv("METADATA_SERVICE_URL"); url != "" && isURLOverrideAllowed() {
		return url
	}
	return MetadataURL
}

// GetSigningURL returns the OCI signing service URL.
// Follows the same override rules as GetMetadataURL.
func GetSigningURL() string {
	if url := os.Getenv("SIGNING_SERVICE_URL"); url != "" && isURLOverrideAllowed() {
		return url
	}
	return SigningURL
}
