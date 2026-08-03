//go:build e2e

package validator

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAgentTypeDefinition_Failure(t *testing.T) {
	invalidDefinition := `name: my-broken-agent-type-definition`

	absPath := filepath.Join(t.TempDir(), "invalid-agent-type.yaml")
	require.NoError(t, os.WriteFile(absPath, []byte(invalidDefinition), 0644))

	err := ValidateAgentTypeDefinitionFunc(context.Background(), absPath)
	require.Error(t, err, "expected newrelic-agent-control-cli to reject this definition")

	var valErr *ValidationError
	require.True(t, errors.As(err, &valErr), "expected *ValidationError, got %T: %v", err, err)

	assert.NotEmpty(t, valErr.Output)
	t.Logf("CLI rejection output:\n%s", valErr.Output)
}

func TestValidateAgentTypeDefinition_Success(t *testing.T) {
	validDefinition := `namespace: fake_namespace
name: fake_name
version: "1.0.0"
platform: host
operating_system: linux
protocol_version: "1.0"
variables:
  fake-var:
    description: "fake description"
    type: string
    required: true
deployment:
  health:
    interval: 30s
    initial_delay: 30s
    checks:
      - kind: Process
  executables:
    - id: my_agent
      path: /usr/bin/my_agent
      args:
        - "--fake"
        - ${nr-var:fake-var}
`

	absPath := filepath.Join(t.TempDir(), "valid-agent-type.yaml")
	require.NoError(t, os.WriteFile(absPath, []byte(validDefinition), 0644))

	err := ValidateAgentTypeDefinitionFunc(context.Background(), absPath)
	assert.NoError(t, err, "expected newrelic-agent-control-cli to accept this definition")
}
