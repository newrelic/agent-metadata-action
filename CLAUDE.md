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
│   │   ├── config.go
│   │   ├── config_test.go
│   │   └── integration_test.go
│   └── models/                    # Data structures
│       └── models.go
├── .fleetControl/                 # Configuration files
│   ├── configurationDefinitions.yml
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
- Loads workspace path via `config.LoadEnv()`
- Calls `config.ReadConfigurationDefinitions()` to read and parse YAML
- Marshals results to JSON for output
- Uses GitHub Actions annotation format for logging (`::error::`, `::notice::`, `::debug::`)

**internal/config**: Configuration file I/O
- `LoadEnv()`: Reads `GITHUB_WORKSPACE` environment variable
- `ReadConfigurationDefinitions()`: Reads YAML file from local filesystem and parses it
- Tests cover environment variable validation, file reading, and YAML parsing errors

**internal/models**: Data structures
- `ConfigurationDefinition`: Represents a single configuration with fields like name, slug, platform, description, type, version, format, and schema
- `ConfigFile`: Root YAML structure containing `configurationDefinitions` array
- `AgentMetadata`: (Currently unused - may be used for future service integration)

### Data Flow

1. `config.LoadEnv()` reads `GITHUB_WORKSPACE` environment variable
2. `config.ReadConfigurationDefinitions()` constructs file path: `{workspace}/.fleetControl/configurationDefinitions.yml`
3. `os.ReadFile()` reads the YAML file from local filesystem
4. YAML is unmarshaled into `models.ConfigFile` structure
5. Array of `ConfigurationDefinition` is returned
6. Main function marshals each config to JSON and prints with `::debug::` annotations

### GitHub Action Integration

**action.yml** defines the composite action:
- Sets up Go using version from `go.mod` (currently Go 1.25.3)
- Builds binary from source: `go build -o agent-metadata-action ./cmd/agent-metadata-action`
- Executes the built binary in the workspace
- The action builds and runs on every invocation (no pre-built binary)
- Supports optional `cache` input to control Go build caching (defaults to `true`)

The action expects to run after `actions/checkout` which sets the `GITHUB_WORKSPACE` environment variable and checks out the repository containing the configuration file.

## Key Behaviors

- Reads from **local filesystem** only (no network calls)
- Requires `GITHUB_WORKSPACE` environment variable to be set
- Error output uses GitHub Actions annotation format: `::error::`, `::notice::`, `::debug::`
- All errors result in exit code 1
- Target file path is hardcoded: `.fleetControl/configurationDefinitions.yml`
- YAML structure maps directly to JSON output (no transformation/filtering)
- The `schema` field in configurations contains relative paths to JSON schema files
