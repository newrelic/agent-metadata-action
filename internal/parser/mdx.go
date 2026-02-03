package parser

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// MDXFrontmatter represents the YAML frontmatter in an MDX file.
// It uses a map to allow any attributes to be added or removed without code changes.
type MDXFrontmatter map[string]interface{}

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
	//EBPF   Subject = "" @todo update once eBPF is publishing release notes
)

// SubjectToAgentTypeMapping list based on these: https://source.datanerd.us/fleet-management/fleet-deployment-api/blob/master/src/main/java/com/newrelic/fleetdeploymentapi/constant/AgentConstants.java
var SubjectToAgentTypeMapping = map[Subject]string{
	DotNet:   "NRDotNetAgent",
	Infra:    "NRInfra", // maps to the same agent type for k8s & host
	InfraK8s: "NRInfra", // maps to the same agent type for k8s & host
	Java:     "NRJavaAgent",
	Node:     "NRNodeAgent",
	NRDot:    "NRDOT",
	Python:   "NRPythonAgent",
	Ruby:     "NRRubyAgent",
	// EBPF: "NReBPFAgent" @todo update once eBPF is publishing release notes
}

// ParseMDXFile reads an MDX file and extracts the YAML frontmatter
func ParseMDXFile(filePath string) (MDXFrontmatter, error) {
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

	return frontmatter, nil
}
