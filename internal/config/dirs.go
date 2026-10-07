package config

import (
	"os"
	"path/filepath"
	"strings"
)

// GetRootFolderForAgentRepo loads the root folder where configuration info is stored
// Returns the configured directory or defaults to ".fleetControl"
func GetRootFolderForAgentRepo() string {
	configDir := GetConfigDirectory()
	if configDir == "" {
		return ".fleetControl"
	}
	return strings.TrimSpace(configDir)
}

// CandidateRootFoldersForAgentRepo returns, in preference order, the directory names
// tried when no explicit config-directory override is set.
func CandidateRootFoldersForAgentRepo() []string {
	return []string{".fleetControl", ".nrcontrol"}
}

// ResolveRootFolderForAgentRepo determines the single root control directory to use for
// this run. If the config-directory input is explicitly set, that value is returned as-is
// (no filesystem check, no fallback). Otherwise it checks each candidate from
// CandidateRootFoldersForAgentRepo, in order, against workspacePath and returns the first
// one that exists on disk. If none exist, it returns the default (".fleetControl") so
// downstream existence-check error messages stay anchored to a sensible name.
func ResolveRootFolderForAgentRepo(workspacePath string) string {
	if GetConfigDirectory() != "" {
		return GetRootFolderForAgentRepo()
	}
	for _, candidate := range CandidateRootFoldersForAgentRepo() {
		if dirExists(filepath.Join(workspacePath, candidate)) {
			return candidate
		}
	}
	return GetRootFolderForAgentRepo()
}

// ResolveFilenameInDir returns which of two filename candidates exists inside dir,
// preferring primaryFilename (camelCase) over fallbackFilename (kebab-case) when both
// exist. If neither exists, it returns primaryFilename so the caller's eventual "file not
// found" error still references the expected default name.
func ResolveFilenameInDir(dir, primaryFilename, fallbackFilename string) string {
	if fileExists(filepath.Join(dir, primaryFilename)) {
		return primaryFilename
	}
	if fileExists(filepath.Join(dir, fallbackFilename)) {
		return fallbackFilename
	}
	return primaryFilename
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func GetConfigurationDefinitionsFilename() string {
	return "configurationDefinitions.yml"
}

func GetConfigurationDefinitionsFilenameFallback() string {
	return "configuration-definitions.yml"
}

func GetAgentControlDefinitionsFilename() string {
	return "agentControlDefinitions.yml"
}

func GetAgentControlDefinitionsFilenameFallback() string {
	return "agent-control-definitions.yml"
}

// GetAgentDefinitionFilename returns the filename of the optional agent definition file.
func GetAgentDefinitionFilename() string {
	return "agentDefinition.yml"
}

func GetAgentDefinitionFilenameFallback() string {
	return "agent-definition.yml"
}

func GetReleaseNotesDirectory() string {
	return "src/content/docs/release-notes"
}
