package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"agent-metadata-action/internal/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateAgentType_Success(t *testing.T) {
	t.Setenv("INPUT_AGENT_TYPE", "java")

	err := validateAgentType()
	assert.NoError(t, err)
}

func TestValidateAgentType_NotSet(t *testing.T) {
	t.Setenv("INPUT_AGENT_TYPE", "")

	err := validateAgentType()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent-type is required")
	assert.Contains(t, err.Error(), "INPUT_AGENT_TYPE not set")
}

func TestPrintJSON_ValidData(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create test data
	metadata := models.Metadata{
		Version:  "1.2.3",
		Features: []string{"Feature A", "Feature B"},
		Bugs:     []string{"Bug fix 1"},
	}

	// Call printJSON
	printJSON("Test Metadata", metadata)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output format
	assert.Contains(t, output, "::debug::Test Metadata:")
	assert.Contains(t, output, "\"version\": \"1.2.3\"")
	assert.Contains(t, output, "\"features\":")
	assert.Contains(t, output, "Feature A")
	assert.Contains(t, output, "Feature B")
	assert.Contains(t, output, "Bug fix 1")

	// Verify it's valid JSON by unmarshaling
	// Extract JSON from the debug line
	jsonStart := strings.Index(output, "{")
	if jsonStart != -1 {
		jsonStr := output[jsonStart:]
		jsonStr = strings.TrimSpace(jsonStr)
		var result models.Metadata
		err := json.Unmarshal([]byte(jsonStr), &result)
		assert.NoError(t, err)
		assert.Equal(t, "1.2.3", result.Version)
	}
}

func TestPrintJSON_EmptyData(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Create empty metadata
	metadata := models.Metadata{
		Version: "1.0.0",
	}

	// Call printJSON
	printJSON("Empty Metadata", metadata)

	// Restore stdout and read captured output
	w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	io.Copy(&buf, r)
	output := buf.String()

	// Verify output
	assert.Contains(t, output, "::debug::Empty Metadata:")
	assert.Contains(t, output, "\"version\": \"1.0.0\"")
}

func TestRun_AgentRepoFlow(t *testing.T) {
	// Get project root
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	// Set environment variables
	t.Setenv("INPUT_AGENT_TYPE", "java")
	t.Setenv("INPUT_VERSION", "1.2.3")
	t.Setenv("GITHUB_WORKSPACE", workspace)

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Call run
	err = run()

	// Restore stdout/stderr and read captured output
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	outputStr := bufOut.String()
	stderrStr := bufErr.String()

	// Verify no error
	require.NoError(t, err)

	// Verify output
	assert.Contains(t, outputStr, "::debug::Agent version: 1.2.3")
	assert.Contains(t, outputStr, "::debug::Reading config from workspace:")
	assert.Contains(t, outputStr, "::notice::Successfully read configs file")
	assert.Contains(t, outputStr, "::debug::Agent Metadata:")
	assert.Contains(t, outputStr, "\"configurationDefinitions\":")
	assert.Contains(t, outputStr, "\"metadata\":")
	assert.Contains(t, outputStr, "\"agentControl\":")

	// Stderr may contain debug messages (e.g., about no PR context) but not errors
	if stderrStr != "" {
		assert.NotContains(t, stderrStr, "::error::")
		t.Logf("Stderr: %s", stderrStr)
	}
}

func TestRun_DocsFlow(t *testing.T) {
	// Get project root
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	workspace := filepath.Join(projectRoot, "integration-test", "docs-flow")

	// Set environment variables (workspace with no .fleetControl)
	t.Setenv("INPUT_AGENT_TYPE", "java")
	t.Setenv("INPUT_VERSION", "2.0.0")
	t.Setenv("GITHUB_WORKSPACE", workspace)

	// Capture stdout and stderr
	oldStdout := os.Stdout
	oldStderr := os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout = wOut
	os.Stderr = wErr

	// Call run
	err = run()

	// Restore stdout/stderr and read captured output
	wOut.Close()
	wErr.Close()
	os.Stdout = oldStdout
	os.Stderr = oldStderr

	var bufOut, bufErr bytes.Buffer
	io.Copy(&bufOut, rOut)
	io.Copy(&bufErr, rErr)

	outputStr := bufOut.String()
	stderrStr := bufErr.String()

	// Verify no error
	require.NoError(t, err)

	// Verify output
	assert.Contains(t, outputStr, "::debug::Agent version: 2.0.0")
	assert.Contains(t, outputStr, "::notice::Running in metadata-only mode")
	assert.Contains(t, outputStr, "::debug::Metadata:")
	assert.Contains(t, outputStr, "\"version\": \"2.0.0\"")

	// Should NOT contain agent metadata fields
	assert.NotContains(t, outputStr, "\"configurationDefinitions\":")
	assert.NotContains(t, outputStr, "\"agentControl\":")

	// Stderr may have debug message about no PR context
	t.Logf("Stderr: %s", stderrStr)
}

func TestRun_MissingAgentType(t *testing.T) {
	// Don't set INPUT_AGENT_TYPE
	t.Setenv("INPUT_VERSION", "1.0.0")

	err := run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "agent-type is required")
}

func TestRun_MissingVersion(t *testing.T) {
	// Get project root
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	workspace := filepath.Join(projectRoot, "integration-test", "docs-flow")

	// Don't set INPUT_VERSION
	t.Setenv("INPUT_AGENT_TYPE", "java")
	t.Setenv("GITHUB_WORKSPACE", workspace)

	err = run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Error loading metadata")
	assert.Contains(t, err.Error(), "unable to determine version")
}

func TestRun_InvalidWorkspace(t *testing.T) {
	// Set invalid workspace
	t.Setenv("INPUT_AGENT_TYPE", "java")
	t.Setenv("INPUT_VERSION", "1.0.0")
	t.Setenv("GITHUB_WORKSPACE", "/nonexistent/path")

	err := run()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Error reading configs")
}

// Integration test that runs the actual binary
func TestMain_AgentRepoFlow(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agent-metadata-action-test")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = filepath.Join(".") // Current directory is cmd/agent-metadata-action
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build binary")

	// Get project root (two levels up from cmd/agent-metadata-action)
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	workspace := filepath.Join(projectRoot, "integration-test", "agent-flow")

	// Run the binary with environment variables
	cmd := exec.Command(binaryPath)
	cmd.Env = []string{
		"INPUT_AGENT_TYPE=java",
		"INPUT_VERSION=1.2.3",
		"GITHUB_WORKSPACE=" + workspace,
	}

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Binary execution failed: %s", string(output))

	outputStr := string(output)

	// Verify output contains expected debug messages
	assert.Contains(t, outputStr, "::debug::Agent version: 1.2.3")
	assert.Contains(t, outputStr, "::debug::Reading config from workspace:")
	assert.Contains(t, outputStr, "::notice::Successfully read configs file")
	assert.Contains(t, outputStr, "::debug::Found")
	assert.Contains(t, outputStr, "::debug::Agent Metadata:")

	// Verify JSON structure is present
	assert.Contains(t, outputStr, "\"configurationDefinitions\":")
	assert.Contains(t, outputStr, "\"metadata\":")
	assert.Contains(t, outputStr, "\"agentControl\":")
}

func TestMain_DocsFlow(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agent-metadata-action-test")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = filepath.Join(".") // Current directory is cmd/agent-metadata-action
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build binary")

	// Get project root (two levels up from cmd/agent-metadata-action)
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	workspace := filepath.Join(projectRoot, "integration-test", "docs-flow")

	// Run with docs-flow workspace (no .fleetControl)
	cmd := exec.Command(binaryPath)
	cmd.Env = []string{
		"INPUT_AGENT_TYPE=java",
		"INPUT_VERSION=2.0.0",
		"GITHUB_WORKSPACE=" + workspace,
	}

	output, err := cmd.CombinedOutput()
	require.NoError(t, err, "Binary execution failed: %s", string(output))

	outputStr := string(output)

	// Verify output contains expected messages for docs flow
	assert.Contains(t, outputStr, "::debug::Agent version: 2.0.0")
	assert.Contains(t, outputStr, "::notice::Running in metadata-only mode")
	assert.Contains(t, outputStr, "::debug::Metadata:")

	// Verify it's metadata only (not full agent metadata)
	assert.Contains(t, outputStr, "\"version\": \"2.0.0\"")
	// Should NOT contain agent metadata fields
	assert.NotContains(t, outputStr, "\"configurationDefinitions\":")
	assert.NotContains(t, outputStr, "\"agentControl\":")
}

func TestMain_MissingAgentType(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agent-metadata-action-test")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = filepath.Join(".") // Current directory is cmd/agent-metadata-action
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build binary")

	// Run without INPUT_AGENT_TYPE
	cmd := exec.Command(binaryPath)
	cmd.Env = []string{
		"INPUT_VERSION=1.0.0",
	}

	output, err := cmd.CombinedOutput()
	assert.Error(t, err, "Expected binary to exit with error")

	outputStr := string(output)
	assert.Contains(t, outputStr, "::error::agent-type is required")
	assert.Contains(t, outputStr, "INPUT_AGENT_TYPE not set")
}

func TestMain_MissingVersion(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agent-metadata-action-test")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = filepath.Join(".") // Current directory is cmd/agent-metadata-action
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build binary")

	// Get project root (two levels up from cmd/agent-metadata-action)
	projectRoot, err := filepath.Abs("../..")
	require.NoError(t, err)

	workspace := filepath.Join(projectRoot, "integration-test", "docs-flow")

	// Run without INPUT_VERSION
	cmd := exec.Command(binaryPath)
	cmd.Env = []string{
		"INPUT_AGENT_TYPE=java",
		"GITHUB_WORKSPACE=" + workspace,
	}

	output, err := cmd.CombinedOutput()
	assert.Error(t, err, "Expected binary to exit with error")

	outputStr := string(output)
	assert.Contains(t, outputStr, "::error::Error loading metadata:")
	assert.Contains(t, outputStr, "unable to determine version")
}

func TestMain_InvalidWorkspace(t *testing.T) {
	// Build the binary
	tmpDir := t.TempDir()
	binaryPath := filepath.Join(tmpDir, "agent-metadata-action-test")

	buildCmd := exec.Command("go", "build", "-o", binaryPath, ".")
	buildCmd.Dir = filepath.Join(".") // Current directory is cmd/agent-metadata-action
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build binary")

	// Run with invalid workspace path
	cmd := exec.Command(binaryPath)
	cmd.Env = []string{
		"INPUT_AGENT_TYPE=java",
		"INPUT_VERSION=1.0.0",
		"GITHUB_WORKSPACE=/nonexistent/path",
	}

	output, err := cmd.CombinedOutput()
	assert.Error(t, err, "Expected binary to exit with error")

	outputStr := string(output)
	assert.Contains(t, outputStr, "::error::Error reading configs:")
}
