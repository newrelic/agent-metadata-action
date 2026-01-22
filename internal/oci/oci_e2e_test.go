//go:build e2e

package oci

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"agent-metadata-action/internal/models"
)

// setupOCIRegistry spins up a local OCI registry container and returns the registry URL and cleanup function.
func setupOCIRegistry(t *testing.T) (registryURL string, cleanup func()) {
	t.Helper()

	pool, err := dockertest.NewPool("")
	if err != nil {
		t.Fatalf("Could not construct pool: %v", err)
	}

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "registry",
		Tag:        "2",
		Env:        []string{},
	}, func(config *docker.HostConfig) {
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{Name: "no"}
	})
	if err != nil {
		t.Fatalf("Could not start resource: %v", err)
	}

	// Get the dynamic port mapping
	port := resource.GetPort("5000/tcp")
	registryURL = fmt.Sprintf("localhost:%s/test-agents", port)

	// Wait for registry to be ready
	pool.MaxWait = 30 * time.Second
	if err := pool.Retry(func() error {
		resp, err := http.Get(fmt.Sprintf("http://localhost:%s/v2/", port))
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("registry not ready, status: %d", resp.StatusCode)
		}
		return nil
	}); err != nil {
		t.Fatalf("Could not connect to registry: %v", err)
	}

	t.Logf("Registry ready at %s", registryURL)

	cleanup = func() {
		if err := pool.Purge(resource); err != nil {
			t.Logf("Could not purge resource: %v", err)
		}
	}

	return registryURL, cleanup
}

// setupTestWorkspace creates a temporary workspace with test artifacts.
func setupTestWorkspace(t *testing.T) string {
	t.Helper()

	workspace := t.TempDir()
	artifactsDir := filepath.Join(workspace, "artifacts")
	if err := os.MkdirAll(artifactsDir, 0755); err != nil {
		t.Fatalf("Failed to create artifacts directory: %v", err)
	}

	// Create test tar.gz files
	createTestTarGz(t, filepath.Join(artifactsDir, "sample.tar.gz"))
	createTestTarGz(t, filepath.Join(artifactsDir, "sample-arm.tar.gz"))

	// Create test zip file
	createTestZip(t, filepath.Join(artifactsDir, "sample-windows.zip"))

	return workspace
}

// createTestTarGz creates a tar+gzip archive with sample content.
func createTestTarGz(t *testing.T, path string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create tar.gz file: %v", err)
	}
	defer file.Close()

	gzWriter := gzip.NewWriter(file)
	defer gzWriter.Close()

	tarWriter := tar.NewWriter(gzWriter)
	defer tarWriter.Close()

	// Add a sample file with unique content (timestamp-based)
	content := fmt.Sprintf("Sample binary content - %s - %d\n", filepath.Base(path), time.Now().UnixNano())
	header := &tar.Header{
		Name: "sample-binary",
		Mode: 0755,
		Size: int64(len(content)),
	}

	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatalf("Failed to write tar header: %v", err)
	}

	if _, err := tarWriter.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write tar content: %v", err)
	}
}

// createTestZip creates a zip archive with sample content.
func createTestZip(t *testing.T, path string) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create zip file: %v", err)
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	// Add a sample file with unique content (timestamp-based)
	content := fmt.Sprintf("Sample binary content - %s - %d\n", filepath.Base(path), time.Now().UnixNano())
	writer, err := zipWriter.Create("sample-binary.exe")
	if err != nil {
		t.Fatalf("Failed to create zip entry: %v", err)
	}

	if _, err := writer.Write([]byte(content)); err != nil {
		t.Fatalf("Failed to write zip content: %v", err)
	}
}

// verifyManifestIndex fetches and validates the manifest index from the registry.
func verifyManifestIndex(t *testing.T, registryURL, version string, expectedManifestCount int) ocispec.Index {
	t.Helper()

	// Extract registry host and repository from registryURL (format: "localhost:port/repo")
	parts := strings.SplitN(registryURL, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("Invalid registry URL format: %s", registryURL)
	}
	registryHost := parts[0]
	repository := parts[1]

	url := fmt.Sprintf("http://%s/v2/%s/manifests/%s", registryHost, repository, version)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.oci.image.index.v1+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to fetch manifest index: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to fetch manifest index, status %d: %s", resp.StatusCode, string(body))
	}

	var index ocispec.Index
	if err := json.NewDecoder(resp.Body).Decode(&index); err != nil {
		t.Fatalf("Failed to decode manifest index: %v", err)
	}

	// Validate index structure
	if index.MediaType != "application/vnd.oci.image.index.v1+json" {
		t.Errorf("Expected index media type 'application/vnd.oci.image.index.v1+json', got '%s'", index.MediaType)
	}

	if index.SchemaVersion != 2 {
		t.Errorf("Expected schema version 2, got %d", index.SchemaVersion)
	}

	if len(index.Manifests) != expectedManifestCount {
		t.Errorf("Expected %d manifests in index, got %d", expectedManifestCount, len(index.Manifests))
	}

	// Validate index annotations
	if index.Annotations == nil {
		t.Fatal("Expected index annotations to be present")
	}

	versionAnnotation, ok := index.Annotations["org.opencontainers.image.version"]
	if !ok {
		t.Error("Expected 'org.opencontainers.image.version' annotation in index")
	} else if versionAnnotation != version {
		t.Errorf("Expected version annotation '%s', got '%s'", version, versionAnnotation)
	}

	t.Logf("Manifest index validated successfully: %d manifests, version=%s", len(index.Manifests), version)
	return index
}

// verifyArtifactManifest fetches and validates a single artifact manifest.
func verifyArtifactManifest(t *testing.T, registryURL string, manifestDesc ocispec.Descriptor, expectedArtifact models.ArtifactDefinition, version string) {
	t.Helper()

	// Verify descriptor fields
	if manifestDesc.Platform == nil {
		t.Fatal("Expected platform to be set in manifest descriptor")
	}

	if manifestDesc.Platform.OS != expectedArtifact.OS {
		t.Errorf("Expected OS '%s', got '%s'", expectedArtifact.OS, manifestDesc.Platform.OS)
	}

	if manifestDesc.Platform.Architecture != expectedArtifact.Arch {
		t.Errorf("Expected architecture '%s', got '%s'", expectedArtifact.Arch, manifestDesc.Platform.Architecture)
	}

	expectedMediaType := expectedArtifact.GetMediaType()
	if manifestDesc.ArtifactType != expectedMediaType {
		t.Errorf("Expected artifact type '%s', got '%s'", expectedMediaType, manifestDesc.ArtifactType)
	}

	if !strings.HasPrefix(string(manifestDesc.Digest), "sha256:") {
		t.Errorf("Expected SHA256 digest, got '%s'", manifestDesc.Digest)
	}

	if manifestDesc.Size <= 0 {
		t.Errorf("Expected positive size, got %d", manifestDesc.Size)
	}

	// Fetch the actual manifest by digest
	parts := strings.SplitN(registryURL, "/", 2)
	if len(parts) != 2 {
		t.Fatalf("Invalid registry URL format: %s", registryURL)
	}
	registryHost := parts[0]
	repository := parts[1]

	url := fmt.Sprintf("http://%s/v2/%s/manifests/%s", registryHost, repository, manifestDesc.Digest)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		t.Fatalf("Failed to create request: %v", err)
	}
	req.Header.Set("Accept", "application/vnd.oci.image.manifest.v1+json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to fetch manifest: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("Failed to fetch manifest, status %d: %s", resp.StatusCode, string(body))
	}

	var manifest ocispec.Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		t.Fatalf("Failed to decode manifest: %v", err)
	}

	// Verify manifest annotations
	if manifest.Annotations == nil {
		t.Fatal("Expected manifest annotations to be present")
	}

	createdAnnotation, ok := manifest.Annotations["org.opencontainers.image.created"]
	if !ok {
		t.Error("Expected 'org.opencontainers.image.created' annotation in manifest")
	} else {
		// Validate RFC3339 format and recency
		createdTime, err := time.Parse(time.RFC3339, createdAnnotation)
		if err != nil {
			t.Errorf("Expected RFC3339 timestamp, got '%s': %v", createdAnnotation, err)
		} else if time.Since(createdTime) > time.Hour {
			t.Errorf("Expected timestamp to be recent, got %s (age: %s)", createdAnnotation, time.Since(createdTime))
		}
	}

	// Verify layer (should be exactly 1)
	if len(manifest.Layers) != 1 {
		t.Fatalf("Expected exactly 1 layer, got %d", len(manifest.Layers))
	}

	layer := manifest.Layers[0]

	// Verify layer annotations
	if layer.Annotations == nil {
		t.Fatal("Expected layer annotations to be present")
	}

	expectedFilename := expectedArtifact.GetFilename()
	titleAnnotation, ok := layer.Annotations["org.opencontainers.image.title"]
	if !ok {
		t.Error("Expected 'org.opencontainers.image.title' annotation in layer")
	} else if titleAnnotation != expectedFilename {
		t.Errorf("Expected title '%s', got '%s'", expectedFilename, titleAnnotation)
	}

	versionAnnotation, ok := layer.Annotations["org.opencontainers.image.version"]
	if !ok {
		t.Error("Expected 'org.opencontainers.image.version' annotation in layer")
	} else if versionAnnotation != version {
		t.Errorf("Expected version '%s', got '%s'", version, versionAnnotation)
	}

	artifactTypeAnnotation, ok := layer.Annotations["com.newrelic.artifact.type"]
	if !ok {
		t.Error("Expected 'com.newrelic.artifact.type' annotation in layer")
	} else if artifactTypeAnnotation != "binary" {
		t.Errorf("Expected artifact type 'binary', got '%s'", artifactTypeAnnotation)
	}

	// Verify layer media type
	if layer.MediaType != expectedMediaType {
		t.Errorf("Expected layer media type '%s', got '%s'", expectedMediaType, layer.MediaType)
	}

	// Verify layer digest and size
	if !strings.HasPrefix(string(layer.Digest), "sha256:") {
		t.Errorf("Expected SHA256 digest, got '%s'", layer.Digest)
	}

	if layer.Size <= 0 {
		t.Errorf("Expected positive size, got %d", layer.Size)
	}

	t.Logf("Artifact manifest validated: %s/%s, size=%d bytes", expectedArtifact.OS, expectedArtifact.Arch, layer.Size)
}

func TestOCIArtifactUpload(t *testing.T) {
	// Setup registry and workspace
	registryURL, cleanup := setupOCIRegistry(t)
	defer cleanup()

	workspace := setupTestWorkspace(t)

	tests := []struct {
		name                  string
		artifacts             []models.ArtifactDefinition
		version               string
		expectError           bool
		expectedManifestCount int
	}{
		{
			name: "Single Artifact Upload",
			artifacts: []models.ArtifactDefinition{
				{
					Name:   "linux-tar",
					Path:   "./artifacts/sample.tar.gz",
					OS:     "linux",
					Arch:   "amd64",
					Format: "tar+gzip",
				},
			},
			version:               "1.0.0-e2e-single",
			expectError:           false,
			expectedManifestCount: 1,
		},
		{
			name: "Multi-Platform Upload",
			artifacts: []models.ArtifactDefinition{
				{
					Name:   "linux-amd64",
					Path:   "./artifacts/sample.tar.gz",
					OS:     "linux",
					Arch:   "amd64",
					Format: "tar+gzip",
				},
				{
					Name:   "linux-arm64",
					Path:   "./artifacts/sample-arm.tar.gz",
					OS:     "linux",
					Arch:   "arm64",
					Format: "tar+gzip",
				},
				{
					Name:   "windows-amd64",
					Path:   "./artifacts/sample-windows.zip",
					OS:     "windows",
					Arch:   "amd64",
					Format: "zip",
				},
			},
			version:               "1.0.0-e2e-multi",
			expectError:           false,
			expectedManifestCount: 3,
		},
		{
			name: "Upload Failure - Nonexistent File",
			artifacts: []models.ArtifactDefinition{
				{
					Name:   "invalid",
					Path:   "./artifacts/nonexistent.tar.gz",
					OS:     "linux",
					Arch:   "amd64",
					Format: "tar+gzip",
				},
			},
			version:               "1.0.0-e2e-fail",
			expectError:           true,
			expectedManifestCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := &models.OCIConfig{
				Registry:  registryURL,
				Username:  "", // No auth for local registry
				Password:  "", // No auth for local registry
				Artifacts: tt.artifacts,
			}

			// Call the main function under test
			err := HandleUploads(config, workspace, "test-agent", tt.version)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				} else {
					t.Logf("Got expected error: %v", err)
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Verify the manifest index
			index := verifyManifestIndex(t, registryURL, tt.version, tt.expectedManifestCount)

			// Verify each artifact manifest
			for i, manifestDesc := range index.Manifests {
				if i >= len(tt.artifacts) {
					t.Errorf("More manifests in index (%d) than expected artifacts (%d)", len(index.Manifests), len(tt.artifacts))
					break
				}
				verifyArtifactManifest(t, registryURL, manifestDesc, tt.artifacts[i], tt.version)
			}
		})
	}
}
