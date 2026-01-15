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

// GetAgentControlDefinitionsFilepath returns the path to the agentControlDefinitions.yml file
func GetAgentControlDefinitionsFilepath() string {
	return filepath.Join(GetRootFolderForAgentRepo(), GetAgentControlDefinitionsFilename())
}

func GetAgentControlDefinitionsFilename() string {
	return "agentControlDefinitions.yml"
}

func GetReleaseNotesDirectory() string {
	return "src/content/docs/release-notes"
}
