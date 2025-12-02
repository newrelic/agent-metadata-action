package parser

import (
	"fmt"
	"os"
	"strings"

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

// ParseMDXFile reads an MDX file and extracts the YAML frontmatter
func ParseMDXFile(filePath string) (*MDXFrontmatter, error) {
	data, err := os.ReadFile(filePath)
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

// ParseMDXFiles parses multiple MDX files and aggregates their metadata
func ParseMDXFiles(filePaths []string, workspace string) (features, bugs, security, deprecations, supportedOperatingSystems []string, eol string, err error) {
	featuresMap := make(map[string]bool)
	bugsMap := make(map[string]bool)
	securityMap := make(map[string]bool)
	deprecationsMap := make(map[string]bool)
	supportedOSMap := make(map[string]bool)

	for _, filePath := range filePaths {
		// Construct full path
		fullPath := filePath
		if workspace != "" {
			fullPath = workspace + "/" + filePath
		}

		frontmatter, err := ParseMDXFile(fullPath)
		if err != nil {
			return nil, nil, nil, nil, nil, "", fmt.Errorf("failed to parse %s: %w", filePath, err)
		}

		// Aggregate features
		for _, feature := range frontmatter.Features {
			if feature != "" {
				featuresMap[feature] = true
			}
		}

		// Aggregate bugs
		for _, bug := range frontmatter.Bugs {
			if bug != "" {
				bugsMap[bug] = true
			}
		}

		// Aggregate security
		for _, sec := range frontmatter.Security {
			if sec != "" {
				securityMap[sec] = true
			}
		}

		// Aggregate deprecations
		for _, deprecation := range frontmatter.Deprecations {
			if deprecation != "" {
				deprecationsMap[deprecation] = true
			}
		}

		// Aggregate supported operating systems
		for _, suppOS := range frontmatter.SupportedOperatingSystems {
			if suppOS != "" {
				supportedOSMap[suppOS] = true
			}
		}

		// Use the last EOL date found (could be modified to use earliest/latest)
		if frontmatter.EOL != "" {
			eol = frontmatter.EOL
		}
	}

	// Convert maps to slices
	for feature := range featuresMap {
		features = append(features, feature)
	}
	for bug := range bugsMap {
		bugs = append(bugs, bug)
	}
	for sec := range securityMap {
		security = append(security, sec)
	}
	for deprecation := range deprecationsMap {
		deprecations = append(deprecations, deprecation)
	}
	for currOS := range supportedOSMap {
		supportedOperatingSystems = append(supportedOperatingSystems, currOS)
	}

	return features, bugs, security, deprecations, supportedOperatingSystems, eol, nil
}
