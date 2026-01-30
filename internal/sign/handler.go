package sign

import (
	"context"
	"fmt"
	"time"

	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
)

const (
	// MaxRetries is the maximum number of retry attempts for signing
	MaxRetries = 3
	// RetryDelay is the base delay between retries (will be multiplied by attempt number)
	RetryDelay = 2 * time.Second
)

// SignIndex signs the manifest index
// Retries failed signing operations up to MaxRetries times
// Returns error if signing fails after all retries
func SignIndex(ctx context.Context, ociRegistry, indexDigest, version, token, githubRepo string) error {
	logging.Notice(ctx, "Starting manifest index signing...")

	// Parse registry URL once
	registry, repository, err := ParseRegistryURL(ociRegistry)
	if err != nil {
		return fmt.Errorf("failed to parse registry URL: %w", err)
	}
	logging.Debugf(ctx, "Parsed registry URL - Registry: %s, Repository: %s", registry, repository)

	// Create signing client
	client := NewClient(config.GetSigningURL(), token)

	logging.Log(ctx, "group", "Signing manifest index")

	// Create signing request for the index
	signingReq := &models.SigningRequest{
		Registry:   registry,
		Repository: repository,
		Tag:        version,
		Digest:     indexDigest,
	}

	// Attempt signing with retries
	var signErr error
	for attempt := 1; attempt <= MaxRetries; attempt++ {
		if attempt > 1 {
			delay := time.Duration(attempt-1) * RetryDelay
			logging.Debugf(ctx, "Retry attempt %d/%d after %s delay...", attempt, MaxRetries, delay)
			time.Sleep(delay)
		}

		signErr = client.SignArtifact(ctx, githubRepo, signingReq)
		if signErr == nil {
			// Success!
			logging.Noticef(ctx, "Successfully signed manifest index (digest: %s)", indexDigest)
			logging.Log(ctx, "endgroup", "")
			return nil
		}

		// Log retry info
		if attempt < MaxRetries {
			logging.Warnf(ctx, "Signing attempt %d failed: %v - will retry", attempt, signErr)
		}
	}

	// All retries failed
	logging.Errorf(ctx, "Failed to sign manifest index after %d attempts: %v", MaxRetries, signErr)
	logging.Log(ctx, "endgroup", "")
	return fmt.Errorf("failed to sign manifest index after %d attempts: %w", MaxRetries, signErr)
}
