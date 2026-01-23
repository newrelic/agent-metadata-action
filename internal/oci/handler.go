package oci

import (
	"context"
	"fmt"

	"agent-metadata-action/internal/logging"
	"agent-metadata-action/internal/models"
)

func HandleUploads(ctx context.Context, ociConfig *models.OCIConfig, workspace, agentType, version string) ([]models.ArtifactUploadResult, error) {
	if !ociConfig.IsEnabled() {
		logging.Debug(ctx, "OCI upload is not enabled")
		return nil, nil
	}

	logging.Notice(ctx, "OCI upload enabled, starting binary uploads...")

	if err := ValidateAllArtifacts(ctx, workspace, ociConfig); err != nil {
		logging.NoticeErrorWithCategory(ctx, err, "oci.validation", map[string]interface{}{
			"error.operation": "validate_artifacts",
			"oci.registry":    ociConfig.Registry,
			"artifact.count":  len(ociConfig.Artifacts),
		})
		return nil, fmt.Errorf("binary validation failed: %w", err)
	}

	client, err := NewClient(ctx, ociConfig.Registry, ociConfig.Username, ociConfig.Password)
	if err != nil {
		logging.NoticeErrorWithCategory(ctx, err, "oci.client", map[string]interface{}{
			"error.operation": "create_oci_client",
			"oci.registry":    ociConfig.Registry,
		})
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	uploadResults := UploadArtifacts(ctx, client, ociConfig, workspace, agentType, version)

	for _, result := range uploadResults {
		if result.Uploaded {
			logging.Noticef(ctx, "Uploaded %s: %s (os: %s, arch: %s, digest: %s, manifest size: %d bytes)",
				result.Name, result.Path, result.OS, result.Arch, result.Digest, result.Size)
		} else {
			artifactErr := fmt.Errorf("upload failed: %s", result.Error)
			logging.NoticeErrorWithCategory(ctx, artifactErr, "oci.artifact.upload", map[string]interface{}{
				"error.operation": "upload_artifact",
				"artifact.name":   result.Name,
				"artifact.path":   result.Path,
				"artifact.os":     result.OS,
				"artifact.arch":   result.Arch,
				"oci.registry":    ociConfig.Registry,
			})
			logging.Errorf(ctx, "Failed to upload %s (%s): %s",
				result.Name, result.Path, result.Error)
		}
	}

	logging.Notice(ctx, "All binaries uploaded successfully")

	// Create manifest index to tag uploaded artifacts with version
	if len(uploadResults) > 0 {
		logging.Notice(ctx, "Creating multi-platform manifest index...")
		indexDigest, err := client.CreateManifestIndex(ctx, uploadResults, version)
		if err != nil {
			logging.NoticeErrorWithCategory(ctx, err, "oci.manifest", map[string]interface{}{
				"error.operation": "create_manifest_index",
				"oci.registry":    ociConfig.Registry,
				"manifest.count":  len(uploadResults),
			})
			return uploadResults, fmt.Errorf("failed to create manifest index: %w", err)
		}
		logging.Noticef(ctx, "Created manifest index with tag '%s' (digest: %s)", version, indexDigest)
	}

	if HasFailures(uploadResults) {
		return uploadResults, fmt.Errorf("one or more binary uploads failed")
	}

	logging.Notice(ctx, "All binaries uploaded successfully")

	return uploadResults, nil
}
