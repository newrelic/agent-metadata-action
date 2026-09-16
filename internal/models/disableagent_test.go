package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDisableAgentRequest_Marshal(t *testing.T) {
	req := DisableAgentRequest{Metadata: DisableAgentMetadata{Enabled: false}}

	body, err := json.Marshal(req)

	require.NoError(t, err)
	assert.JSONEq(t, `{"metadata":{"enabled":false}}`, string(body))
}
