package oci

import (
	"agent-metadata-action/internal/models"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
)

type Client struct {
	repo     *remote.Repository
	registry string
}

func NewClient(registry, username, password string) (*Client, error) {
	repo, err := remote.NewRepository(registry)
	if err != nil {
		return nil, fmt.Errorf("failed to create OCI repository: %w", err)
	}

	repo.Client = &auth.Client{
		Credential: auth.StaticCredential(registry, auth.Credential{
			Username: username,
			Password: password,
		}),
	}

	if strings.HasPrefix(registry, "localhost:") || strings.HasPrefix(registry, "127.0.0.1:") {
		repo.PlainHTTP = true
	}

	fmt.Printf("::debug::OCI client configured: registry=%s, plainHTTP=%v\n", registry, repo.PlainHTTP)

	return &Client{
		repo:     repo,
		registry: registry,
	}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	err := c.repo.Tags(ctx, "", func(tags []string) error {
		return nil
	})

	if err != nil {
		return fmt.Errorf("registry not accessible at %s: %w", c.registry, err)
	}

	return nil
}

func (c *Client) UploadArtifact(ctx context.Context, artifact *models.ArtifactDefinition, artifactPath, agentType, version string) (string, int64, error) {
	tempDir, err := os.MkdirTemp("", "oras-upload-*")
	if err != nil {
		return "", 0, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	fs, err := file.New(tempDir)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create file store: %w", err)
	}
	defer fs.Close()

	layerAnnotations := CreateLayerAnnotations(artifact, version)

	layerDesc, err := fs.Add(ctx, artifact.Name, artifact.GetMediaType(), artifactPath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to add file to store: %w", err)
	}

	layerDesc.Annotations = layerAnnotations

	manifestAnnotations := CreateManifestAnnotations()

	emptyConfig := []byte("{}")
	emptyConfigDesc := ocispec.Descriptor{
		MediaType: "application/vnd.oci.empty.v1+json",
		Digest:    digest.FromBytes(emptyConfig),
		Size:      int64(len(emptyConfig)),
	}

	if err = fs.Push(ctx, emptyConfigDesc, bytes.NewReader(emptyConfig)); err != nil {
		return "", 0, fmt.Errorf("failed to push empty config: %w", err)
	}

	artifactType := artifact.GetArtifactType()
	packOpts := oras.PackManifestOptions{
		ConfigDescriptor:    &emptyConfigDesc,
		Layers:              []ocispec.Descriptor{layerDesc},
		ManifestAnnotations: manifestAnnotations,
	}

	manifestDesc, err := oras.PackManifest(ctx, fs, oras.PackManifestVersion1_1, artifactType, packOpts)
	if err != nil {
		return "", 0, fmt.Errorf("failed to pack manifest: %w", err)
	}

	// Tag temporarily in file store so we can reference it during copy
	tempTag := fmt.Sprintf("temp-%s-%s-%s", version, artifact.OS, artifact.Arch)
	if err = fs.Tag(ctx, manifestDesc, tempTag); err != nil {
		return "", 0, fmt.Errorf("failed to tag manifest: %w", err)
	}

	copyCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()

	copyOpts := oras.CopyOptions{}
	digestRef := manifestDesc.Digest.String()
	fmt.Printf("::debug::Copying artifact %s to registry (digest: %s)\n", artifact.Name, digestRef)
	if _, err = oras.Copy(copyCtx, fs, tempTag, c.repo, digestRef, copyOpts); err != nil {
		return "", 0, fmt.Errorf("failed to copy artifact to registry: %w", err)
	}

	return manifestDesc.Digest.String(), manifestDesc.Size, nil
}

func (c *Client) CreateManifestIndex(ctx context.Context, uploadResults []models.ArtifactUploadResult, version string) (string, error) {
	// Create manifest descriptors for each uploaded artifact
	manifests := make([]ocispec.Descriptor, 0, len(uploadResults))

	for _, result := range uploadResults {
		if !result.Uploaded {
			continue
		}

		digest, err := parseDigest(result.Digest)
		if err != nil {
			return "", fmt.Errorf("invalid digest for %s: %w", result.Name, err)
		}

		platform := &ocispec.Platform{
			OS:           result.OS,
			Architecture: result.Arch,
		}

		artifactType := fmt.Sprintf("application/vnd.newrelic.agent.v1+%s", result.Format)

		manifest := ocispec.Descriptor{
			MediaType:    ocispec.MediaTypeImageManifest,
			Digest:       digest,
			Size:         result.Size,
			Platform:     platform,
			ArtifactType: artifactType,
		}

		manifests = append(manifests, manifest)
	}

	if len(manifests) == 0 {
		return "", fmt.Errorf("no manifests to include in index")
	}

	index := ocispec.Index{
		MediaType: ocispec.MediaTypeImageIndex,
		Manifests: manifests,
		Annotations: map[string]string{
			"org.opencontainers.image.version": version,
		},
	}
	index.SchemaVersion = 2

	indexBytes, err := json.Marshal(index)
	if err != nil {
		return "", fmt.Errorf("failed to marshal index: %w", err)
	}

	indexDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageIndex,
		Digest:    digest.FromBytes(indexBytes),
		Size:      int64(len(indexBytes)),
	}

	fmt.Printf("::debug::Pushing manifest index to %s with tag %s (size: %d bytes)\n",
		c.registry, version, len(indexBytes))
	fmt.Printf("::debug::Index contains %d manifests\n", len(manifests))
	fmt.Printf("::debug::Attempting to push reference: %s\n", version)

	err = c.repo.PushReference(ctx, indexDesc, bytes.NewReader(indexBytes), version)
	if err != nil {
		return "", fmt.Errorf("failed to push manifest index to %s:%s - %w",
			c.registry, version, err)
	}

	fmt.Printf("::debug::Successfully pushed reference: %s\n", version)
	fmt.Printf("::debug::Manifest index push completed successfully\n")

	return indexDesc.Digest.String(), nil
}

func parseDigest(digestStr string) (digest.Digest, error) {
	return digest.Parse(digestStr)
}
