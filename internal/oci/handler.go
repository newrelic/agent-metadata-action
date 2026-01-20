package oci

import (
	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
	"context"
	"fmt"
)

func HandleUploads(ctx context.Context, ociConfig *models.OCIConfig, workspace, agentType, version string) error {
	if !ociConfig.IsEnabled() {
		logging.Debug(ctx, "OCI upload is not enabled")
		return nil
	}

	logging.Notice(ctx, "OCI upload enabled, starting binary uploads...")

	if err := ValidateAllArtifacts(ctx, workspace, ociConfig); err != nil {
		return fmt.Errorf("binary validation failed: %w", err)
	}

	client, err := NewClient(ctx, ociConfig.Registry, ociConfig.Username, ociConfig.Password)
	if err != nil {
		return fmt.Errorf("failed to create OCI client: %w", err)
	}

	uploadResults := UploadArtifacts(ctx, client, ociConfig, workspace, agentType, version)

	for _, result := range uploadResults {
		if result.Uploaded {
			logging.Noticef(ctx, "Uploaded %s: %s (os: %s, arch: %s, digest: %s, manifest size: %d bytes)",
				result.Name, result.Path, result.OS, result.Arch, result.Digest, result.Size)
		} else {
			logging.Errorf(ctx, "Failed to upload %s (%s): %s",
				result.Name, result.Path, result.Error)
		}
	}

	// Create manifest index to tag uploaded artifacts with version
	if len(uploadResults) > 0 {
		logging.Notice(ctx, "Creating multi-platform manifest index...")
		indexDigest, err := client.CreateManifestIndex(ctx, uploadResults, version)
		if err != nil {
			return fmt.Errorf("failed to create manifest index: %w", err)
		}
		logging.Noticef(ctx, "Created manifest index with tag '%s' (digest: %s)", version, indexDigest)
	}

	if HasFailures(uploadResults) {
		return fmt.Errorf("one or more binary uploads failed")
	}

	logging.Notice(ctx, "All binaries uploaded successfully")

	return nil
}
