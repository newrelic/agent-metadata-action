package oci

import (
	"agent-metadata-action/internal/models"
	"context"
	"fmt"
	"os"
)

func HandleUploads(ociConfig *models.OCIConfig, workspace, agentType, version string) ([]models.ArtifactUploadResult, error) {
	if !ociConfig.IsEnabled() {
		fmt.Println("::debug::OCI upload is not enabled")
		return nil, nil
	}

	fmt.Println("::notice::OCI upload enabled, starting binary uploads...")

	if err := ValidateAllArtifacts(workspace, ociConfig); err != nil {
		return nil, fmt.Errorf("binary validation failed: %w", err)
	}

	client, err := NewClient(ociConfig.Registry, ociConfig.Username, ociConfig.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI client: %w", err)
	}

	ctx := context.Background()
	uploadResults := UploadArtifacts(ctx, client, ociConfig, workspace, agentType, version)

	for _, result := range uploadResults {
		if result.Uploaded {
			fmt.Printf("::notice::Uploaded %s: %s (os: %s, arch: %s, digest: %s, manifest size: %d bytes)\n",
				result.Name, result.Path, result.OS, result.Arch, result.Digest, result.Size)
		} else {
			fmt.Fprintf(os.Stderr, "::error::Failed to upload %s (%s): %s\n",
				result.Name, result.Path, result.Error)
		}
	}

	fmt.Println("::notice::All binaries uploaded successfully")

	// Create manifest index to tag uploaded artifacts with version
	if len(uploadResults) > 0 {
		fmt.Println("::notice::Creating multi-platform manifest index...")
		indexDigest, err := client.CreateManifestIndex(ctx, uploadResults, version)
		if err != nil {
			return uploadResults, fmt.Errorf("failed to create manifest index: %w", err)
		}
		fmt.Printf("::notice::Created manifest index with tag '%s' (digest: %s)\n", version, indexDigest)
	}

	if HasFailures(uploadResults) {
		return uploadResults, fmt.Errorf("one or more binary uploads failed")
	}

	return uploadResults, nil
}
