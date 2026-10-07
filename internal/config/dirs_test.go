package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetRootFolderForAgentRepo(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T)
		expected  string
	}{
		{
			name: "returns explicit override when INPUT_CONFIG_DIRECTORY is set",
			setupFunc: func(t *testing.T) {
				if err := os.Setenv("INPUT_CONFIG_DIRECTORY", ".custom"); err != nil {
					t.Fatalf("failed to set env: %v", err)
				}
				t.Cleanup(func() {
					os.Unsetenv("INPUT_CONFIG_DIRECTORY")
				})
			},
			expected: ".custom",
		},
		{
			name: "returns .fleetControl as default when no override",
			setupFunc: func(t *testing.T) {
				os.Unsetenv("INPUT_CONFIG_DIRECTORY")
			},
			expected: ".fleetControl",
		},
		{
			name: "trims whitespace from override values",
			setupFunc: func(t *testing.T) {
				if err := os.Setenv("INPUT_CONFIG_DIRECTORY", "  .custom  "); err != nil {
					t.Fatalf("failed to set env: %v", err)
				}
				t.Cleanup(func() {
					os.Unsetenv("INPUT_CONFIG_DIRECTORY")
				})
			},
			expected: ".custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setupFunc(t)
			got := GetRootFolderForAgentRepo()
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestResolveRootFolderForAgentRepo(t *testing.T) {
	tests := []struct {
		name       string
		override   string
		candidates []string
		expected   string
	}{
		{
			name:       "explicit override skips fallback even when both candidates exist",
			override:   ".custom",
			candidates: []string{".fleetControl", ".nrcontrol"},
			expected:   ".custom",
		},
		{
			name:       "no override, only .fleetControl exists",
			candidates: []string{".fleetControl"},
			expected:   ".fleetControl",
		},
		{
			name:       "no override, only .nrcontrol exists",
			candidates: []string{".nrcontrol"},
			expected:   ".nrcontrol",
		},
		{
			name:       "no override, both exist, .fleetControl takes precedence",
			candidates: []string{".fleetControl", ".nrcontrol"},
			expected:   ".fleetControl",
		},
		{
			name:       "no override, neither exists, defaults to .fleetControl",
			candidates: nil,
			expected:   ".fleetControl",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.override != "" {
				t.Setenv("INPUT_CONFIG_DIRECTORY", tt.override)
			} else {
				os.Unsetenv("INPUT_CONFIG_DIRECTORY")
			}

			workspace := t.TempDir()
			for _, candidate := range tt.candidates {
				if err := os.MkdirAll(filepath.Join(workspace, candidate), 0o755); err != nil {
					t.Fatalf("failed to create candidate dir: %v", err)
				}
			}

			got := ResolveRootFolderForAgentRepo(workspace)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestResolveFilenameInDir(t *testing.T) {
	const primary = "configurationDefinitions.yml"
	const fallback = "configuration-definitions.yml"

	tests := []struct {
		name           string
		createPrimary  bool
		createFallback bool
		expectedResult string
	}{
		{
			name:           "only primary exists",
			createPrimary:  true,
			expectedResult: primary,
		},
		{
			name:           "only fallback exists",
			createFallback: true,
			expectedResult: fallback,
		},
		{
			name:           "both exist, primary wins",
			createPrimary:  true,
			createFallback: true,
			expectedResult: primary,
		},
		{
			name:           "neither exists, defaults to primary",
			expectedResult: primary,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.createPrimary {
				if err := os.WriteFile(filepath.Join(dir, primary), []byte("x"), 0o644); err != nil {
					t.Fatalf("failed to create primary file: %v", err)
				}
			}
			if tt.createFallback {
				if err := os.WriteFile(filepath.Join(dir, fallback), []byte("x"), 0o644); err != nil {
					t.Fatalf("failed to create fallback file: %v", err)
				}
			}

			got := ResolveFilenameInDir(dir, primary, fallback)
			if got != tt.expectedResult {
				t.Errorf("expected %q, got %q", tt.expectedResult, got)
			}
		})
	}
}
