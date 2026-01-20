package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"agent-metadata-action/internal/config"
	"agent-metadata-action/internal/logging"
)

const ReleaseNotesFileExtension = ".mdx"

var IgnoredFilenames = []string{"index.mdx"}

// gitSHARegex validates Git SHA-1 hashes (40 hexadecimal characters)
var gitSHARegex = regexp.MustCompile(`^[0-9a-f]{40}$`)

// PushEvent represents the GitHub PR event payload
type PushEvent struct {
	Before string `json:"before"`
	After  string `json:"after"`
	Ref    string `json:"ref"`
}

// GetChangedMDXFiles returns ReleaseNotesFileExtension type files changed in the PR under the expected release notes direcotry, excluding IgnoredFilenames
func GetChangedMDXFiles() ([]string, error) {
	return GetChangedMDXFilesFunc(context.Background())
}

// isIgnoredFilename checks if the filename should be ignored
func isIgnoredFilename(filename string) bool {
	for _, ignored := range IgnoredFilenames {
		if filename == ignored {
			return true
		}
	}
	return false
}

// isValidGitSHA validates that a string is a valid Git SHA-1 hash
// Git SHA-1 hashes are exactly 40 hexadecimal characters
func isValidGitSHA(sha string) bool {
	return gitSHARegex.MatchString(sha)
}

// GetChangedMDXFilesFunc is a variable that holds the function to get changed MDX files
// This allows tests to override the implementation
var GetChangedMDXFilesFunc = getChangedMDXFilesImpl

// getChangedMDXFilesImpl is the actual implementation
func getChangedMDXFilesImpl(ctx context.Context) ([]string, error) {
	eventPath := config.GetEventPath()
	if eventPath == "" {
		return nil, fmt.Errorf("GITHUB_EVENT_PATH not set")
	}
	logging.Debugf(ctx, "GH event path: %s", eventPath)

	data, err := os.ReadFile(eventPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read event payload: %w", err)
	}

	var event PushEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, fmt.Errorf("failed to parse event payload: %w", err)
	}

	logging.Debugf(ctx, "event payload %s", event)
	logging.Debugf(ctx, "GH branch name: %s", event.Ref)
	logging.Debugf(ctx, "GH SHAs: before %s and after %s", event.Before, event.After)

	// Validate SHAs to prevent command injection
	if !isValidGitSHA(event.Before) {
		return nil, fmt.Errorf("invalid before SHA format: must be 40 hexadecimal characters")
	}
	if !isValidGitSHA(event.After) {
		return nil, fmt.Errorf("invalid after SHA format: must be 40 hexadecimal characters")
	}

	cmd := exec.Command("git", "diff", "--diff-filter=ACMR", "--name-only",
		fmt.Sprintf("%s...%s", event.Before, event.After))

	// Set working directory to GITHUB_WORKSPACE so git can find the repository
	workspace := config.GetWorkspace()
	if workspace != "" {
		cmd.Dir = workspace
	}
	logging.Debugf(ctx, "workspace: %s", workspace)

	var out bytes.Buffer
	cmd.Stdout = &out

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	logging.Debugf(ctx, "git diff output:\n%s", out.String())

	var mdxFiles []string
	for _, line := range strings.Split(out.String(), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasSuffix(line, ReleaseNotesFileExtension) {
			continue
		}
		if isIgnoredFilename(filepath.Base(line)) {
			continue
		}
		if strings.Contains(line, config.GetReleaseNotesDirectory()) {
			// Convert to absolute path if workspace is set
			if workspace != "" {
				line = filepath.Join(workspace, line)
			}
			logging.Debugf(ctx, "mdx append line: %s", line)
			mdxFiles = append(mdxFiles, line)
		}
	}

	return mdxFiles, nil
}
