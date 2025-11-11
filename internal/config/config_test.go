package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadWithoutAgentRepo(t *testing.T) {
	if err := os.Unsetenv("AGENT_REPO"); err != nil {
		assert.Fail(t, "Error unsetting AGENT_REPO environment variable", err)
	}

	_, err := LoadEnv()
	if err == nil {
		assert.Fail(t, "Expected error reading AGENT_REPO env var")
	}
}

func TestLoadWithoutGitHubToken(t *testing.T) {
	if err := os.Setenv("AGENT_REPO", "test/repo"); err != nil {
		assert.Fail(t, "Error setting AGENT_REPO environment variable", err)
	}

	if err := os.Unsetenv("GITHUB_TOKEN"); err != nil {
		assert.Fail(t, "Error unsetting GITHUB_TOKEN environment variable", err)
	}

	_, err := LoadEnv()
	if err == nil {
		assert.Fail(t, "Expected error reading GITHUB_TOKEN env var")
	}
}

func TestLoadSuccess(t *testing.T) {
	if err := os.Setenv("AGENT_REPO", "test/repo"); err != nil {
		assert.Fail(t, "Error setting AGENT_REPO environment variable", err)
	}
	if err := os.Setenv("GITHUB_TOKEN", "test-token"); err != nil {
		assert.Fail(t, "Error setting GITHUB_TOKEN environment variable", err)
	}
	if err := os.Setenv("BRANCH", "test-branch"); err != nil {
		assert.Fail(t, "Error setting BRANCH environment variable", err)
	}

	cfg, err := LoadEnv()
	assert.NoError(t, err)
	assert.NotNil(t, cfg)
	assert.Equal(t, "test/repo", cfg.AgentRepo)
	assert.Equal(t, "test-token", cfg.GitHubToken)
	assert.Equal(t, "test-branch", cfg.Branch)
}
