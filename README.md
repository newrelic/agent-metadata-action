<a href="https://opensource.newrelic.com/oss-category/#community-project"><picture><source media="(prefers-color-scheme: dark)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/dark/Community_Project.png"><source media="(prefers-color-scheme: light)" srcset="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Project.png"><img alt="New Relic Open Source community project banner." src="https://github.com/newrelic/opensource-website/raw/main/src/images/categories/Community_Project.png"></picture></a>

# Agent Metadata Action

A GitHub Action that reads agent configuration metadata from the calling repository. There are 2 scearios to use this action:
1. An agent release - This action parses the `.fleetControl/configurationDefinitions.yml` file and makes the configuration data and metadata available in New Relic.
2. A docs update for an agent release - This action parses the frontmatter of the docs mdx files and makes the metadata available in New Relic.

## Installation

Add this action to your workflow:

```yaml
- name: Read agent metadata
  uses: newrelic/agent-metadata-action@v1
```

## Usage

### Prerequisites

This action requires OAuth credentials to authenticate with New Relic services. You must configure the following secrets in your repository:

- `OAUTH_CLIENT_ID` - Your OAuth client ID for system identity authentication
- `OAUTH_CLIENT_SECRET` - Your OAuth client secret for system identity authentication (base64 encoded)

These must be passed as action inputs using the `with:` parameter in your workflow.

### Example Workflow For Releasing a New Agent Version
This action automatically checks out your repository at the specified version tag, then reads the `.fleetControl/configurationDefinitions.yml` file and other associated files in `/fleetControl` and saves the agent information in New Relic. 

```yaml
name: Process Agent Metadata
on:
  release:
    types:
      - published

jobs:
  read-metadata:
    runs-on: ubuntu-latest
    steps:
      - name: Read agent metadata
        uses: newrelic/agent-metadata-action@v1
        with:
          newrelic-client-id: ${{ secrets.OAUTH_CLIENT_ID }}
          newrelic-private-key: ${{ secrets.OAUTH_CLIENT_SECRET }}
          agent-type: dotnet-agent # Required for agent release workflow: The type of agent (e.g., nodejs-agent, java-agent)
          version: 1.0.0 # Required for agent release workflow: will be used to check out appropriate release tag
          cache: true  # Optional: Enable Go build cache (default: true)
```

### Example Workflow For Updating Docs Metadata for a new/existing Agent Version
This action should be triggered on a push to the main docs branch. It will automatically detect the changed release notes in the push and save the agent metadata in New Relic.

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
          newrelic-client-id: ${{ secrets.OAUTH_CLIENT_ID }}
          newrelic-private-key: ${{ secrets.OAUTH_CLIENT_SECRET }}
          cache: true  # Optional: Enable Go build cache (default: true)
```

### Configuration File Format (Agent Scenario)

For the agent scenario, the action expects a YAML file at `.fleetControl/configurationDefinitions.yml` with the following structure:

```yaml
configurationDefinitions:
  - platform: "KUBERNETESCLUSTER"  # or "HOST" or "ALL" if there is no distinction
    description: "Description of the configuration"
    type: "agent-config"
    version: "1.0.0" -- config schema version
    format: "yml"   -- format of the agent config file
    schema: "./schemas/config-schema.json"
```


**Dec 2025 - schema temporarily optional until full functionality is ready

**Schema paths must be relative to the `.fleetControl` directory and cannot use directory traversal (`..`) for security.

## Building

```bash
# Build the binary
go build -o agent-metadata-action ./cmd/agent-metadata-action
```

## Testing

Run the test suite:

```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...
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
