package main

import (
	"context"
	"testing"

	"agent-metadata-action/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockDisableClient is a mock implementation for testing
type mockDisableClient struct {
	receivedAgentType    string
	receivedAgentVersion string
}

func (m *mockDisableClient) DisableAgentVersion(ctx context.Context, agentType string, agentVersion string) error {
	m.receivedAgentType = agentType
	m.receivedAgentVersion = agentVersion
	return nil
}

type mockFailingDisableClient struct{}

func (m *mockFailingDisableClient) DisableAgentVersion(ctx context.Context, agentType string, agentVersion string) error {
	return assert.AnError
}

func setDisableEnv(t *testing.T) {
	t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")
	t.Setenv("INPUT_AGENT_TYPE", "java")
	t.Setenv("INPUT_VERSION", "1.2.3")
}

func TestRun_Success(t *testing.T) {
	originalCreateClient := createDisableClientFunc
	mockClient := &mockDisableClient{}
	createDisableClientFunc = func(baseURL, token string) disableClient {
		return mockClient
	}
	defer func() { createDisableClientFunc = originalCreateClient }()

	setDisableEnv(t)

	getStdout, getStderr := testutil.CaptureOutput(t)

	err := run(nil)

	outputStr := getStdout()
	stderrStr := getStderr()

	require.NoError(t, err)
	assert.Equal(t, "java", mockClient.receivedAgentType)
	assert.Equal(t, "1.2.3", mockClient.receivedAgentVersion)
	assert.Contains(t, outputStr, "Disabling java version 1.2.3")
	assert.Contains(t, outputStr, "Disabled java version 1.2.3")
	assert.NotContains(t, stderrStr, "::error::")
}

func TestRun_MissingToken(t *testing.T) {
	setDisableEnv(t)
	t.Setenv("NEWRELIC_TOKEN", "")

	err := run(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "NEWRELIC_TOKEN is required")
}

func TestRun_MissingAgentType(t *testing.T) {
	setDisableEnv(t)
	t.Setenv("INPUT_AGENT_TYPE", "")

	err := run(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent-type is required")
}

func TestRun_MissingVersion(t *testing.T) {
	setDisableEnv(t)
	t.Setenv("INPUT_VERSION", "")

	err := run(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestRun_ClientError(t *testing.T) {
	originalCreateClient := createDisableClientFunc
	createDisableClientFunc = func(baseURL, token string) disableClient {
		return &mockFailingDisableClient{}
	}
	defer func() { createDisableClientFunc = originalCreateClient }()

	setDisableEnv(t)

	err := run(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to disable java version 1.2.3")
}
