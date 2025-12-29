package parser

import (
	"fmt"
	"strings"

	"agent-metadata-action/internal/fileutil"

	"gopkg.in/yaml.v3"
)

// MDXFrontmatter represents the YAML frontmatter in an MDX file
type MDXFrontmatter struct {
	Subject                   string   `yaml:"subject"`
	ReleaseDate               string   `yaml:"releaseDate"`
	Version                   string   `yaml:"version"`
	MetaDescription           string   `yaml:"metaDescription"`
	Features                  []string `yaml:"features"`
	Bugs                      []string `yaml:"bugs"`
	Security                  []string `yaml:"security"`
	Deprecations              []string `yaml:"deprecations"`
	SupportedOperatingSystems []string `yaml:"supportedOperatingSystems"`
	EOL                       string   `yaml:"eol"`
}

type Subject string

const (
	DotNet   Subject = ".NET agent"
	Infra    Subject = "Infrastructure agent"
	InfraK8s Subject = "Kubernetes integration"
	Java     Subject = "Java agent"
	Node     Subject = "Node.js agent"
	NRDot    Subject = "NRDOT"
	Python   Subject = "Python agent"
	Ruby     Subject = "Ruby agent"
)

var SubjectToAgentTypeMapping = map[Subject]string{
	DotNet:   "DotnetAgent",
	Infra:    "InfrastructureAgent",
	InfraK8s: "InfrastructureK8sAgent",
	Java:     "JavaAgent",
	Node:     "NodeAgent",
	NRDot:    "NrdotAgent",
	Python:   "PythonAgent",
	Ruby:     "RubyAgent",
}

// ParseMDXFile reads an MDX file and extracts the YAML frontmatter
func ParseMDXFile(filePath string) (*MDXFrontmatter, error) {
	data, err := fileutil.ReadFileSafe(filePath, fileutil.MaxMDXFileSize)
	if err != nil {
		return nil, fmt.Errorf("failed to read MDX file: %w", err)
	}

	content := string(data)

	// Extract frontmatter between --- markers
	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("MDX file does not start with frontmatter delimiter")
	}

	// Find the closing --- delimiter
	endIndex := strings.Index(content[4:], "\n---")
	if endIndex == -1 {
		return nil, fmt.Errorf("MDX file missing closing frontmatter delimiter")
	}

	// Extract YAML content (skip first "---\n" and before second "---")
	yamlContent := content[4 : 4+endIndex]

	var frontmatter MDXFrontmatter
	if err := yaml.Unmarshal([]byte(yamlContent), &frontmatter); err != nil {
		return nil, fmt.Errorf("failed to parse YAML frontmatter: %w", err)
	}

	return &frontmatter, nil
}
