package config

import (
	"os"
	"testing"
)

func TestGetRootFolderForAgentRepo(t *testing.T) {
	tests := []struct {
		name        string
		setupFunc   func(t *testing.T)
		expected    string
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
