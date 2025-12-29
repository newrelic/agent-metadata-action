package loader

import (
	"agent-metadata-action/internal/github"
	"agent-metadata-action/internal/models"
	"agent-metadata-action/internal/parser"
	"fmt"
	"os"
)

// LoadMetadataForAgents loads metadata with only version populated
func LoadMetadataForAgents(version string) models.Metadata {
	return models.Metadata{
		Version: version,
	}
}

type MetadataForDocs struct {
	AgentType             string
	AgentMetadataFromDocs models.Metadata
}

// LoadMetadataForDocs loads metadata from changed MDX files in a PR
// Loads as many files as it can and warns on issues with certain files
func LoadMetadataForDocs() ([]MetadataForDocs, error) {
	filesProcessed := 0

	// Get changed MDX files (for PR context)
	changedFilepaths, err := github.GetChangedMDXFiles()
	if err != nil {
		return nil, fmt.Errorf("could not get changed files")
	} else if len(changedFilepaths) > 0 {
		var metadataForDocs []MetadataForDocs
		for _, filepath := range changedFilepaths {
			frontMatter, err := parser.ParseMDXFile(filepath)
			if err != nil {
				fmt.Printf("::warn::Failed to parse MDX file %s %s - skipping ", filepath, err)
				continue
			}

			if frontMatter.Version == "" {
				fmt.Printf("::warn::Version is required in metadata for file %s - skipping ", filepath)
				continue
			}

			agentType := parser.SubjectToAgentTypeMapping[parser.Subject(frontMatter.Subject)]

			if agentType == "" {
				fmt.Printf("::warn::Subject (to derive agent type) is required in metadata for file %s - skipping ", filepath)
				continue
			}

			metadata := models.Metadata{
				Version:                   frontMatter.Version,
				Features:                  frontMatter.Features,
				Bugs:                      frontMatter.Bugs,
				Security:                  frontMatter.Security,
				Deprecations:              frontMatter.Deprecations,
				SupportedOperatingSystems: frontMatter.SupportedOperatingSystems,
				EOL:                       frontMatter.EOL,
			}

			metadataForDocs = append(metadataForDocs, MetadataForDocs{
				AgentType:             agentType,
				AgentMetadataFromDocs: metadata,
			})

			filesProcessed++
		}

		if filesProcessed == 0 {
			return nil, fmt.Errorf("unable to load metadata for any of the %d changed MDX files", len(changedFilepaths))
		} else {
			fmt.Fprintf(os.Stderr, "::notice::Loaded metadata for %d out of %d changed MDX files\n", filesProcessed, len(changedFilepaths))
		}

		return metadataForDocs, nil
	}
	return nil, fmt.Errorf("unknown error in LoadMetadataForDocs")
}
