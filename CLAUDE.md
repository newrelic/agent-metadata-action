# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a GitHub Action written in Go that reads agent configuration metadata from a repository and sends it to a NewRelic instrumentation service. The action automatically checks out the calling repository at a specified version tag (for agent repos) or at the PR commit (for docs repos), then reads configuration from `.fleetControl/configurationDefinitions.yml` and/or metadata from changed MDX files. The action supports two workflows:

1. **Agent Repository Workflow**: When both `agent-type` and `version` inputs are provided, reads configuration definitions and agent control files from `.fleetControl` directory, then sends to instrumentation service
2. **Documentation Workflow**: When `agent-type` or `version` are not provided, reads metadata from changed MDX files in PR context and sends each entry separately to instrumentation service

Both workflows authenticate with NewRelic and send data to the instrumentation service via HTTP POST requests.

## Project Structure

The project follows standard Go conventions:

```
├── cmd/
│   └── agent-metadata-action/    # Main application entry point
│       ├── main.go
│       └── main_test.go
├── internal/                      # Private application code
│   ├── client/                    # HTTP client for instrumentation service
│   │   ├── instrumentation.go     # NewRelic instrumentation API client
│   │   └── instrumentation_test.go
│   ├── config/                    # Environment and configuration paths
│   │   ├── env.go                 # Environment variable loading
│   │   ├── dirs.go                # Directory path configuration
│   │   └── urls.go                # Service URL configuration
│   ├── loader/                    # Data loading and encoding
│   │   ├── configuration_definitions.go       # Config definitions loader
│   │   ├── configuration_definitions_test.go
│   │   ├── agent_control_definitions.go       # Agent control loader
│   │   ├── agent_control_definitions_test.go
│   │   ├── metadata.go            # Metadata loading for both flows
│   │   └── metadata_test.go
│   ├── fileutil/                  # File utilities
│   │   ├── fileutil.go
│   │   └── fileutil_test.go
│   ├── github/                    # GitHub API integration
│   │   ├── push.go                # Changed files detection in PRs
│   │   └── push_test.go
│   ├── parser/                    # MDX file parsing
│   │   ├── mdx.go                 # Frontmatter metadata extraction
│   │   ├── mdx_test.go
│   │   └── mdx_bench_test.go
│   ├── models/                    # Data structures with validation
│   │   ├── models.go              # Type definitions + custom unmarshalers
│   │   └── models_test.go
│   └── testutil/                  # Test utilities
│       └── testutil.go
├── .fleetControl/                 # Configuration files (example structure)
│   ├── configurationDefinitions.yml
│   ├── agentControl/
│   │   └── *.yml                  # Multiple agent control YAML files
│   └── schemas/
│       └── *.json                 # Schema files
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
- `validateEnvironment()`: Validates required environment variables
  - Checks `GITHUB_WORKSPACE` is set and directory exists
  - Checks `NEWRELIC_TOKEN` is set (required for service authentication)
- `run()`: Main orchestration logic
  - Creates instrumentation client for sending data to service
  - Determines which flow to execute based on inputs:
    - If both `INPUT_AGENT_TYPE` and `INPUT_VERSION` are set → agent flow
    - Otherwise → docs flow
- `runAgentFlow()`: Agent repository workflow
  - Validates `.fleetControl` directory exists
  - Loads configuration definitions via `loader.ReadConfigurationDefinitions()`
  - Loads agent control definitions via `loader.ReadAgentControlDefinitions()` (optional, warns on error)
  - Creates metadata structure with version
  - Sends to instrumentation service via `client.SendMetadata()`
- `runDocsFlow()`: Documentation workflow
  - Loads metadata from changed MDX files via `loader.LoadMetadataForDocs()`
  - Sends each metadata entry separately to instrumentation service
  - Continues on errors (warns but doesn't fail entire flow)
- `printJSON()`: Helper to marshal and output data with `::debug::` annotation
- Uses GitHub Actions annotation format for logging (`::error::`, `::notice::`, `::debug::`, `::group::`)

**internal/client**: HTTP client for instrumentation service

1. **instrumentation.go**: NewRelic instrumentation API client
   - `NewInstrumentationClient()`: Creates HTTP client with 30s timeout
   - `SendMetadata()`: Sends agent metadata to instrumentation service
     - POST to `/v1/agents/{agentType}/versions/{agentVersion}`
     - Uses Bearer token authentication
     - Returns error on non-2xx status codes
     - Includes detailed debug logging throughout request lifecycle

**internal/config**: Environment variables and configuration paths

1. **env.go**: Environment variable loading
   - `GetWorkspace()`: Reads `GITHUB_WORKSPACE`
   - `GetRepo()`: Reads `GITHUB_REPOSITORY`
   - `GetAgentType()`: Reads `INPUT_AGENT_TYPE`
   - `GetVersion()`: Reads `INPUT_VERSION`
   - `GetEventPath()`: Reads `GITHUB_EVENT_PATH`
   - `GetToken()`: Reads `NEWRELIC_TOKEN`

2. **dirs.go**: Directory path configuration
   - `GetRootFolderForAgentRepo()`: Returns `.fleetControl`
   - `GetConfigurationDefinitionsFilepath()`: Returns `.fleetControl/configurationDefinitions.yml`
   - `GetAgentControlFolderForAgentRepo()`: Returns `.fleetControl/agentControl`

3. **urls.go**: Service URL configuration
   - `GetMetadataURL()`: Returns instrumentation service base URL

**internal/loader**: Data loading and encoding

1. **configuration_definitions.go**: Configuration definitions loader
   - `ReadConfigurationDefinitions()`: Reads and validates configuration YAML
     - Parses `.fleetControl/configurationDefinitions.yml`
     - Validates array is not empty
     - For each config, loads and base64-encodes schema file (if provided)
   - `loadAndEncodeSchema()`: Loads schema files and base64-encodes them
     - Validates paths to prevent directory traversal attacks (rejects `..`)
     - Ensures resolved paths stay within `.fleetControl` directory
     - Validates JSON format
     - Empty schemas are rejected

2. **agent_control_definitions.go**: Agent control definitions loader
   - `ReadAgentControlDefinitions()`: Reads all YAML files from agentControl folder
     - Uses glob pattern `*.y*ml` to find all YAML files
     - Each file is read and base64-encoded separately
     - Platform is set to "ALL" for all entries
     - Continues on individual file errors (warns but doesn't fail)
   - `loadAndEncodeAgentControl()`: Reads and base64-encodes a single agent control file

3. **metadata.go**: Metadata loading for both flows
   - `LoadMetadataForAgents()`: Creates simple metadata with only version field
   - `LoadMetadataForDocs()`: Loads metadata from changed MDX files in PR
     - Gets changed MDX files via `github.GetChangedMDXFiles()`
     - Parses frontmatter via `parser.ParseMDXFile()`
     - Extracts `subject` field and maps to agent type
     - Validates `version` and `subject` are present
     - Returns list of `MetadataForDocs` entries
     - Continues on individual file errors (warns but doesn't fail)

**internal/github**: GitHub API integration
- `GetChangedMDXFiles()`: Detects changed MDX files in pull request context
- Returns list of file paths for parsing

**internal/parser**: MDX file parsing
- `ParseMDXFile()`: Extracts frontmatter metadata from a single MDX file
- Parses YAML frontmatter and returns as map
- `SubjectToAgentTypeMapping`: Maps subject values to agent type strings

**internal/models**: Data structures with validation
- `ConfigurationDefinition`: Type alias for `map[string]any` with flexible field support
- `Metadata`: Type alias for `map[string]any` with flexible field support
- `AgentControlDefinition`: Struct with `Platform` and `Content` (base64-encoded) fields
- `ConfigFile`: Root YAML structure containing `Configs` array (renamed from `configurationDefinitions`)
- `AgentMetadata`: Complete metadata structure with three fields:
  - `ConfigurationDefinitions`: Array of configuration maps
  - `Metadata`: Metadata map
  - `AgentControlDefinitions`: Array of agent control definitions
- Validation is now more flexible - fields are not strictly required, allowing varied data shapes

### Data Flow

**Agent Repository Workflow** (when both `INPUT_AGENT_TYPE` and `INPUT_VERSION` are provided):
1. `validateEnvironment()` validates required environment variables:
   - `GITHUB_WORKSPACE` must be set and directory must exist
   - `NEWRELIC_TOKEN` must be set (for service authentication)
2. `run()` reads `INPUT_AGENT_TYPE` and `INPUT_VERSION` from environment
3. Both are present → calls `runAgentFlow()`
4. `runAgentFlow()` validates `.fleetControl` directory exists in workspace
5. `loader.ReadConfigurationDefinitions()` loads configuration definitions:
   - Reads `{workspace}/.fleetControl/configurationDefinitions.yml`
   - Parses YAML into array of configuration maps
   - Validates array is not empty
   - For each config with a schema path:
     - Validates schema path (no `..`, must stay within `.fleetControl`)
     - Reads schema file and validates JSON format
     - Base64-encodes content and replaces path with encoded content
     - Warns and continues if schema loading fails
6. `loader.ReadAgentControlDefinitions()` loads agent control definitions:
   - Finds all `.yml` and `.yaml` files in `{workspace}/.fleetControl/agentControl/`
   - For each file:
     - Reads file content
     - Base64-encodes entire file
     - Creates `AgentControlDefinition` with platform "ALL"
   - Warns and continues if individual files fail to load
   - Returns empty array if no files found or all fail (non-fatal)
7. `loader.LoadMetadataForAgents()` creates simple metadata map with version
8. Main constructs `AgentMetadata` combining configs, metadata, and agent control
9. `printJSON()` outputs metadata with `::debug::` annotation
10. `client.SendMetadata()` sends data to instrumentation service:
    - POST to `{baseURL}/v1/agents/{agentType}/versions/{agentVersion}`
    - Bearer token authentication with `NEWRELIC_TOKEN`
    - Returns error on non-2xx response
11. Success message logged with `::notice::`

**Documentation Workflow** (when `INPUT_AGENT_TYPE` or `INPUT_VERSION` is not provided):
1. `validateEnvironment()` validates required environment variables:
   - `GITHUB_WORKSPACE` must be set and directory must exist
   - `NEWRELIC_TOKEN` must be set (for service authentication)
2. `run()` reads `INPUT_AGENT_TYPE` and `INPUT_VERSION` from environment
3. One or both are missing → calls `runDocsFlow()`
4. `loader.LoadMetadataForDocs()` loads metadata from changed MDX files:
   - `github.GetChangedMDXFiles()` detects changed MDX files in PR context
   - For each changed MDX file:
     - `parser.ParseMDXFile()` extracts frontmatter metadata
     - Validates `version` and `subject` fields are present
     - Maps `subject` to agent type using `SubjectToAgentTypeMapping`
     - Creates `MetadataForDocs` entry
   - Warns and continues if individual files fail to parse
   - Returns error if all files fail
   - Returns empty list if no changed files detected
5. If no metadata entries, logs notice and exits successfully
6. For each metadata entry:
   - `printJSON()` outputs metadata with `::debug::` annotation
   - `client.SendMetadata()` sends to instrumentation service
     - POST to `{baseURL}/v1/agents/{agentType}/versions/{version}`
     - Bearer token authentication with `NEWRELIC_TOKEN`
   - Warns and continues if individual entry fails to send
7. Summary logged with count of successful sends

### GitHub Action Integration

**action.yml** defines the composite action with the following inputs and steps:

**Inputs:**
- `newrelic-client-id` (required): NewRelic client ID for authentication
- `newrelic-private-key` (required): NewRelic private key content (base64-encoded)
- `agent-type` (optional): Agent type in lowercase (e.g., "dotnet-agent")
- `version` (optional): Agent version in semver format (e.g., "1.2.3" or "v1.2.3")
- `fetch-depth` (optional, default: 1): Number of commits to fetch (>1 may be needed for docs flow)
- `cache` (optional, default: true): Enable Go build caching

**Steps:**

1. **Normalize version tag**: Prepends "v" to version if not present for tag checkout
   - Output: `ref` variable with normalized version tag

2. **Checkout repository**: Uses `actions/checkout@v4`
   - For agent flow: Checks out the normalized version tag (e.g., "v1.2.3")
   - For docs flow: Checks out the PR commit (when `ref` is empty)
   - Uses `fetch-depth` input for controlling commit history depth
   - Sets `GITHUB_WORKSPACE` environment variable automatically

3. **Setup Go**: Uses `actions/setup-go@v4`
   - Reads Go version from action's `go.mod` file
   - Uses `cache` input to control Go build caching (defaults to `true`)
   - Caches based on action's `go.sum` file

4. **Install NewRelic Auth CLI**: Downloads and installs `newrelic-auth-cli`
   - Fetches latest release from GitHub
   - Downloads AMD64 Linux binary
   - Installs to `/usr/local/bin/`
   - Skips installation if `MOCK_NEWRELIC_AUTH_CLI=true` (testing mode)

5. **Authenticate with NewRelic**: Obtains access token
   - Masks sensitive credentials in logs
   - Writes private key to temporary file with secure permissions (600)
   - Calls `newrelic-auth-cli authenticate` with client ID and private key
   - Extracts JWT access token from JSON response
   - Masks token in logs for security
   - Cleans up temporary key file
   - Sets `token` output for next step

6. **Build and Run**: Builds and executes the action
   - Changes to action directory: `cd ${{ github.action_path }}`
   - Builds: `go build -o agent-metadata-action ./cmd/agent-metadata-action`
   - Executes the built binary
   - Passes environment variables:
     - `INPUT_AGENT_TYPE`: From `inputs.agent-type`
     - `INPUT_VERSION`: From `inputs.version`
     - `NEWRELIC_TOKEN`: From authentication step output
   - The action builds and runs on every invocation (no pre-built binary)

**Key behaviors:**
- The action automatically handles repository checkout, so users don't need a separate `actions/checkout` step
- Version normalization ensures consistent tag format (with "v" prefix)
- Authentication is handled automatically and securely (credentials masked in logs)
- Both agent and docs flows are supported based on whether inputs are provided

## Key Behaviors

### File Operations
- Reads from **local filesystem** only (no network calls except GitHub API and instrumentation service)
- `GITHUB_WORKSPACE` is **required** - action fails if not set
- Action automatically checks out repository via `actions/checkout@v4`:
  - Agent flow: Checks out specified version tag
  - Docs flow: Checks out PR commit
- Target file paths are hardcoded:
  - `.fleetControl/configurationDefinitions.yml`
  - `.fleetControl/agentControl/*.yml` (all YAML files in directory)
- Schema files are read from `.fleetControl/` (typically in `schemas/` subdirectory)
- MDX files are detected via GitHub API in PR context and parsed for frontmatter metadata

### Service Integration
- **Sends data to NewRelic instrumentation service** via HTTP POST
- Endpoint: `POST /v1/agents/{agentType}/versions/{agentVersion}`
- Authentication: Bearer token from NewRelic Auth CLI
- Timeout: 30 seconds per request
- Both agent and docs flows send data to the same service
- Docs flow sends one request per changed MDX file
- Agent flow sends one consolidated request with all configuration data

### Security
- **Directory traversal protection**: Schema paths cannot contain `..`
- **Path validation**: Resolved absolute paths must stay within `.fleetControl` directory
- **Credential masking**: NewRelic credentials and tokens are masked in logs
- **Private key handling**: Private keys written to temporary files with secure permissions (600)
- **Token security**: Authentication tokens never logged in plaintext
- Multiple layers of validation prevent escaping the designated directory

### Validation
- **Required for all flows**:
  - `GITHUB_WORKSPACE` must be set and directory must exist
  - `NEWRELIC_TOKEN` must be set (obtained via authentication step)
- **Required for agent flow only**:
  - Both `INPUT_AGENT_TYPE` and `INPUT_VERSION` must be set
  - `.fleetControl` directory must exist
  - `configurationDefinitions.yml` must contain non-empty array
- **Required for docs flow only**:
  - MDX files must have `version` and `subject` fields in frontmatter
- **Flexible field validation**:
  - Configuration definitions use flexible map structure
  - Metadata uses flexible map structure
  - Allows varied data shapes for different agent types
- **Validation timing**: Most validation happens at load time with clear error messages
- **Error handling**: Partial failures are tolerated (warns and continues)

### Schema Handling
- Schema files are **optional** - warnings issued if missing but action continues
- When present, schemas are automatically loaded and **base64-encoded**
- Original relative paths (e.g., `./schemas/config.json`) are **replaced** with base64 content
- Empty schema files are rejected
- JSON validation performed on schema content

### Agent Control Handling
- **Multiple files supported**: All `.yml` and `.yaml` files in agentControl folder
- Each file is read and base64-encoded separately
- Platform is set to "ALL" for all entries
- **Optional**: Warnings issued if files missing or fail to load, but action continues
- Empty files are rejected

### Error Handling
- Error output uses GitHub Actions annotation format: `::error::`, `::notice::`, `::debug::`, `::group::`, `::warn::`
- Fatal errors result in exit code 1
- Non-fatal errors (individual file failures) warn and continue
- Error messages include context (file paths, agent types, versions)
- HTTP errors include status codes and response bodies (truncated if large)

### Resilience
- **Partial failure tolerance**: Individual file parsing errors don't fail entire flow
- **Graceful degradation**: Missing optional components (schemas, agent control) don't block execution
- **Detailed logging**: Extensive debug logging helps troubleshooting without failing execution
- **Continue-on-error pattern**: Docs flow sends as many entries as possible even if some fail

### Testing

**Testing Philosophy**:
- Maintain high test coverage (94%+ overall, 100% for critical packages)
- Use parameterized/table-driven tests to avoid duplication
- Test both success and error paths comprehensively
- Mock external dependencies (GitHub API, HTTP clients, file system where needed)
- Use descriptive test names that explain what is being tested

**Test Patterns**:
- **Parameterized tests**: Group related test cases into single test functions with subtests
  - Example: `TestReadConfigurationDefinitions_ErrorCases` with multiple sub-cases for different error scenarios
  - Reduces code duplication and makes it easier to add new test cases
  - Each test case clearly labeled with `name` field for easy identification
- **Table-driven tests**: Use struct slices to define test cases with inputs and expected outputs
  - Common pattern: `tests := []struct { name string; input X; expected Y; expectedErr string }{...}`
- **Setup functions**: Complex test scenarios use setup functions that return test fixtures
  - Example: `setupFunc func(t *testing.T, tmpDir string) string` for preparing test files
- **Mock patterns**: Use function variables and interfaces for dependency injection
  - Example: `github.GetChangedMDXFilesFunc` allows tests to mock GitHub API calls
  - HTTP client tests use custom `RoundTripper` implementations for error scenarios

**Package-Specific Testing**:
- **models**: Data structure validation, custom unmarshalers, JSON parsing
- **loader** (94.6% coverage):
  - `agent_control_definitions`: Multiple YAML file loading, base64 encoding, error handling
  - `configuration_definitions`: Schema validation, directory traversal protection, base64 encoding
  - `metadata`: MDX file parsing, version validation, subject mapping to agent types
- **client**: HTTP client behavior, authentication, error responses, network failures
- **github**: Git diff parsing, SHA validation (prevents command injection), MDX file filtering
- **parser**: MDX frontmatter extraction, YAML parsing, subject-to-agent-type mapping
- **main**: Flow orchestration, environment validation, error handling for both agent and docs flows

**Test Utilities** (`internal/testutil`):
- `CaptureOutput()`: Captures stdout/stderr for testing log output and warnings
- Used extensively to verify `::warn::`, `::debug::`, and `::notice::` GitHub Actions annotations

**Coverage Goals**:
- Critical paths: 100% (authentication, data sending, security validations)
- Business logic: 95%+ (loaders, parsers, validators)
- Overall: 90%+ maintained
- Uncovered lines are typically: OS-level errors (filepath.Abs failures), unreachable defensive code

**Running Tests**:
```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test -v ./internal/loader

# Run specific test
go test -v -run TestLoadMetadataForDocs_ErrorCases ./internal/loader
```
