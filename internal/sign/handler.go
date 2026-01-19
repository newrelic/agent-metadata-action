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

// SignArtifacts signs all uploaded artifacts in batch
// Retries failed signing operations up to MaxRetries times
// Returns error if any artifact fails to sign after all retries
func SignArtifacts(results []models.ArtifactUploadResult, ociRegistry, token, githubRepo, version string) error {
	if len(results) == 0 {
		fmt.Println("::debug::No artifacts to sign")
		return nil
	}

	fmt.Printf("::notice::Starting artifact signing for %d artifacts...\n", len(results))

	// Parse registry URL once
	registry, repository, err := ParseRegistryURL(ociRegistry)
	if err != nil {
		return fmt.Errorf("failed to parse registry URL: %w", err)
	}
	fmt.Printf("::debug::Parsed registry URL - Registry: %s, Repository: %s\n", registry, repository)

	// Create signing client
	client := NewClient(config.GetSigningURL(), token)

	ctx := context.Background()
	successCount := 0
	failCount := 0

	// Sign each artifact
	for i, result := range results {
		// Skip artifacts that failed to upload
		if !result.Uploaded {
			fmt.Printf("::warn::Skipping signing for %s - upload failed\n", result.Name)
			continue
		}

		fmt.Printf("::group::Signing artifact %d/%d: %s\n", i+1, len(results), result.Name)

		// Create signing request
		signingReq := &models.SigningRequest{
			Registry:   registry,
			Repository: repository,
			Tag:        version,
			Digest:     result.Digest,
		}

		// Attempt signing with retries
		var signErr error
		for attempt := 1; attempt <= MaxRetries; attempt++ {
			if attempt > 1 {
				delay := time.Duration(attempt-1) * RetryDelay
				fmt.Printf("::debug::Retry attempt %d/%d after %s delay...\n", attempt, MaxRetries, delay)
				time.Sleep(delay)
			}

			signErr = client.SignArtifact(ctx, githubRepo, signingReq)
			if signErr == nil {
				// Success!
				fmt.Printf("::notice::Successfully signed artifact %s (digest: %s)\n", result.Name, result.Digest)
				results[i].Signed = true
				successCount++
				break
			}

			// Log retry info
			if attempt < MaxRetries {
				fmt.Printf("::warn::Signing attempt %d failed: %v - will retry\n", attempt, signErr)
			}
		}

		// Check if all retries failed
		if signErr != nil {
			fmt.Printf("::error::Failed to sign artifact %s after %d attempts: %v\n", result.Name, MaxRetries, signErr)
			results[i].Signed = false
			results[i].SigningError = signErr.Error()
			failCount++

			fmt.Println("::endgroup::")
			return fmt.Errorf("failed to sign artifact %s after %d attempts: %w", result.Name, MaxRetries, signErr)
		}

		fmt.Println("::endgroup::")
	}

	// Summary
	fmt.Printf("::notice::Artifact signing complete: %d/%d signed successfully\n", successCount, successCount+failCount)

	return nil
}
