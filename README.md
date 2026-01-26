<p align="center">
  <img src="https://raw.githubusercontent.com/huskyci-org/huskyCI/refs/heads/main/huskyCI-logo.png" align="center" height="" />
  <!-- logo font: Anton -->
</p>

<p align="center">
  <a href="https://github.com/huskyci-org/huskyCI/releases"><img src="https://img.shields.io/github/v/release/huskyci-org/huskyCI"/></a>
  <a href="https://github.com/rafaveira3/writing-and-presentations/blob/master/DEFCON-27-APP-SEC-VILLAGE-Rafael-Santos-huskyCI-Finding-security-flaws-in-CI-before-deploying-them.pdf"><img src="https://img.shields.io/badge/DEFCON%2027-AppSec%20Village-black"/></a>
  <a href="https://github.com/rafaveira3/contributions/blob/master/huskyCI-BlackHat-Europe-2019.pdf"><img src="https://img.shields.io/badge/Black%20Hat%20Europe%202019-Arsenal-black"/></a>
  <a href="https://defectdojo.readthedocs.io/en/latest/integrations.html#huskyci-report"><img src="https://img.shields.io/badge/DefectDojo-Compatible-brightgreen"/></a>
</p>

---
## IMPORTANT ORIENTATIONS FOR THIS FORK

This is a fork from this [original HuskyCI](github.com/globocom/huskyCI) repository. I've had a few problems when trying to make use of Husky so I decided to improve the code in order to be able to utilize it in production-ready critical environments.
The problems addressed until now were:
  - Outdated MongoDB package/driver - Solution: [new MongoDB driver utilized](go.mongodb.org/mongo-driver) with support to the most up to date MongoDB versions
  - Broken Kubernetes Compatibility - Solution: fully functioning [HuskyCI API Server helm chart](https://github.com/huskyci-org/helm-chart-huskyci-api) 
  - Broken Sonarqube Integration Issue Importing - Solution: Updated [generic external issues importing standard](https://docs.sonarsource.com/sonarqube-server/latest/analyzing-source-code/importing-external-issues/generic-issue-import-format/)
  - Gitlab only integration configuration - Set up a fully functioning [GitHub Actions workflow file](.github/workflows/e2e-tests.yml) for end-to-end testing
  - Outdated tests - Every test container was updated and a new dockerhub repository was created for each one of them (also a [new organization](https://hub.docker.com/orgs/huskyciorg/repositories) was created)

### Ongoing activities
  - Fix Sonarqube integration file output of some specific tests (like npmaudit, which has only vulnerabilities based on one file and the filepath of the analysed file comes as a placeholder)
  - Transitioning from TFSec to [Trivy](https://github.com/aquasecurity/trivy) for infrastructure scanning (TFSec functionality is now available in Trivy)
  - Documentation improvement

### Tips

If you're setting a Github and Kubernetes environment, think about using the [Actions Runner Controller](https://github.com/actions/actions-runner-controller/tree/master) in order to have a highly-available testing pipeline.

---
## Overview

HuskyCI is an open-source tool designed to orchestrate security tests within CI pipelines, centralizing results into a database for further analysis and metrics. It supports multiple programming languages and integrates with popular static analysis tools to identify vulnerabilities early in the development lifecycle.

### Key Features

- **Multi-language support**: Python, Ruby, JavaScript, Golang, Java, C#, and infrastructure-as-code (HCL).
- **Comprehensive security testing**: 
  - **Python**: Bandit (static analysis) and Safety (dependency checking)
  - **Ruby**: Brakeman (static analysis)
  - **JavaScript**: NpmAudit and YarnAudit (dependency vulnerability scanning)
  - **Go**: Gosec (static analysis)
  - **Java**: SpotBugs with FindSecBugs plugin (static analysis)
  - **C#**: SecurityCodeScan (static analysis)
  - **Infrastructure**: Trivy (container and infrastructure scanning)
- **Secrets detection**: Audit repositories for sensitive information like AWS keys, SSH private keys, and other secrets using Gitleaks.
- **Integration-ready**: Works seamlessly with CI/CD pipelines including GitHub Actions and GitLab CI/CD.
- **SonarQube integration**: Export results in SonarQube Generic Issue Import Format for centralized reporting.
- **Multiple database support**: MongoDB (default) and PostgreSQL.
- **Infrastructure support**: Compatible with Docker and Kubernetes deployments.
- **CLI tool**: Command-line interface for managing targets and running analyses locally.

---

## Installation

### Prerequisites

- **Docker** and **Docker Compose** installed.
- **Golang 1.23+** installed (for development purposes).
  - API module requires Go 1.24.0
  - CLI and Client modules require Go 1.23.0+

### Steps

1. Clone the repository:

   ```bash
   git clone https://github.com/huskyci-org/huskyCI.git
   cd huskyCI
   ```

2. Install dependencies and set up the environment:

   ```bash
   make install
   ```

3. Start the HuskyCI environment:

   ```bash
   make compose
   ```

---

## Usage

### Using the CLI Tool

The HuskyCI CLI provides an easy way to interact with HuskyCI from your local machine.

#### Installation

Build the CLI:

```bash
make build-cli
```

#### Available Commands

- **`huskyci run <path>`**: Run a security analysis on a local directory
- **`huskyci login`**: Authenticate with GitHub using device flow
- **`huskyci target-add <name> <endpoint>`**: Add a new HuskyCI API target
- **`huskyci target-list`**: List all configured targets
- **`huskyci target-set <name>`**: Set the current target
- **`huskyci target-remove <name>`**: Remove a target
- **`huskyci version`**: Display version information

#### Example: Running an Analysis

1. Configure your target (if not already done):

   ```bash
   huskyci target-add local http://localhost:8888 --set-current
   ```

2. Run an analysis on a local directory:

   ```bash
   huskyci run /path/to/your/project
   ```

### Using the Client

1. Set up environment variables:

   ```bash
   export HUSKYCI_CLIENT_REPO_URL="https://github.com/huskyci-org/huskyCI.git"
   export HUSKYCI_CLIENT_REPO_BRANCH="main"
   export HUSKYCI_CLIENT_API_ADDR="http://localhost:8888"
   export HUSKYCI_CLIENT_API_USE_HTTPS="false"
   export HUSKYCI_CLIENT_TOKEN="{YOUR_TOKEN_HERE}"
   ```

2. Run the HuskyCI client:

   ```bash
   make run-client
   ```

   Or with JSON output:

   ```bash
   make run-client-json
   ```

3. View results in the terminal.

### API Endpoints

The HuskyCI API provides REST endpoints for integration:

- **`POST /analysis`**: Submit a new security analysis
- **`GET /analysis/:id`**: Get analysis results by RID
- **`GET /healthcheck`**: Health check endpoint
- **`GET /version`**: Get API version
- **`GET /stats/:metric_type`**: Get statistics
- **`POST /api/1.0/token`**: Generate authentication token (requires basic auth)
- **`PUT /user`**: Update user information

### Integrating with CI/CD

Refer to the [integration guide](https://github.com/huskyci-org/huskyCI/wiki/4.-Guides.md) for detailed instructions on adding HuskyCI to your CI/CD pipeline. HuskyCI supports:

- **GitHub Actions**: Workflow examples available
- **GitLab CI/CD**: Native integration support
- **Kubernetes**: Helm chart available at [huskyci-org/helm-chart-huskyci-api](https://github.com/huskyci-org/helm-chart-huskyci-api)

---

## Architecture

HuskyCI consists of three main components:

1. **API Server** (`api/`): The core backend service that orchestrates security tests, manages analysis, and stores results in the database.
2. **CLI Tool** (`cli/`): Command-line interface for local development and interaction with HuskyCI API.
3. **Client** (`client/`): Lightweight client for submitting analyses and retrieving results.

### Infrastructure Support

- **Docker**: Full support for running security tests in Docker containers
- **Kubernetes**: Native Kubernetes support with configurable deployment options
- **Database**: 
  - MongoDB (default, fully supported)
  - PostgreSQL (supported, requires configuration)

### Security Tests

HuskyCI orchestrates the following security testing tools:

| Tool | Language/Type | Purpose |
|------|--------------|---------|
| Bandit | Python | Static security analysis |
| Safety | Python | Dependency vulnerability scanning |
| Brakeman | Ruby | Static security analysis |
| Gosec | Go | Static security analysis |
| NpmAudit | JavaScript | NPM dependency vulnerability scanning |
| YarnAudit | JavaScript | Yarn dependency vulnerability scanning |
| SpotBugs | Java | Static security analysis with FindSecBugs |
| SecurityCodeScan | C# | Static security analysis |
| Gitleaks | Generic | Secrets detection |
| Trivy | Generic | Infrastructure and container scanning |
| Enry | Generic | Language detection |
| GitAuthors | Generic | Commit author analysis |

## Documentation

Comprehensive documentation is available in the [HuskyCI Wiki](https://github.com/huskyci-org/huskyCI/wiki). It includes:

- [Getting Started](https://github.com/huskyci-org/huskyCI/wiki/3.-Getting-Started.md)
- [API Reference](https://github.com/huskyci-org/huskyCI/wiki/5.-API.md)
- [Integration Guides](https://github.com/huskyci-org/huskyCI/wiki/4.-Guides.md)

---

## Development

### Building from Source

Build all components:

```bash
make build-all
```

Build individual components:

```bash
make build-api      # Build API server
make build-client   # Build client
make build-cli      # Build CLI tool
```

### Running Tests

Run all unit tests:

```bash
make test
```

Run end-to-end tests:

```bash
make test-e2e
```

### Development Environment

The project uses Go workspaces. The workspace file (`go.work`) defines the modules:

- `api/` - API server module
- `cli/` - CLI tool module  
- `client/` - Client module

### Security Testing

Run security static analysis on the codebase:

```bash
make check-sec
```

## Contributing

We welcome contributions! Please read our [contributing guide](CONTRIBUTING.md) to learn about our development process and how to propose changes.

---

## License

HuskyCI is licensed under the [BSD 3-Clause License](https://github.com/huskyci-org/huskyCI/blob/master/LICENSE.md).
