package sign

import (
	"context"
	"fmt"
	"time"

	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/models"
)

const (
	// MaxRetries is the maximum number of retry attempts for signing
	MaxRetries = 3
	// RetryDelay is the base delay between retries (will be multiplied by attempt number)
	RetryDelay = 2 * time.Second
)

// SignIndex signs the manifest index with the given digest and tag
// Makes a single signing request to oci-signer for the index
func SignIndex(registry, indexDigest, version, token, githubRepo string) error {
	if indexDigest == "" {
		return fmt.Errorf("index digest is required for signing")
	}

	fmt.Printf("::notice::Signing manifest index with tag '%s'...\n", version)

	// Parse registry URL
	registryHost, repository, err := ParseRegistryURL(registry)
	if err != nil {
		return fmt.Errorf("failed to parse registry URL: %w", err)
	}
	fmt.Printf("::debug::Parsed registry URL - Registry: %s, Repository: %s\n", registryHost, repository)


	client := NewClient(config.GetSigningURL(), token)

	// Create signing request for index
	signingReq := &models.SigningRequest{
		Registry:   registryHost,
		Repository: repository,
		Tag:        version,     // Index tag (e.g., "v1.2.3")
		Digest:     indexDigest, // Index digest
	}

	// Attempt signing with retries
	ctx := context.Background()
	var signErr error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		if attempt > 1 {
			delay := time.Duration(attempt-1) * RetryDelay
			fmt.Printf("::debug::Retry attempt %d/%d after %s delay...\n", attempt, MaxRetries, delay)
			time.Sleep(delay)
		}

		signErr = client.SignArtifact(ctx, githubRepo, signingReq)
		if signErr == nil {
			fmt.Printf("::notice::Successfully signed manifest index (tag: %s, digest: %s)\n", version, indexDigest)
			return nil
		}

		if attempt < MaxRetries {
			fmt.Printf("::warn::Signing attempt %d failed: %v - will retry\n", attempt, signErr)
		}
	}

	fmt.Printf("::error::Failed to sign manifest index after %d attempts: %v\n", MaxRetries, signErr)
	return fmt.Errorf("failed to sign manifest index after %d attempts: %w", MaxRetries, signErr)
}
