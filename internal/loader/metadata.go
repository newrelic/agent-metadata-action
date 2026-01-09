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
		"version": version,
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
		return nil, fmt.Errorf("could not get changed files -- %s", err)
	} else if len(changedFilepaths) > 0 {
		var metadataForDocs []MetadataForDocs
		for _, filepath := range changedFilepaths {
			frontMatter, err := parser.ParseMDXFile(filepath)
			if err != nil {
				fmt.Printf("::warn::Failed to parse MDX file %s %s - skipping\n", filepath, err)
				continue
			}

			if frontMatter["version"] == "" {
				fmt.Printf("::warn::Version is required in metadata for file %s - skipping\n", filepath)
				continue
			}

			if frontMatter["subject"] == nil || frontMatter["subject"] == "" {
				fmt.Printf("::warn::Subject (to derive agent type) is required in metadata for file %s - skipping\n", filepath)
				continue
			}
			agentType := parser.SubjectToAgentTypeMapping[parser.Subject(frontMatter["subject"].(string))]

			// Convert frontMatter directly to Metadata (both are maps)
			metadata := models.Metadata(frontMatter)

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
	} else {
		fmt.Print("::debug::no changed files detected in the PR context\n")
		return nil, nil
	}
}
