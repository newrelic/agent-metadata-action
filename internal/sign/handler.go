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

// SignArtifacts signs all uploaded artifacts in batch
// Retries failed signing operations up to MaxRetries times
// Returns error if any artifact fails to sign after all retries
func SignArtifacts(ctx context.Context, results []models.ArtifactUploadResult, ociRegistry, token, githubRepo, version string) error {
	if len(results) == 0 {
		logging.Debug(ctx, "No artifacts to sign")
		return nil
	}

	logging.Noticef(ctx, "Starting artifact signing for %d artifacts...", len(results))

	// Parse registry URL once
	registry, repository, err := ParseRegistryURL(ociRegistry)
	if err != nil {
		return fmt.Errorf("failed to parse registry URL: %w", err)
	}
	logging.Debugf(ctx, "Parsed registry URL - Registry: %s, Repository: %s", registry, repository)

	// Create signing client
	client := NewClient(config.GetSigningURL(), token)

	successCount := 0
	failCount := 0

	// Sign each artifact
	for i, result := range results {
		// Skip artifacts that failed to upload
		if !result.Uploaded {
			logging.Warnf(ctx, "Skipping signing for %s - upload failed", result.Name)
			continue
		}

		logging.Log(ctx, "group", fmt.Sprintf("Signing artifact %d/%d: %s", i+1, len(results), result.Name))

		// Create signing request
		signingReq := &models.SigningRequest{
			Registry:   registry,
			Repository: repository,
			Tag:        result.Tag,
			Digest:     result.Digest,
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
				logging.Noticef(ctx, "Successfully signed artifact %s (digest: %s)", result.Name, result.Digest)
				results[i].Signed = true
				successCount++
				break
			}

			// Log retry info
			if attempt < MaxRetries {
				logging.Warnf(ctx, "Signing attempt %d failed: %v - will retry", attempt, signErr)
			}
		}

		// Check if all retries failed
		if signErr != nil {
			logging.Errorf(ctx, "Failed to sign artifact %s after %d attempts: %v", result.Name, MaxRetries, signErr)
			results[i].Signed = false
			results[i].SigningError = signErr.Error()
			failCount++

			logging.Log(ctx, "endgroup", "")
			return fmt.Errorf("failed to sign artifact %s after %d attempts: %w", result.Name, MaxRetries, signErr)
		}

		logging.Log(ctx, "endgroup", "")
	}

	// Summary
	logging.Noticef(ctx, "Artifact signing complete: %d/%d signed successfully", successCount, successCount+failCount)

	return nil
}
