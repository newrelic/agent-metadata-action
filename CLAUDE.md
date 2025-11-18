# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a GitHub Action written in Go that reads agent configuration metadata from a checked-out repository. The action reads a YAML configuration file (`.fleetControl/configurationDefinitions.yml`) from the local filesystem after the repository has been checked out by `actions/checkout`.

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
│   │   ├── definitions_test.go    # Integration tests (33 tests)
│   │   ├── env.go                 # Environment variable loading
│   │   ├── env_test.go            # Environment tests
│   │   ├── metadata.go            # Version and metadata loading
│   │   └── metadata_test.go       # Metadata tests
│   └── models/                    # Data structures with validation
│       ├── models.go              # Type definitions + custom unmarshalers
│       └── models_test.go         # Model validation tests (17 tests)
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
- Loads workspace path via `config.LoadEnv()` (returns empty string if not set)
- Loads metadata via `config.LoadMetadata()` (version, features, bugs, security)
- If workspace is set (agent repo flow):
  - Reads configuration definitions via `config.ReadConfigurationDefinitions()`
  - Reads agent control via `config.LoadAndEncodeAgentControl()`
  - Combines into `AgentMetadata` structure
- If workspace not set (docs flow):
  - Only outputs metadata
- Uses `printJSON()` helper to marshal and output data
- Uses GitHub Actions annotation format for logging (`::error::`, `::notice::`, `::debug::`)

**internal/config**: Configuration file I/O with three main modules:

1. **env.go**: Environment variable handling
   - `LoadEnv()`: Reads `GITHUB_WORKSPACE` (optional, returns empty string if not set)

2. **definitions.go**: Configuration file reading with security
   - `ReadConfigurationDefinitions()`: Reads and validates configuration YAML
   - `loadAndEncodeSchema()`: Loads schema files and base64-encodes them
     - Validates paths to prevent directory traversal attacks (rejects `..`)
     - Ensures resolved paths stay within `.fleetControl` directory
   - `LoadAndEncodeAgentControl()`: Reads and base64-encodes agent control YAML
   - Validates that `configurationDefinitions` array is not empty

3. **metadata.go**: Version and changelog metadata
   - `LoadMetadata()`: Loads version and changelog info from environment variables
   - `LoadVersion()`: Reads `INPUT_VERSION` with validation (format: X.Y.Z)
   - `parseCommaSeparated()`: Parses comma-separated lists (features, bugs, security)

**internal/models**: Data structures with validation
- `ConfigurationDefinition`: Configuration with 8 required fields (validated via custom `UnmarshalYAML`)
  - Fields: slug, name, version, platform, description, type, format, schema
  - Custom unmarshaler validates all fields are present before accepting
- `Metadata`: Version and changelog info (version required, validated via custom `UnmarshalYAML`)
- `AgentControl`: Agent control content (platform and content required, validated via custom `UnmarshalJSON`)
- `ConfigFile`: Root YAML structure containing `configurationDefinitions` array
- `AgentMetadata`: Complete metadata structure (configurationDefinitions + metadata + agentControl)
- All validation happens at unmarshal time with clear error messages

### Data Flow

**Agent Repository Workflow** (GITHUB_WORKSPACE is set):
1. `config.LoadEnv()` reads `GITHUB_WORKSPACE` environment variable
2. `config.LoadMetadata()` reads version and changelog from environment variables
3. `config.ReadConfigurationDefinitions()` constructs file path: `{workspace}/.fleetControl/configurationDefinitions.yml`
   - Reads and parses YAML file
   - Custom `UnmarshalYAML` validates all required fields on each ConfigurationDefinition
   - Validates array is not empty
   - For each config, loads schema file and base64-encodes it:
     - Validates schema path (no `..`, must stay within `.fleetControl`)
     - Reads schema file
     - Base64-encodes content and replaces path with encoded content
4. `config.LoadAndEncodeAgentControl()` reads `.fleetControl/agentControl/agent-schema-for-agent-control.yml`
   - Base64-encodes entire file content
   - Returns as single `AgentControl` entry with platform "all"
5. Main constructs `AgentMetadata` combining configs, metadata, and agent control
6. `printJSON()` marshals to JSON and prints with `::debug::` annotation

**Documentation Workflow** (GITHUB_WORKSPACE not set):
1. `config.LoadEnv()` returns empty string
2. `config.LoadMetadata()` reads version and changelog from environment variables
3. Main outputs only the `Metadata` structure
4. `printJSON()` marshals to JSON and prints with `::debug::` annotation

### GitHub Action Integration

**action.yml** defines the composite action:
- Sets up Go using version from `go.mod` (currently Go 1.25.3)
- Builds binary from source: `go build -o agent-metadata-action ./cmd/agent-metadata-action`
- Executes the built binary in the workspace
- The action builds and runs on every invocation (no pre-built binary)
- Supports optional `cache` input to control Go build caching (defaults to `true`)

The action expects to run after `actions/checkout` which sets the `GITHUB_WORKSPACE` environment variable and checks out the repository containing the configuration file.

## Key Behaviors

### File Operations
- Reads from **local filesystem** only (no network calls)
- `GITHUB_WORKSPACE` is optional (enables agent repo flow when set, uses docs flow when empty)
- Target file paths are hardcoded:
  - `.fleetControl/configurationDefinitions.yml`
  - `.fleetControl/agentControl/agent-schema-for-agent-control.yml`
- Schema files are read from `.fleetControl/schemas/` (or subdirectories)

### Security
- **Directory traversal protection**: Schema paths cannot contain `..`
- **Path validation**: Resolved absolute paths must stay within `.fleetControl` directory
- Multiple layers of validation prevent escaping the designated directory

### Validation
- **All configuration fields are required**: name, slug, version, platform, description, type, format, schema
- **Version format validation**: Must match `X.Y.Z` (three numeric components)
- **Empty array rejection**: `configurationDefinitions` cannot be an empty array
- **Validation timing**: All validation happens during YAML/JSON unmarshaling via custom unmarshalers
- **Error messages**: Clear, contextual errors (e.g., "platform is required for config 'MyConfig'")

### Schema Handling
- Schema files are automatically loaded and **base64-encoded**
- Original relative paths (e.g., `./schemas/config.json`) are **replaced** with base64 content
- Empty schema files are rejected

### Error Handling
- Error output uses GitHub Actions annotation format: `::error::`, `::notice::`, `::debug::`
- All errors result in exit code 1
- Error messages include context (file paths, config names)

### Testing
- **50 total tests** across 2 packages (33 config integration tests, 17 model unit tests)
- Tests use table-driven patterns for comprehensive coverage
- Unit tests (models) focus on validation logic
- Integration tests (config) focus on file I/O and wiring
