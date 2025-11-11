package models

// AgentMetadata represents the complete agent metadata structure
type AgentMetadata struct {
	Schema        string     `json:"schema"`
	Configuration ConfigJson `json:"configuration"`
	Metadata      string     `json:"metadata"`
}

// ConfigYaml represents the YAML configuration structure
type ConfigYaml struct {
	Name        string `yaml:"name"`
	Slug        string `yaml:"slug"`
	Platform    string `yaml:"platform"`
	Description string `yaml:"description"`
	Type        string `yaml:"type"`
	Version     string `yaml:"version"`
	Schema      string `yaml:"schema"`
}

// ConfigJson represents the JSON configuration structure (without schema)
type ConfigJson struct {
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Platform    string `json:"platform"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Version     string `json:"version"`
}

// ConfigFile represents the YAML file structure containing multiple configs
type ConfigFile struct {
	Configs []ConfigYaml `yaml:"configs"`
}

// ConvertToConfigJson converts ConfigYaml array to ConfigJson array
func ConvertToConfigJson(yamlConfigs []ConfigYaml) []ConfigJson {
	configs := make([]ConfigJson, 0, len(yamlConfigs))
	for _, c := range yamlConfigs {
		configs = append(configs, ConfigJson{
			Name:        c.Name,
			Slug:        c.Slug,
			Platform:    c.Platform,
			Description: c.Description,
			Type:        c.Type,
			Version:     c.Version,
		})
	}
	return configs
}
