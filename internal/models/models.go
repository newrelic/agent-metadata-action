package models

// AgentMetadata represents the complete agent metadata structure
type AgentMetadata struct {
	ConfigurationDefinitions []ConfigurationDefinition `json:"configurationDefinitions"`
	Metadata                 string                    `json:"metadata"`
}

// ConfigurationDefinition represents a configuration that can be read from YAML and sent as JSON
type ConfigurationDefinition struct {
	Name        string `yaml:"name" json:"name"`
	Slug        string `yaml:"slug" json:"slug"`
	Platform    string `yaml:"platform" json:"platform"`
	Description string `yaml:"description" json:"description"`
	Type        string `yaml:"type" json:"type"`
	Version     string `yaml:"version" json:"version"`
	Format      string `yaml:"format" json:"format"`
	Schema      string `yaml:"schema" json:"schema"`
}

// ConfigFile represents the YAML file structure containing multiple configs
type ConfigFile struct {
	Configs []ConfigurationDefinition `yaml:"configurationDefinitions"`
}
