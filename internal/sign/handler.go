package sign

import (
	"context"
	"fmt"
	"time"

	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/retry"
)

// SignIndex signs the manifest index
// Retries failed signing operations up to 3 times
// Returns error if signing fails after all retries
func SignIndex(ctx context.Context, ociRegistry, indexDigest, version, token, githubRepo string) error {
	logging.Notice(ctx, "Starting manifest index signing...")

	// Parse registry URL once
	registry, repository, err := ParseRegistryURL(ociRegistry)
	if err != nil {
		return retry.NewNonRetryableError(fmt.Errorf("failed to parse registry URL: %w", err))
	}
	logging.Debugf(ctx, "Parsed registry URL - Registry: %s, Repository: %s", registry, repository)

	// Create signing client
	client := NewClient(config.GetSigningURL(), token)

	logging.Log(ctx, "group", "Signing manifest index")
	defer logging.Log(ctx, "endgroup", "")

	// Create signing request for the index
	signingReq := &models.SigningRequest{
		Registry:   registry,
		Repository: repository,
		Tag:        version,
		Digest:     indexDigest,
	}

	// Attempt signing with retries
	retryConfig := retry.Config{
		MaxAttempts: 3,
		BaseDelay:   2 * time.Second,
		Operation:   "Signing",
	}

	err = retry.Do(ctx, retryConfig, func() error {
		return client.SignArtifact(ctx, githubRepo, signingReq)
	})

	if err != nil {
		logging.Errorf(ctx, "Failed to sign manifest index: %v", err)
		return err
	}

	logging.Noticef(ctx, "Successfully signed manifest index (digest: %s)", indexDigest)
	return nil
}
