<p align="center">
  <img src="https://raw.githubusercontent.com/wiki/huskyci-org/huskyCI/huskyCI-logo.png" align="center" height="" />
  <!-- logo font: Anton -->
</p>

<p align="center">
  <a href="https://github.com/huskyci-org/huskyCI/releases"><img src="https://img.shields.io/github/v/release/huskyci-org/huskyCI"/></a>
  <a href="https://github.com/rafaveira3/writing-and-presentations/blob/master/DEFCON-27-APP-SEC-VILLAGE-Rafael-Santos-huskyCI-Finding-security-flaws-in-CI-before-deploying-them.pdf"><img src="https://img.shields.io/badge/DEFCON%2027-AppSec%20Village-black"/></a>
  <a href="https://github.com/rafaveira3/contributions/blob/master/huskyCI-BlackHat-Europe-2019.pdf"><img src="https://img.shields.io/badge/Black%20Hat%20Europe%202019-Arsenal-black"/></a>
  <a href="https://defectdojo.readthedocs.io/en/latest/integrations.html#huskyci-report"><img src="https://img.shields.io/badge/DefectDojo-Compatible-brightgreen"/></a>
</p>

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

---

## Contributing

We welcome contributions! Please read our [contributing guide](https://github.com/huskyci-org/huskyCI/blob/master/CONTRIBUTING.md) to learn about our development process and how to propose changes.

---

## License

HuskyCI is licensed under the [BSD 3-Clause License](https://github.com/huskyci-org/huskyCI/blob/master/LICENSE.md).
