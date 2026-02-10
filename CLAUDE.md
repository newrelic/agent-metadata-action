# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a GitHub Action written in Go that reads agent configuration metadata from a repository and sends it to a NewRelic instrumentation service. The action automatically checks out the calling repository at a specified version tag (for agent repos) or at the PR commit (for docs repos), then reads configuration from `.fleetControl/configurationDefinitions.yml` and/or metadata from changed MDX files. The action supports two workflows:

1. **Agent Repository Workflow**: When both `agent-type` and `version` inputs are provided, reads configuration definitions and agent control files from `.fleetControl` directory, sends to instrumentation service, and optionally uploads binary artifacts to an OCI registry
2. **Documentation Workflow**: When `agent-type` or `version` are not provided, reads metadata from changed MDX files in PR context and sends each entry separately to instrumentation service

Both workflows authenticate with NewRelic and send data to the instrumentation service via HTTP POST requests. The agent workflow can also upload binary artifacts (tar.gz, zip) to OCI-compatible registries like Docker Hub, GitHub Container Registry, or local registries.

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
│   │   ├── definitions.go         # Config & agent control definitions loader
│   │   ├── definitions_test.go
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
│   │   ├── models.go              # Agent metadata type definitions
│   │   ├── models_test.go
│   │   ├── binary.go              # OCI artifact type definitions
│   │   └── binary_test.go
│   ├── oci/                       # OCI registry integration
│   │   ├── annotations.go         # OCI metadata annotations
│   │   ├── annotations_test.go
│   │   ├── client.go              # OCI registry client (using oras-go)
│   │   ├── config.go              # OCI configuration loader
│   │   ├── handler.go             # Upload orchestration
│   │   ├── handler_test.go
│   │   ├── upload.go              # Artifact upload logic
│   │   └── validation.go          # Binary path validation
│   │   └── validation_test.go
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
  - Handles optional OCI binary uploads via `oci.HandleUploads()`:
    - Loads OCI configuration from environment variables
    - If OCI registry is configured, validates and uploads binary artifacts
    - Creates multi-platform manifest index tagged with version
    - Continues on errors if OCI is disabled or fails validation
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

1. **definitions.go**: Configuration and agent control definitions loader
   - `ReadConfigurationDefinitions()`: Reads and validates configuration YAML
     - Parses `.fleetControl/configurationDefinitions.yml`
     - Validates array is not empty
     - For each config, loads and base64-encodes schema file (if provided)
     - Warns and continues if schema path is invalid or file is missing
   - `ReadAgentControlDefinitions()`: Reads and validates agent control YAML
     - Parses `.fleetControl/agentControlDefinitions.yml`
     - Validates array is not empty
     - For each definition, loads and base64-encodes content file (if provided)
     - Warns and continues if content path is invalid or file is missing
   - `readDefinitionsFile()`: Generic YAML array reader
     - Finds first array in YAML file at top level
     - Validates array items are maps
     - Returns error if array is empty or not found
   - `loadAndEncodeFile()`: Reads files and base64-encodes them
     - Validates paths to prevent directory traversal attacks (rejects `..`)
     - Ensures resolved paths stay within `.fleetControl` directory
     - Returns error for empty files
     - Used for both schema and content files

2. **metadata.go**: Metadata loading for both flows
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

**internal/oci**: OCI registry integration (optional feature)

1. **config.go**: OCI configuration loader
   - `LoadConfig()`: Reads OCI configuration from environment variables
     - Reads `INPUT_OCI_REGISTRY`, `INPUT_OCI_USERNAME`, `INPUT_OCI_PASSWORD`, `INPUT_BINARIES`
     - Parses `INPUT_BINARIES` JSON into array of artifact definitions
     - Validates configuration via `OCIConfig.Validate()`
     - Returns error if registry is set but binaries are missing or invalid

2. **client.go**: OCI registry client (using oras-go v2)
   - `NewClient()`: Creates OCI registry client
     - Configures authentication with username/password
     - Enables plainHTTP for localhost registries (localhost:*, 127.0.0.1:*)
     - Returns configured client for registry operations
   - `UploadArtifact()`: Uploads a single artifact to registry
     - Creates temporary file store for manifest generation
     - Adds artifact layer with annotations (title, version, type)
     - Creates OCI manifest with empty config descriptor
     - Packs manifest with proper media types and annotations
     - Copies manifest to remote registry by digest
     - Returns manifest digest and size on success
   - `CreateManifestIndex()`: Creates multi-platform manifest index
     - Builds index from uploaded artifact digests
     - Includes platform information (OS/arch) for each manifest
     - Tags index with version (e.g., "v1.2.3")
     - Returns index digest on success

3. **handler.go**: Upload orchestration
   - `HandleUploads()`: Main entry point for OCI uploads
     - Returns early if OCI is disabled (no registry configured)
     - Validates all artifact paths exist and are readable
     - Creates OCI client
     - Uploads artifacts in sequence
     - Logs success/failure for each artifact with digest and size
     - Creates manifest index if all uploads succeed
     - Returns error if any uploads fail

4. **upload.go**: Artifact upload logic
   - `UploadArtifacts()`: Uploads multiple artifacts
     - Resolves artifact paths relative to workspace
     - Calls `client.UploadArtifact()` for each artifact
     - Collects results (success/failure) for all artifacts
     - Returns array of `ArtifactUploadResult`
   - `HasFailures()`: Checks if any uploads failed

5. **validation.go**: Binary path validation
   - `ValidateBinaryPath()`: Security validation for artifact paths
     - Rejects paths with directory traversal (`..`)
     - Ensures paths resolve within workspace directory
     - Checks file exists, is readable, and is not empty
     - Returns error if file is a directory
   - `ValidateAllArtifacts()`: Validates all configured artifacts
   - `ResolveArtifactPath()`: Resolves relative paths to absolute paths

6. **annotations.go**: OCI metadata annotations
   - `CreateLayerAnnotations()`: Creates annotations for artifact layers
     - Sets `org.opencontainers.image.title` to filename
     - Sets `org.opencontainers.image.version` to version
     - Sets `com.newrelic.artifact.type` to "binary"
   - `CreateManifestAnnotations()`: Creates manifest-level annotations
     - Sets `org.opencontainers.image.created` timestamp (RFC3339 format)

**internal/models**: Data structures with validation

**models.go** - Agent metadata types:
- `ConfigurationDefinition`: Type alias for `map[string]any` with flexible field support
- `Metadata`: Type alias for `map[string]any` with flexible field support
- `AgentControlDefinition`: Type alias for `map[string]any` with flexible field support
- `AgentMetadata`: Complete metadata structure with three fields:
  - `ConfigurationDefinitions`: Array of configuration maps
  - `Metadata`: Metadata map
  - `AgentControlDefinitions`: Array of agent control definitions
- Validation is flexible - fields are not strictly required, allowing varied data shapes

**binary.go** - OCI artifact types:
- `ArtifactDefinition`: Defines a binary artifact for OCI upload
  - `Name`: Unique identifier for artifact (alphanumeric, hyphens, underscores only)
  - `Path`: File path relative to workspace (e.g., "./dist/agent.tar.gz")
  - `OS`: Target operating system ("linux", "windows", "darwin", or "any")
  - `Arch`: Target architecture ("amd64", "arm64", or "any")
  - `Format`: Archive format ("tar", "tar+gzip", or "zip")
  - Includes validation methods and helpers for media types, platform strings, filenames
- `OCIConfig`: OCI registry configuration
  - `Registry`: OCI registry URL (e.g., "ghcr.io/newrelic/agents")
  - `Username`: Registry username (optional for local registries)
  - `Password`: Registry password or token (optional for local registries)
  - `Artifacts`: Array of artifact definitions
  - `IsEnabled()`: Returns true if registry is configured
  - `Validate()`: Validates configuration (requires binaries if registry is set, checks for duplicate names)
- `ArtifactUploadResult`: Result of artifact upload operation
  - Includes success/failure status, digest, size, and error message

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
12. `oci.LoadConfig()` loads OCI configuration from environment variables:
    - Reads `INPUT_OCI_REGISTRY`, `INPUT_OCI_USERNAME`, `INPUT_OCI_PASSWORD`, `INPUT_BINARIES`
    - Parses binaries JSON into artifact definitions
    - Validates configuration (optional - skips if registry not set)
13. `oci.HandleUploads()` performs binary uploads (if OCI is enabled):
    - Validates all artifact files exist and are readable
    - Creates OCI registry client with authentication
    - For each artifact:
      - Uploads artifact with proper OCI manifest and annotations
      - Logs success with digest and size, or logs error
    - Creates multi-platform manifest index tagged with version
    - Returns error if any uploads fail
14. Final success message logged

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
- `agent-type` (optional): Agent type (e.g., "NRDotNetAgent")
- `version` (optional): Agent version tag name (e.g., "v1.2.3"). Must match the exact git tag name for checkout.
- `fetch-depth` (optional, default: 1): Number of commits to fetch (>1 may be needed for docs flow)
- `cache` (optional, default: true): Enable Go build caching
- `oci-registry` (optional): OCI registry URL for binary uploads (e.g., "ghcr.io/newrelic/agents"). Leave empty to skip binary upload.
- `oci-username` (optional): OCI registry username (required if oci-registry is set)
- `oci-password` (optional): OCI registry password or token (required if oci-registry is set)
- `binaries` (optional): JSON array with artifact definitions. Each artifact must specify name, path, os, arch, and format. Example: `[{"name": "linux-tar", "path": "./dist/agent.tar.gz", "os": "linux", "arch": "amd64", "format": "tar+gzip"}]`

**Steps:**

1. **Set version ref**: Sets the version tag for checkout
   - Output: `ref` variable with the exact version tag provided

2. **Checkout repository**: Uses `actions/checkout@v4`
   - For agent flow: Checks out the exact version tag as provided (e.g., "v1.2.3")
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
     - `INPUT_OCI_REGISTRY`: From `inputs.oci-registry`
     - `INPUT_OCI_USERNAME`: From `inputs.oci-username`
     - `INPUT_OCI_PASSWORD`: From `inputs.oci-password`
     - `INPUT_BINARIES`: From `inputs.binaries`
   - The action builds and runs on every invocation (no pre-built binary)

**Key behaviors:**
- The action automatically handles repository checkout, so users don't need a separate `actions/checkout` step
- Version tag must match the exact git tag name (no automatic normalization)
- Authentication is handled automatically and securely (credentials masked in logs)
- Both agent and docs flows are supported based on whether inputs are provided
- OCI binary uploads are optional and only occur in agent flow when `oci-registry` is configured

## Key Behaviors

### File Operations
- Reads from **local filesystem** only (no network calls except GitHub API, instrumentation service, and OCI registry)
- `GITHUB_WORKSPACE` is **required** - action fails if not set
- Action automatically checks out repository via `actions/checkout@v4`:
  - Agent flow: Checks out specified version tag
  - Docs flow: Checks out PR commit
- Target file paths are hardcoded:
  - `.fleetControl/configurationDefinitions.yml`
  - `.fleetControl/agentControlDefinitions.yml`
- Schema and content files are read from `.fleetControl/` (typically in `schemas/` or `agentControl/` subdirectories)
- MDX files are detected via GitHub API in PR context and parsed for frontmatter metadata
- Binary artifacts are read from workspace-relative paths specified in `binaries` input (e.g., `./dist/agent.tar.gz`)

### Service Integration
- **Sends data to NewRelic instrumentation service** via HTTP POST
  - Endpoint: `POST /v1/agents/{agentType}/versions/{agentVersion}`
  - Authentication: Bearer token from NewRelic Auth CLI
  - Timeout: 30 seconds per request
  - Both agent and docs flows send data to the same service
  - Docs flow sends one request per changed MDX file
  - Agent flow sends one consolidated request with all configuration data
- **Uploads binaries to OCI registry** (optional, agent flow only)
  - Uses ORAS (OCI Registry as Storage) protocol via oras-go v2 library
  - Supports any OCI-compatible registry (Docker Hub, GitHub Container Registry, local registries)
  - Uploads each artifact with proper OCI manifest, platform metadata, and annotations
  - Creates multi-platform manifest index tagged with version for easy discovery
  - Timeout: 5 minutes per artifact upload
  - Authentication: Username/password or token-based (optional for localhost registries)

### Security
- **Directory traversal protection**:
  - Schema and content paths in `.fleetControl` cannot contain `..`
  - Binary artifact paths cannot contain `..`
  - Resolved absolute paths must stay within designated directories
- **Path validation**:
  - `.fleetControl` files must stay within `.fleetControl` directory
  - Binary artifacts must stay within `GITHUB_WORKSPACE` directory
  - Files must exist, be readable, and not be empty
  - Directories are rejected (must be files)
- **Credential masking**: NewRelic credentials and tokens are masked in logs
- **Private key handling**: Private keys written to temporary files with secure permissions (600)
- **Token security**: Authentication tokens never logged in plaintext
- **OCI authentication**: Registry credentials passed securely through environment variables
- Multiple layers of validation prevent escaping designated directories

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
- **OCI upload validation** (optional, agent flow only):
  - If `oci-registry` is set, `binaries` input is required and must be valid JSON
  - Each artifact must have: `name`, `path`, `os`, `arch`, `format`
  - Artifact names must be unique and alphanumeric (hyphens and underscores allowed)
  - OS must be valid: "linux", "windows", "darwin", or "any"
  - Arch must be valid: "amd64", "arm64", or "any"
  - Format must be valid: "tar", "tar+gzip", or "zip"
  - All artifact file paths must exist within workspace, be readable, and not empty
- **Flexible field validation**:
  - Configuration definitions use flexible map structure
  - Metadata uses flexible map structure
  - Allows varied data shapes for different agent types
- **Validation timing**: Most validation happens at load time with clear error messages
- **Error handling**: Partial failures are tolerated (warns and continues for optional features)

### Schema Handling
- Schema files are **optional** - warnings issued if missing but action continues
- When present, schemas are automatically loaded and **base64-encoded**
- Original relative paths (e.g., `./schemas/config.json`) are **replaced** with base64 content
- Empty schema files are rejected
- JSON validation performed on schema content

### Agent Control Handling
- Reads from single file: `.fleetControl/agentControlDefinitions.yml`
- Contains array of agent control definitions
- Each definition references a content file path (relative to `.fleetControl`)
- Content files are read and base64-encoded separately
- **Optional**: Warnings issued if files missing or fail to load, but action continues
- Empty content files are rejected
- Invalid content paths (with `..`) are rejected

### OCI Binary Upload Handling (optional feature)
- **Completely optional**: Skipped entirely if `oci-registry` input is not provided
- **Only runs in agent flow**: Docs flow never triggers binary uploads
- **Artifact configuration via JSON**: `binaries` input specifies what to upload
- **Multi-platform support**: Each artifact can target specific OS/arch combinations
- **Validation before upload**: All artifact paths validated before any uploads begin
  - If validation fails, no uploads occur and error is returned
  - Prevents partial uploads of invalid artifact sets
- **Sequential uploads**: Artifacts uploaded one at a time to avoid resource exhaustion
- **Manifest index creation**: After all uploads succeed, creates multi-platform index
  - Index tagged with version for easy discovery (e.g., `v1.2.3`)
  - Contains references to all uploaded manifests with platform metadata
- **Error handling**: If any artifact upload fails, entire operation fails
  - Individual failures logged with details (digest, size, error)
  - Success cases also logged with digest and size for verification
- **OCI standards compliance**: Uses proper OCI manifest format and media types
  - Artifacts stored as OCI image layers
  - Includes standard annotations (title, version, created timestamp)
  - Compatible with any OCI-compliant registry

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
- Maintain high test coverage (95%+ overall, 100% for critical packages)
- Use parameterized/table-driven tests to avoid duplication
- Test both success and error paths comprehensively
  - Error scenarios include: invalid field types, missing files, empty files, malformed YAML, directory traversal attacks
  - Security validations are thoroughly tested (path traversal, workspace boundary enforcement)
  - Graceful degradation paths are verified (warnings logged, execution continues)
- Mock external dependencies (GitHub API, HTTP clients, file system where needed)
- Use descriptive test names that explain what is being tested

**Test Patterns**:
- **Parameterized tests**: Group related test cases into single test functions with subtests
  - Example: `TestReadConfigurationDefinitions_ErrorCases` with multiple sub-cases for different error scenarios
  - Example: `TestReadConfigurationDefinitions_InvalidFieldTypes` tests schema field with different invalid types (number, boolean, object, nil, empty string)
  - Example: `TestLoadAndEncodeFile_PathValidation` tests various directory traversal attack patterns
  - Reduces code duplication and makes it easier to add new test cases
  - Each test case clearly labeled with `name` field for easy identification
- **Table-driven tests**: Use struct slices to define test cases with inputs and expected outputs
  - Common pattern: `tests := []struct { name string; input X; expected Y; expectedErr string }{...}`
  - Uses `expectedWarning` and `expectedDebug` fields to verify specific log output patterns
- **Setup functions**: Complex test scenarios use setup functions that return test fixtures
  - Example: `setupFunc func(t *testing.T, tmpDir string) string` for preparing test files
  - Example: `setupFunc func(t *testing.T) (workspace string, filePath string)` for multi-value returns
- **Mock patterns**: Use function variables and interfaces for dependency injection
  - Example: `github.GetChangedMDXFilesFunc` allows tests to mock GitHub API calls
  - HTTP client tests use custom `RoundTripper` implementations for error scenarios

**Package-Specific Testing**:
- **models**:
  - `models.go`: Agent metadata data structure validation, custom unmarshalers, JSON parsing
  - `binary.go`: OCI artifact validation (name format, OS/arch values, format values, duplicate names), OCIConfig validation (registry requirements, empty validation), helper methods (media types, platform strings, filenames)
- **loader** (95%+ coverage):
  - `definitions.go`: Comprehensive error scenario testing with parameterized tests
    - `ReadConfigurationDefinitions`: File not found, invalid YAML, empty arrays, schema loading warnings, directory traversal protection, invalid field types (non-string schema fields), multiple configs
    - `ReadAgentControlDefinitions`: File not found, invalid YAML, empty arrays, content loading warnings, directory traversal protection, invalid field types (non-string content fields), multiple definitions
    - `readDefinitionsFile`: Items not maps, no array found in YAML
    - `loadAndEncodeFile`: Path validation (directory traversal with `..`, paths outside workspace, empty files)
  - `metadata`: MDX file parsing, version validation, subject mapping to agent types, error cases for missing/invalid fields
- **oci**: OCI registry integration (optional feature)
  - `annotations.go`: Layer and manifest annotation creation
  - `validation.go`: Binary path validation (directory traversal, workspace boundaries, file existence, empty files, directories rejected)
  - `handler.go`: Upload orchestration, error handling, manifest index creation
  - Integration tests with local OCI registry (localhost:5000)
- **client**: HTTP client behavior, authentication, error responses, network failures
- **github**: Git diff parsing, SHA validation (prevents command injection), MDX file filtering
- **parser**: MDX frontmatter extraction, YAML parsing, subject-to-agent-type mapping
- **main**: Flow orchestration, environment validation, error handling for both agent and docs flows
  - `TestRunAgentFlow_AgentControlDefinitionsError`: Verifies graceful degradation when agent control definitions fail to load (warns but continues)

**Test Utilities** (`internal/testutil`):
- `CaptureOutput()`: Captures stdout/stderr for testing log output and warnings
- Used extensively to verify `::warn::`, `::debug::`, and `::notice::` GitHub Actions annotations

**Coverage Goals**:
- Critical paths: 100% (authentication, data sending, security validations)
- Business logic: 95%+ (loaders, parsers, validators)
  - `ReadConfigurationDefinitions`: 100%
  - `ReadAgentControlDefinitions`: 100%
  - `readDefinitionsFile`: 94%+
  - `loadAndEncodeFile`: 81%+ (uncovered lines are OS-level errors like `filepath.Abs` failures)
- Overall: 95%+ maintained
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
