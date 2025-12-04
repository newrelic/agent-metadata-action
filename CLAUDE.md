# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a GitHub Action written in Go that reads agent configuration metadata from a repository. The action automatically checks out the calling repository at a specified version tag, then reads configuration from `.fleetControl/configurationDefinitions.yml` and metadata from changed MDX files (in PR context). The action supports two workflows:

1. **Agent Repository Workflow**: When `.fleetControl` directory exists, reads full configuration definitions and agent control files
2. **Documentation Workflow**: When `.fleetControl` doesn't exist, reads only metadata from MDX files

**Future Direction**: This action will eventually use the data read in to call a service.

## Project Structure

The project follows standard Go conventions:

```
├── cmd/
│   └── agent-metadata-action/    # Main application entry point
│       └── main.go
├── internal/                      # Private application code
│   ├── config/                    # Configuration loading and file I/O
│   │   ├── definitions.go         # Config file reading with schema encoding
│   │   ├── definitions_test.go    # Integration tests
│   │   ├── env.go                 # Environment variable loading
│   │   ├── env_test.go            # Environment tests
│   │   ├── metadata.go            # Version and metadata loading from MDX
│   │   └── metadata_test.go       # Metadata tests
│   ├── github/                    # GitHub API integration
│   │   └── ...                    # Changed files detection
│   ├── parser/                    # MDX file parsing
│   │   └── ...                    # Frontmatter metadata extraction
│   └── models/                    # Data structures with validation
│       ├── models.go              # Type definitions + custom unmarshalers
│       └── models_test.go         # Model validation tests
├── .fleetControl/                 # Configuration files
│   ├── configurationDefinitions.yml
│   ├── agentControl/
│   │   └── agent-schema-for-agent-control.yml
│   └── schemas/
│       └── myagent-config.json
├── action.yml                     # GitHub Action definition
├── go.mod
└── run_local.sh                   # Local testing script
```

## Build and Test Commands

```bash
# Build the action
go build -o agent-metadata-action ./cmd/agent-metadata-action

# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test -v ./internal/config

# Run a specific test
go test -v -run TestLoadEnv_Success ./internal/config

# Local development (requires GITHUB_WORKSPACE to be set)
export GITHUB_WORKSPACE=/path/to/repo
./agent-metadata-action

# Or use run_local.sh for testing
./run_local.sh
```

## Architecture

### Package Organization

**cmd/agent-metadata-action/main.go**: Application entry point
- Validates agent-type is provided (required via `INPUT_AGENT_TYPE` environment variable)
- Loads workspace path via `config.GetWorkspace()` (required - returns error if not set)
- Validates that workspace directory exists
- Loads metadata via `config.LoadMetadata()` (version, features, bugs, security, deprecations, supportedOperatingSystems, eol from changed MDX files)
- Checks if `.fleetControl` directory exists to determine flow:
  - If `.fleetControl` exists (agent repo flow):
    - Reads configuration definitions via `config.ReadConfigurationDefinitions()`
    - Reads agent control via `config.LoadAndEncodeAgentControl()`
    - Combines into `AgentMetadata` structure
  - If `.fleetControl` doesn't exist (docs flow):
    - Only outputs metadata
- Uses `printJSON()` helper to marshal and output data
- Uses GitHub Actions annotation format for logging (`::error::`, `::notice::`, `::debug::`)

**internal/config**: Configuration file I/O with three main modules:

1. **env.go**: Environment variable handling
   - `GetWorkspace()`: Reads `GITHUB_WORKSPACE` environment variable

2. **definitions.go**: Configuration file reading with security
   - `ReadConfigurationDefinitions()`: Reads and validates configuration YAML
   - `loadAndEncodeSchema()`: Loads schema files and base64-encodes them
     - Validates paths to prevent directory traversal attacks (rejects `..`)
     - Ensures resolved paths stay within `.fleetControl` directory
   - `LoadAndEncodeAgentControl()`: Reads and base64-encodes agent control YAML
   - Validates that `configurationDefinitions` array is not empty

3. **metadata.go**: Version and changelog metadata
   - `LoadMetadata()`: Loads version and changelog info from changed MDX files
     - Gets changed MDX files via `github.GetChangedMDXFiles()` (in PR context)
     - Parses frontmatter metadata via `parser.ParseMDXFiles()`
     - Extracts: features, bugs, security, deprecations, supportedOperatingSystems, eol
   - `LoadVersion()`: Reads `INPUT_VERSION` with validation (format: X.Y.Z)

**internal/github**: GitHub API integration
- `GetChangedMDXFiles()`: Detects changed MDX files in pull request context
- Returns list of file paths for parsing

**internal/parser**: MDX file parsing
- `ParseMDXFiles()`: Extracts frontmatter metadata from MDX files
- Parses YAML frontmatter for: features, bugs, security, deprecations, supportedOperatingSystems, eol
- Returns structured metadata for LoadMetadata()

**internal/models**: Data structures with validation
- `ConfigurationDefinition`: Configuration with 6 required fields (validated via custom `UnmarshalYAML`)
  - Fields: version, platform, description, type, format, schema
  - Custom unmarshaler validates all fields are present before accepting
- `Metadata`: Version and changelog info (version required, validated via custom `UnmarshalYAML`)
  - Required: version
  - Optional: features, bugs, security, deprecations, supportedOperatingSystems, eol
- `AgentControl`: Agent control content (platform and content required, validated via custom `UnmarshalJSON`)
- `ConfigFile`: Root YAML structure containing `configurationDefinitions` array
- `AgentMetadata`: Complete metadata structure (configurationDefinitions + metadata + agentControl)
- All validation happens at unmarshal time with clear error messages

### Data Flow

**Agent Repository Workflow** (.fleetControl directory exists):
1. Validates `INPUT_AGENT_TYPE` is set (required)
2. `config.GetWorkspace()` reads `GITHUB_WORKSPACE` environment variable (required - errors if not set)
3. Validates workspace directory exists
4. `config.LoadMetadata()` loads version and changelog metadata:
   - `LoadVersion()` reads and validates `INPUT_VERSION` (X.Y.Z format)
   - `github.GetChangedMDXFiles()` detects changed MDX files in PR context
   - `parser.ParseMDXFiles()` extracts frontmatter metadata from MDX files
   - Returns structured metadata with features, bugs, security, deprecations, supportedOperatingSystems, eol
5. Checks if `.fleetControl` directory exists
6. `config.ReadConfigurationDefinitions()` constructs file path: `{workspace}/.fleetControl/configurationDefinitions.yml`
   - Reads and parses YAML file
   - Custom `UnmarshalYAML` validates all required fields on each ConfigurationDefinition
   - Validates array is not empty
   - For each config, loads schema file and base64-encodes it:
     - Validates schema path (no `..`, must stay within `.fleetControl`)
     - Reads schema file
     - Base64-encodes content and replaces path with encoded content
7. `config.LoadAndEncodeAgentControl()` reads `.fleetControl/agentControl/agent-schema-for-agent-control.yml`
   - Base64-encodes entire file content
   - Returns as single `AgentControl` entry with platform "all"
8. Main constructs `AgentMetadata` combining configs, metadata, and agent control
9. `printJSON()` marshals to JSON and prints with `::debug::` annotation

**Documentation Workflow** (.fleetControl directory doesn't exist):
1. Validates `INPUT_AGENT_TYPE` is set (required)
2. `config.GetWorkspace()` reads `GITHUB_WORKSPACE` environment variable (required - errors if not set)
3. Validates workspace directory exists
4. `config.LoadMetadata()` loads version and changelog metadata from changed MDX files
5. Checks if `.fleetControl` directory exists (doesn't exist in this flow)
6. Main outputs only the `Metadata` structure
7. `printJSON()` marshals to JSON and prints with `::debug::` annotation

### GitHub Action Integration

**action.yml** defines the composite action with the following steps:
1. **Automatic checkout**: Uses `actions/checkout@v4` to check out the calling repository at the specified version tag
   - The `ref` parameter is set to `inputs.version` to check out the exact version tag
   - Sets the `GITHUB_WORKSPACE` environment variable automatically
2. **Setup Go**: Uses `actions/setup-go@v4` with version from `go.mod` (currently Go 1.25.3)
   - Supports optional `cache` input to control Go build caching (defaults to `true`)
3. **Build and Run**: Builds binary from source and executes it
   - Changes to action directory: `cd ${{ github.action_path }}`
   - Builds: `go build -o agent-metadata-action ./cmd/agent-metadata-action`
   - Executes the built binary in the workspace
   - The action builds and runs on every invocation (no pre-built binary)
   - Passes `INPUT_AGENT_TYPE` and `INPUT_VERSION` as environment variables

The action automatically handles checkout, so users don't need a separate `actions/checkout` step.

## Key Behaviors

### File Operations
- Reads from **local filesystem** only (no network calls, except for GitHub API to get changed files)
- `GITHUB_WORKSPACE` is **required** - action fails if not set
- Action automatically checks out repository at specified version tag via `actions/checkout@v4`
- Target file paths are hardcoded:
  - `.fleetControl/configurationDefinitions.yml`
  - `.fleetControl/agentControl/agent-schema-for-agent-control.yml`
- Schema files are read from `.fleetControl/schemas/` (or subdirectories)
- MDX files are detected via GitHub API in PR context and parsed for frontmatter metadata

### Security
- **Directory traversal protection**: Schema paths cannot contain `..`
- **Path validation**: Resolved absolute paths must stay within `.fleetControl` directory
- Multiple layers of validation prevent escaping the designated directory

### Validation
- **Agent type is required**: `INPUT_AGENT_TYPE` must be set (validated in main.go)
- **Workspace is required**: `GITHUB_WORKSPACE` must be set and directory must exist
- **All configuration fields are required**: version, platform, description, type, format, schema
- **Version format validation**: Must match `X.Y.Z` (three numeric components, strict semver)
- **Empty array rejection**: `configurationDefinitions` cannot be an empty array
- **Validation timing**: All validation happens during YAML/JSON unmarshaling via custom unmarshalers
- **Error messages**: Clear, contextual errors (e.g., "platform is required for config with type 'mytype' and version '1.0.0'")

### Schema Handling
- Schema files are automatically loaded and **base64-encoded**
- Original relative paths (e.g., `./schemas/config.json`) are **replaced** with base64 content
- Empty schema files are rejected

### Error Handling
- Error output uses GitHub Actions annotation format: `::error::`, `::notice::`, `::debug::`
- All errors result in exit code 1
- Error messages include context (file paths, config names)

### Testing
- Tests use table-driven patterns for comprehensive coverage
- Unit tests (models) focus on validation logic
- Integration tests (config) focus on file I/O and wiring
- Tests for github and parser packages cover MDX file detection and parsing
