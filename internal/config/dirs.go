package config

import "path/filepath"

// GetRootFolderForAgentRepo loads the root folder where configuration info is st
func GetRootFolderForAgentRepo() string {
	return ".fleetControl"
}

// GetConfigurationDefinitionsFilepath loads the root folder where configuration info is st
func GetConfigurationDefinitionsFilepath() string {
	return filepath.Join(GetRootFolderForAgentRepo(), GetConfigurationDefinitionsFilename())
}

func GetConfigurationDefinitionsFilename() string {
	return "configurationDefinitions.yml"
}

// GetAgentControlFolderForAgentRepo loads the folder holding the agent control definitions
func GetAgentControlFolderForAgentRepo() string {
	return filepath.Join(GetRootFolderForAgentRepo(), "agentControl")
}

func GetReleaseNotesDirectory() string {
	return "src/content/docs/release-notes"
}
