package main

import (
	"context"
	"testing"

	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/testutil"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockPromotionClient is a mock implementation for testing
type mockPromotionClient struct {
	previousVersion *string
	receivedReq     *models.SetReleaseChannelRequest
}

func (m *mockPromotionClient) PromoteToReleaseChannel(ctx context.Context, agentType string, agentVersion string, req *models.SetReleaseChannelRequest) (*models.ReleaseChannelPromotion, error) {
	m.receivedReq = req
	return &models.ReleaseChannelPromotion{
		AgentType:       agentType,
		Platform:        req.Platform,
		OperatingSystem: req.OperatingSystem,
		Channel:         req.Channel,
		CurrentVersion:  agentVersion,
		PreviousVersion: m.previousVersion,
		PromotedAt:      "2026-08-19T00:00:00Z",
		PromotedBy:      "test-principal",
	}, nil
}

type mockFailingPromotionClient struct{}

func (m *mockFailingPromotionClient) PromoteToReleaseChannel(ctx context.Context, agentType string, agentVersion string, req *models.SetReleaseChannelRequest) (*models.ReleaseChannelPromotion, error) {
	return nil, assert.AnError
}

func setPromotionEnv(t *testing.T) {
	t.Setenv("NEWRELIC_TOKEN", "mock-token-for-testing")
	t.Setenv("INPUT_AGENT_TYPE", "java")
	t.Setenv("INPUT_VERSION", "1.2.3")
	t.Setenv("INPUT_CHANNEL", "REGULAR")
	t.Setenv("INPUT_PLATFORM", "HOST")
	t.Setenv("INPUT_OPERATING_SYSTEM", "LINUX")
}

func TestRun_Success(t *testing.T) {
	originalCreateClient := createPromotionClientFunc
	previous := "1.2.2"
	mockClient := &mockPromotionClient{previousVersion: &previous}
	createPromotionClientFunc = func(baseURL, token string) promotionClient {
		return mockClient
	}
	defer func() { createPromotionClientFunc = originalCreateClient }()

	setPromotionEnv(t)

	getStdout, getStderr := testutil.CaptureOutput(t)

	err := run(nil)

	outputStr := getStdout()
	stderrStr := getStderr()

	require.NoError(t, err)
	assert.Equal(t, "HOST", mockClient.receivedReq.Platform)
	assert.Equal(t, "LINUX", mockClient.receivedReq.OperatingSystem)
	assert.Equal(t, "REGULAR", mockClient.receivedReq.Channel)
	assert.Contains(t, outputStr, "Promoting java version 1.2.3 to release channel REGULAR")
	assert.Contains(t, outputStr, "Promoted java 1.2.3 to channel REGULAR (previous: 1.2.2)")
	assert.NotContains(t, stderrStr, "::error::")
}

func TestRun_SuccessNoPreviousVersion(t *testing.T) {
	originalCreateClient := createPromotionClientFunc
	mockClient := &mockPromotionClient{}
	createPromotionClientFunc = func(baseURL, token string) promotionClient {
		return mockClient
	}
	defer func() { createPromotionClientFunc = originalCreateClient }()

	setPromotionEnv(t)

	getStdout, _ := testutil.CaptureOutput(t)

	err := run(nil)

	outputStr := getStdout()

	require.NoError(t, err)
	assert.Contains(t, outputStr, "Promoted java 1.2.3 to channel REGULAR (first promotion for platform=HOST os=LINUX)")
}

func TestRun_MissingToken(t *testing.T) {
	setPromotionEnv(t)
	t.Setenv("NEWRELIC_TOKEN", "")

	err := run(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "NEWRELIC_TOKEN is required")
}

func TestRun_MissingAgentType(t *testing.T) {
	setPromotionEnv(t)
	t.Setenv("INPUT_AGENT_TYPE", "")

	err := run(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent-type is required")
}

func TestRun_MissingVersion(t *testing.T) {
	setPromotionEnv(t)
	t.Setenv("INPUT_VERSION", "")

	err := run(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "version is required")
}

func TestRun_ValidationFailure(t *testing.T) {
	setPromotionEnv(t)
	t.Setenv("INPUT_PLATFORM", "BAREMETAL")

	err := run(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid release channel promotion request")
	assert.Contains(t, err.Error(), "invalid platform")
}

func TestRun_ClientError(t *testing.T) {
	originalCreateClient := createPromotionClientFunc
	createPromotionClientFunc = func(baseURL, token string) promotionClient {
		return &mockFailingPromotionClient{}
	}
	defer func() { createPromotionClientFunc = originalCreateClient }()

	setPromotionEnv(t)

	err := run(nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to promote java version 1.2.3 to REGULAR")
}
