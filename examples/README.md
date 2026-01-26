# HuskyCI Configuration Examples

This directory contains example configuration files for HuskyCI.

## Available Examples

### CLI Configuration

**File:** `cli-config.yaml.example`

Example configuration file for the HuskyCI CLI tool. This file demonstrates how to configure multiple HuskyCI API targets (endpoints) and manage authentication tokens.

**Usage:**
```bash
# Copy the example file to your home directory
cp examples/cli-config.yaml.example ~/.huskyci/config.yaml

# Edit the file with your actual configuration
nano ~/.huskyci/config.yaml
```

**Features:**
- Multiple target support (local, staging, production, etc.)
- Token storage configuration (keychain, file, or manual)
- Environment variable alternatives

### Kubernetes Configuration

**File:** `kubernetes-config.yaml.example`

Example Kubernetes kubeconfig file for HuskyCI API server when deployed in Kubernetes mode.

**Usage:**
```bash
# Copy the example file to your desired location
cp examples/kubernetes-config.yaml.example /path/to/your/kubeconfig.yaml

# Edit the file with your Kubernetes cluster details
nano /path/to/your/kubeconfig.yaml
```

**Features:**
- Kubernetes cluster configuration
- Certificate and key management
- Context and user configuration

## Documentation

For more detailed information about HuskyCI configuration, please refer to:
- [HuskyCI Wiki](https://github.com/huskyci-org/huskyCI/wiki)
- [Getting Started Guide](https://github.com/huskyci-org/huskyCI/wiki/3.-Getting-Started.md)
- [API Reference](https://github.com/huskyci-org/huskyCI/wiki/5.-API.md)
