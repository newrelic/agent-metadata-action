// Package validator validates agent type definition YAML files by shelling out to
// newrelic-agent-control-cli (distributed as a Docker image) before that data is
// sent to the instrumentation service.
package validator

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"agent-metadata-action/internal/config"
)

// CLIImageName is the newrelic-agent-control-cli Docker image, without a tag.
const CLIImageName = "docker.io/newrelic/newrelic-agent-control-cli"

// DefaultCLIImageTag is the tag used when INPUT_AGENT_CONTROL_CLI_TAG is not set.
const DefaultCLIImageTag = "latest"

// ValidationError indicates that newrelic-agent-control-cli rejected an agent type
// definition file. Callers can distinguish this from unrelated loader errors (e.g.
// a missing file) via errors.As, since only this case should abort the action.
type ValidationError struct {
	Path   string
	Output string
	Err    error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("agent type definition at %s failed validation: %v\n%s", e.Path, e.Err, e.Output)
}

func (e *ValidationError) Unwrap() error {
	return e.Err
}

// ValidateAgentTypeDefinitionFunc validates the agent type definition YAML file at
// absFilePath. It is a variable so tests can override the implementation without
// invoking Docker.
var ValidateAgentTypeDefinitionFunc = validateAgentTypeDefinitionImpl

func validateAgentTypeDefinitionImpl(ctx context.Context, absFilePath string) error {
	dir, file := filepath.Split(absFilePath)

	tag := config.GetAgentControlCLIImageTag()
	if tag == "" {
		tag = DefaultCLIImageTag
	}
	image := fmt.Sprintf("%s:%s", CLIImageName, tag)

	cmd := exec.CommandContext(ctx, "docker", "run", "--rm", "--pull", "always",
		"-v", fmt.Sprintf("%s:/data:ro", dir),
		image, "agent-type", "validate", "--file", "/data/"+file)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return &ValidationError{Path: absFilePath, Output: string(out), Err: err}
	}

	return nil
}
