package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadEnv_Success(t *testing.T) {
	// Set up environment (mimics what would be set from calling workflow when checkout precedes this action)
	t.Setenv("GITHUB_WORKSPACE", "/tmp/workspace")

	cfg := GetWorkspace()
	assert.Equal(t, "/tmp/workspace", cfg)
}

func TestLoadEnv_NotSet(t *testing.T) {
	// Don't set up environment (mimics docs workflow where workspace is not needed)
	t.Setenv("GITHUB_WORKSPACE", "")

	cfg := GetWorkspace()
	assert.Empty(t, cfg)
}
