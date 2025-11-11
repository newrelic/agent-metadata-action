# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a GitHub Action written in Go that fetches and validates agent configuration metadata from a specified repository. The action reads a YAML configuration file (`.fleetControl/configs.yml`) from a target repository via the GitHub API and converts it to JSON format for processing.

## Project Structure

The project follows standard Go conventions:

```
├── cmd/
│   └── agent-metadata-action/    # Main application entry point
│       └── main.go
├── internal/                      # Private application code
│   ├── config/                    # Configuration loading and management
│   │   ├── config.go
│   │   ├── config_test.go
│   │   └── integration_test.go
│   ├── github/                    # GitHub API client
│   │   ├── client.go
│   │   └── client_test.go
│   └── models/                    # Data structures and transformations
│       ├── models.go
│       └── models_test.go
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
go test -v -run TestLoadSuccess ./internal/config

# Local development with environment variables
export AGENT_REPO="owner/repo"
export GITHUB_TOKEN="your_token"
export BRANCH="branch_name"  # Optional, defaults to default branch
./agent-metadata-action
```

The `run_local.sh` script provides a convenient way to test locally with environment variables.

## Architecture

### Package Organization

**cmd/agent-metadata-action/main.go**: Application entry point
- Loads configuration via `config.Load()`
- Calls `config.ReadConfigs()` to fetch and parse data
- Handles errors and prints results

**internal/config**: Configuration management
- `Load()`: Reads environment variables (`AGENT_REPO`, `GITHUB_TOKEN`, `BRANCH`)
- `ReadConfigs()`: Orchestrates fetching from GitHub and parsing YAML
- Tests cover environment variable validation and YAML parsing

**internal/github**: GitHub API client
- `Client`: HTTP client wrapper with 30-second timeout
- `GetClient()`: Returns singleton client instance (thread-safe with sync.Once)
- `NewClient()`: Creates new client instance (deprecated, use GetClient)
- `ResetClient()`: Resets singleton for testing
- `FetchFile()`: Fetches file from GitHub Contents API with optional branch
- Handles Bearer token authentication
- Decodes base64-encoded responses
- Singleton pattern ensures connection reuse across multiple API calls
- Tests use httptest to mock API responses and reset singleton between tests

**internal/models**: Data structures and conversions
- `ConfigYaml`: Structure matching YAML file format (includes `schema` field)
- `ConfigJson`: Output structure (excludes `schema` field)
- `ConfigFile`: Root YAML structure containing configs array
- `ConvertToConfigJson()`: Transforms YAML configs to JSON format
- Tests verify struct conversions and field mappings

### Data Flow

1. `config.LoadEnv()` reads environment variables
2. `config.ReadConfigs()` gets singleton GitHub client via `github.GetClient()`
3. GitHub client fetches `.fleetControl/configs.yml` from target repo
4. YAML is unmarshaled into `models.ConfigFile`
5. `models.ConvertToConfigJson()` strips `schema` field
6. Main prints formatted output to stdout

**Note**: The GitHub client uses a singleton pattern, so the first call to `GetClient()` initializes the client with the token, and subsequent calls return the same instance for connection reuse.

### GitHub Action Integration

**action.yml** defines the composite action:
- Sets up Go 1.21
- Builds binary from source: `go build -o agent-metadata-action ./cmd/agent-metadata-action`
- Passes inputs as environment variables
- The action builds and runs on every invocation (no pre-built binary)

## Key Behaviors

- GitHub API calls include optional branch via `?ref=` query parameter
- Error output uses GitHub Actions annotation format: `::error::` and `::notice::`
- HTTP client has 30-second timeout
- All errors result in exit code 1
- Target file path is hardcoded: `.fleetControl/configs.yml`