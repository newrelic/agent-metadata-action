package oci

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_Success(t *testing.T) {
	tests := []struct {
		name         string
		registry     string
		username     string
		password     string
		expectPlain  bool
	}{
		{
			name:        "standard registry with authentication",
			registry:    "docker.io/newrelic/agents",
			username:    "testuser",
			password:    "testpass",
			expectPlain: false,
		},
		{
			name:        "localhost with plainHTTP",
			registry:    "localhost:5000/test",
			username:    "testuser",
			password:    "testpass",
			expectPlain: true,
		},
		{
			name:        "127.0.0.1 with plainHTTP",
			registry:    "127.0.0.1:5000/test",
			username:    "testuser",
			password:    "testpass",
			expectPlain: true,
		},
		{
			name:        "empty credentials for local registry",
			registry:    "localhost:5000/test-repo",
			username:    "",
			password:    "",
			expectPlain: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.registry, tt.username, tt.password)

			require.NoError(t, err)
			assert.NotNil(t, client)
			assert.Equal(t, tt.registry, client.registry)
			assert.NotNil(t, client.repo)
			assert.Equal(t, tt.expectPlain, client.repo.PlainHTTP)
		})
	}
}

func TestNewClient_InvalidRegistry(t *testing.T) {
	tests := []struct {
		name     string
		registry string
	}{
		{
			name:     "invalid URL with special characters",
			registry: "http://invalid registry with spaces",
		},
		{
			name:     "completely malformed",
			registry: "://invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.registry, "user", "pass")

			assert.Error(t, err)
			assert.Nil(t, client)
			assert.Contains(t, err.Error(), "failed to create OCI repository")
		})
	}
}

func TestParseDigest_Success(t *testing.T) {
	tests := []struct {
		name       string
		digestStr  string
		expectAlgo string
	}{
		{
			name:       "valid sha256 digest",
			digestStr:  "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			expectAlgo: "sha256",
		},
		{
			name:       "valid sha512 digest",
			digestStr:  "sha512:cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
			expectAlgo: "sha512",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseDigest(tt.digestStr)

			require.NoError(t, err)
			assert.Equal(t, tt.expectAlgo, string(result.Algorithm()))
		})
	}
}

func TestParseDigest_Errors(t *testing.T) {
	tests := []struct {
		name      string
		digestStr string
	}{
		{
			name:      "empty string",
			digestStr: "",
		},
		{
			name:      "invalid hex length for sha256",
			digestStr: "sha256:abc123",
		},
		{
			name:      "non-hex characters",
			digestStr: "sha256:gggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggggg",
		},
		{
			name:      "unsupported algorithm",
			digestStr: "md5:abc123",
		},
		{
			name:      "missing colon",
			digestStr: "sha256abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseDigest(tt.digestStr)

			assert.Error(t, err)
		})
	}
}