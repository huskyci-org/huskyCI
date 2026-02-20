<p align="center">
  <img src="https://raw.githubusercontent.com/huskyci-org/huskyCI/refs/heads/main/huskyCI-logo.png" align="center" height="" />
  <!-- logo font: Anton -->
</p>

---
## IMPORTANT ORIENTATIONS FOR THIS FORK

This is a fork from this [original HuskyCI](github.com/globocom/huskyCI) repository. I've had a few problems when trying to make use of Husky so I decided to improve the code in order to be able to utilize it in production-ready critical environments.
The problems addressed until now were:

- Outdated MongoDB package/driver - Solution: [new MongoDB driver utilized](go.mongodb.org/mongo-driver) with support to the most up to date MongoDB versions
- Broken Kubernetes Compatibility - Solution: fully functioning [HuskyCI API Server helm chart](https://github.com/huskyci-org/helm-chart-huskyci-api) 
- Broken Sonarqube Integration Issue Importing - Solution: Updated [generic exxternal issues importing standard](https://docs.sonarsource.com/sonarqube-server/latest/analyzing-source-code/importing-external-issues/generic-issue-import-format/)
- Gitlab only integration configuration - Set up a fully functioning [Github Action workflow file]()
- Outdated tests - Every test container was updated and a new dockerhub repository was created for each one of them (also a [new organization](https://hub.docker.com/orgs/huskyciorg/repositories) was created)

### Recent improvements

- **Docker-in-Docker / Docker Compose**: API uses `HUSKYCI_DOCKERAPI_ADDR` (and optional `HUSKYCI_DOCKERAPI_PORT`) so security tests run in the correct Docker daemon; avoids "lookup /var/run/docker.sock" when the DB or config had a Unix socket path.
- **File upload (file://) analysis**: Robust zip extraction in the Docker API container (retries for large uploads), gitauthors skipped for file:// URLs, and resilient handling of empty/invalid tool output so analyses can complete.
- **Stability**: Buffered error channels in security test orchestration to avoid "send on closed channel" panics; Docker client created with explicit host to avoid races when multiple tests run in parallel.

### Ongoing activities

- Fix Sonarqube integration file output of some specific tests (like npmaudit, which has only vulnerabilities based on one file and the filepath of the analysed file comes as a placeholder)
- Removing TFSec in favour of [Trivy](https://github.com/aquasecurity/trivy) (TFSec was integrated into Trivy)
- Documentation improvement

### Tips

If you're setting a Github and Kubernetes environment, think about using the [Actions Runner Controller](https://github.com/actions/actions-runner-controller/tree/master) in order to have a highly-available testing pipeline.

---

## Overview

HuskyCI is an open-source tool designed to orchestrate security tests within CI pipelines, centralizing results into a database for further analysis and metrics. It supports multiple programming languages and integrates with popular static analysis tools to identify vulnerabilities early in the development lifecycle.

### Key Features

- **Multi-language support**: Python, Ruby, JavaScript, Golang, Java, HCL, and more.
- **Secrets detection**: Audit repositories for sensitive information like AWS keys and SSH private keys.
- **Integration-ready**: Works seamlessly with CI/CD pipelines like GitLab CI/CD.
- **Extensible**: Add new tools and customize configurations.
- **Infrastructure support**: Compatible with Docker and Kubernetes.

---

## Installation

### Prerequisites

- **Docker** and **Docker Compose** installed.
- **Golang** installed (for development purposes).

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

### Running a Security Analysis

1. Set up environment variables:

   ```bash
   export HUSKYCI_CLIENT_REPO_URL="https://github.com/huskyci-org/huskyCI.git"
   export HUSKYCI_CLIENT_REPO_BRANCH="master"
   export HUSKYCI_CLIENT_API_ADDR="http://localhost:8888"
   export HUSKYCI_CLIENT_API_USE_HTTPS="false"
   export HUSKYCI_CLIENT_TOKEN="{YOUR_TOKEN_HERE}"
   ```

2. Run the HuskyCI client:

   ```bash
   make run-client
   ```

3. View results in the terminal.

### Integrating with CI/CD

Refer to the [integration guide](https://github.com/huskyci-org/huskyCI/wiki/4.-Guides.md) for detailed instructions on adding HuskyCI to your CI/CD pipeline.

---

## Documentation

Comprehensive documentation is available in the [HuskyCI Wiki](https://github.com/huskyci-org/huskyCI/wiki). It includes:

- [Getting Started](https://github.com/huskyci-org/huskyCI/wiki/3.-Getting-Started.md)
- [API Reference](https://github.com/huskyci-org/huskyCI/wiki/5.-API.md)
- [Integration Guides](https://github.com/huskyci-org/huskyCI/wiki/4.-Guides.md)

For local development and testing:
- [Local API Deployment and CLI Testing Guide](LOCAL_DEPLOYMENT.md) - Complete guide for deploying the API server locally and performing CLI tests

---

## Contributing

We welcome contributions! Please read our [contributing guide](https://github.com/huskyci-org/huskyCI/blob/master/CONTRIBUTING.md) to learn about our development process and how to propose changes.

---

## License

HuskyCI is licensed under the [BSD 3-Clause License](https://github.com/huskyci-org/huskyCI/blob/master/LICENSE.md).
