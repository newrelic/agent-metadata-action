<a href="https://opensource.newrelic.com/oss-category/#community-project"><picture><source media="(prefers-color-scheme: dark)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/dark/Community_Project.png"><source media="(prefers-color-scheme: light)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Project.png"><img alt="New Relic Open Source community project banner." src="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Project.png"></picture></a>

# Agent Metadata Action

A GitHub Action that reads agent configuration metadata from a checked-out repository. This action parses the `.fleetControl/configurationDefinitions.yml` file and makes the configuration data available for downstream workflow steps.

## Installation

Add this action to your workflow after checking out your repository:

```yaml
- name: Checkout repository
  uses: actions/checkout@v4

- name: Read agent metadata
  uses: newrelic/agent-metadata-action@v1
```

## Usage

This action reads the `.fleetControl/configurationDefinitions.yml` file from your repository and outputs the configuration definitions. The action expects the file to be present after the repository has been checked out.

### Example Workflow For Releasing a New Agent Version

```yaml
name: Process Agent Metadata
on:
  push:
    branches: [main]

jobs:
  read-metadata:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout repository
        uses: actions/checkout@v4
        with:
          ref: v1.0.0 # the release tag for the version being released

      - name: Read agent metadata
        uses: newrelic/agent-metadata-action@v1
        with:
          version: 1.0.0 # only required if different from ref tag
          cache: true  # Optional: Enable Go build cache (default: true)
```

### Example Workflow For Updating Docs Metadata on an Existing Agent Version

The docs header contains things like a bug fix list, a list of features, etc for the given agent version. These
will be updated after the initial agent version has been created so checking out the agent repo is not required.

```yaml
name: Process Agent Metadata
on:
  push:
    branches: [main]

jobs:
  read-metadata:
    runs-on: ubuntu-latest
    steps:
      - name: Read agent metadata
        uses: newrelic/agent-metadata-action@v1
        with:
          version: 1.0.0 # required in the docs case
          cache: true  # Optional: Enable Go build cache (default: true)
```

### Configuration File Format

The action expects a YAML file at `.fleetControl/configurationDefinitions.yml` with the following structure:

```yaml
configurationDefinitions:
  - name: "Configuration Name"
    slug: "config-slug"
    platform: "kubernetes"  # or "host"
    description: "Description of the configuration"
    type: "config-type"
    version: "1.0.0"
    format: "json"
    schema: "./schemas/config-schema.json"
```

**All fields are required.** The action validates each configuration entry and will fail with a clear error message if any required field is missing.

**Schema files** are automatically base64-encoded and embedded in the output. Schema paths must be relative to the `.fleetControl` directory and cannot use directory traversal (`..`) for security.

## Building

To build the action locally:

```bash
# Build the binary
go build -o agent-metadata-action ./cmd/agent-metadata-action

# Run locally (requires GITHUB_WORKSPACE environment variable)
export GITHUB_WORKSPACE=/path/to/your/repo
./agent-metadata-action
```

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for a specific package
go test -v ./internal/config

# Run a specific test
go test -v -run TestLoadEnv_Success ./internal/config
```

## Support

New Relic hosts and moderates an online forum where you can interact with New Relic employees as well as other customers to get help and share best practices. Like all official New Relic open source projects, there's a related Community topic in the New Relic Explorers Hub. You can find this project's topic/threads here:

>Add the url for the support thread here: discuss.newrelic.com

## Contribute

We encourage your contributions to improve Agent Metadata Action! Keep in mind that when you submit your pull request, you'll need to sign the CLA via the click-through using CLA-Assistant. You only have to sign the CLA one time per project.

If you have any questions, or to execute our corporate CLA (which is required if your contribution is on behalf of a company), drop us an email at opensource@newrelic.com.

**A note about vulnerabilities**

As noted in our [security policy](../../security/policy), New Relic is committed to the privacy and security of our customers and their data. We believe that providing coordinated disclosure by security researchers and engaging with the security community are important means to achieve our security goals.

If you believe you have found a security vulnerability in this project or any of New Relic's products or websites, we welcome and greatly appreciate you reporting it to New Relic through [our bug bounty program](https://docs.newrelic.com/docs/security/security-privacy/information-security/report-security-vulnerabilities/).

If you would like to contribute to this project, review [these guidelines](./CONTRIBUTING.md).

To all contributors, we thank you! Without your contribution, this project would not be what it is today.

## License

Agent Metadata Action is licensed under the [Apache 2.0](http://apache.org/licenses/LICENSE-2.0.txt) License.
