# huskyCI CLI Documentation

## Table of Contents

1. [Overview](#overview)
2. [Installation](#installation)
3. [Configuration](#configuration)
4. [Commands](#commands)
5. [Authentication](#authentication)
6. [Target Management](#target-management)
7. [Running Security Analysis](#running-security-analysis)
8. [Environment Variables](#environment-variables)
9. [Error Handling](#error-handling)
10. [Examples](#examples)
11. [Troubleshooting](#troubleshooting)

---

## Overview

huskyCI CLI is a command-line tool that orchestrates security tests and centralizes results for analysis. It provides a simple interface to interact with huskyCI API endpoints and perform security scans on local codebases.

### Supported Security Tests

The CLI supports static security analysis for:

- **Python**: Bandit and Safety
- **Ruby**: Brakeman
- **JavaScript**: Npm Audit and Yarn Audit
- **Golang**: Gosec
- **Java**: SpotBugs plus Find Sec Bugs
- **C#**: Security Code Scan
- **HCL**: TFSec (Terraform)
- **Infrastructure**: Trivy
- **Generic**: GitLeaks (secrets detection)

### Key Features

- Multiple API target management
- GitHub OAuth device flow authentication
- Local directory security scanning
- Language detection and analysis
- Compressed code upload to API
- Real-time analysis status monitoring
- Vulnerability reporting

---

## Installation

### Prerequisites

- Go 1.19 or higher
- Access to a huskyCI API instance

### Building from Source

```bash
# Clone the repository
git clone https://github.com/huskyci-org/huskyCI.git
cd huskyCI

# Build the CLI
make build-cli

# The binary will be created at: cli/huskyci-cli-bin
```

### Installing Globally

```bash
# Copy to your PATH (example for macOS/Linux)
cp cli/huskyci-cli-bin /usr/local/bin/huskyci
chmod +x /usr/local/bin/huskyci
```

---

## Configuration

### Configuration File Location

The CLI uses a YAML configuration file located at:

```
$HOME/.huskyci/config.yaml
```

### Configuration File Structure

```yaml
targets:
  production:
    current: true
    endpoint: https://api.huskyci.example.com
    token-storage: file
  staging:
    current: false
    endpoint: https://staging-api.huskyci.example.com
    token-storage: file
```

### Configuration File Creation

The configuration file and directory are automatically created on first use. The CLI will:

1. Create `$HOME/.huskyci/` directory if it doesn't exist
2. Create `config.yaml` file if it doesn't exist
3. Display messages about configuration creation

### Custom Configuration File

You can specify a custom configuration file using the `--config` flag:

```bash
huskyci --config /path/to/custom/config.yaml target-list
```

---

## Commands

### Global Flags

All commands support the following global flag:

- `--config string`: Specify a custom config file (default: `$HOME/.huskyci/config.yaml`)

### Command: `huskyci`

**Description**: Root command that displays help information.

**Usage**:
```bash
huskyci [command] [flags]
```

**Examples**:
```bash
huskyci --help
huskyci run --help
```

---

### Command: `huskyci version`

**Description**: Display the huskyCI CLI version information.

**Usage**:
```bash
huskyci version
```

**Output**:
```
huskyCI CLI version: 0.12
For more information, visit: https://github.com/huskyci-org/huskyCI
```

**Behavior**:
- Prints version information
- Always exits successfully (exit code 0)

---

### Command: `huskyci login`

**Description**: Authenticate with GitHub using OAuth device flow.

**Usage**:
```bash
huskyci login
```

**Behavior**:

1. **Initiation**: 
   - Connects to GitHub's device flow API endpoint
   - Requests device and user verification codes
   - Displays user code and verification URL

2. **User Interaction**:
   - Attempts to open browser automatically to `https://github.com/login/device`
   - If browser opening fails, displays manual URL
   - Prompts user to enter the user code in the browser
   - Waits for user to press Enter after authorization

3. **Token Exchange**:
   - Polls GitHub API for access token
   - Verifies authorization completion

4. **Token Storage**:
   - Saves access token to `.huskyci` file in current directory
   - File permissions: `0600` (read/write for owner only)

**Output Example**:
```
🔐 Starting GitHub authentication...

📱 User code: WDJB-MJHT
🌐 Opening browser to: https://github.com/login/device

Please:
  1. Enter the user code shown above in the browser
  2. Authorize the application
  3. Press Enter here to continue...

Press Enter when done...

⏳ Verifying authorization...
✓ Login successful! 🚀

Your access token has been saved.
```

**Error Handling**:
- **404 Error**: Indicates device flow is not enabled for the GitHub App
  - Error message includes link to GitHub App settings
- **Network Errors**: Provides tips to check internet connection
- **Authorization Failures**: Prompts user to ensure authorization was completed

**Prerequisites**:
- GitHub App must have device flow enabled
- Internet connection required
- Browser access for authorization

**Token File**:
- Location: `.huskyci` in current working directory
- Format: Plain text access token
- Permissions: `0600` (owner read/write only)

---

### Command: `huskyci target-add`

**Description**: Add a new huskyCI API endpoint to the target list.

**Usage**:
```bash
huskyci target-add <name> <endpoint> [flags]
```

**Arguments**:
- `name` (required): Target name (letters, numbers, underscores only)
- `endpoint` (required): Full API URL (must include scheme and host)

**Flags**:
- `--set-current, -s`: Add and set as current target immediately

**Validation**:
- **Target Name**: Must match regex `^\w+$` (letters, numbers, underscores)
- **Endpoint URL**: Must be valid URL with scheme (`http://` or `https://`) and host
- **Duplicate Check**: Prevents adding targets with existing names

**Behavior**:
1. Validates target name format
2. Validates endpoint URL format
3. Checks for duplicate target names
4. If `--set-current` is used, unsets all other targets
5. Adds target to configuration
6. Saves configuration to file

**Examples**:
```bash
# Add a production target
huskyci target-add production https://api.huskyci.example.com

# Add and set as current
huskyci target-add staging https://staging-api.huskyci.example.com --set-current

# Add local development target
huskyci target-add local http://localhost:8888
```

**Success Output**:
```
✓ Successfully added target 'production' -> https://api.huskyci.example.com (set as current)
```

**Error Scenarios**:
- Invalid target name format
- Invalid URL format
- Duplicate target name
- Configuration file write errors

---

### Command: `huskyci target-list`

**Description**: List all configured huskyCI API targets.

**Usage**:
```bash
huskyci target-list
```

**Behavior**:
- Reads targets from configuration file
- Displays all configured targets
- Marks current target with asterisk (`*`)
- Shows target name and endpoint URL

**Output Format**:
```
Configured targets:

  * production (https://api.huskyci.example.com)
    staging (https://staging-api.huskyci.example.com)
    local (http://localhost:8888)

Legend: * = current target
```

**Empty State**:
If no targets are configured:
```
No targets configured.

Tip: Use 'huskyci target-add <name> <endpoint>' to add a new target
Example: huskyci target-add production https://api.huskyci.example.com
```

---

### Command: `huskyci target-remove`

**Description**: Remove a target from the target list.

**Usage**:
```bash
huskyci target-remove <name>
```

**Arguments**:
- `name` (required): Name of target to remove

**Behavior**:
1. Validates target exists
2. Removes target from configuration
3. Saves updated configuration
4. Displays confirmation message

**Examples**:
```bash
# Remove staging target
huskyci target-remove staging

# Remove old production target
huskyci target-remove old-production
```

**Success Output**:
```
✓ Successfully removed target 'staging' (https://staging-api.huskyci.example.com) from target list
```

**Error Scenarios**:
- Target does not exist
- Configuration file write errors

---

### Command: `huskyci target-set`

**Description**: Set a target as the current active target.

**Usage**:
```bash
huskyci target-set <name>
```

**Arguments**:
- `name` (required): Name of target to set as current

**Behavior**:
1. Validates target exists
2. Sets specified target as current (`current: true`)
3. Unsets all other targets (`current: false`)
4. Saves configuration

**Examples**:
```bash
# Set production as current
huskyci target-set production

# Switch to staging
huskyci target-set staging
```

**Success Output**:
```
✓ Successfully set 'production' (https://api.huskyci.example.com) as the current target
```

**Error Scenarios**:
- Target does not exist
- Configuration file write errors

---

### Command: `huskyci run`

**Description**: Run a security analysis on a local directory.

**Usage**:
```bash
huskyci run <path>
```

**Arguments**:
- `path` (required): Path to directory or file to analyze

**Behavior**:

1. **Path Validation**:
   - Resolves absolute path
   - Checks path exists
   - Validates path is accessible

2. **Language Detection**:
   - Scans directory recursively
   - Detects programming languages using file extensions
   - Uses `enry` library for language detection
   - Maps detected languages to available security tests

3. **File Compression**:
   - Collects all allowed files from path
   - Creates ZIP archive
   - Calculates compressed file size
   - Stores archive at `$HOME/.huskyci/compressed-code.zip`

4. **API Communication**:
   - Sends compressed code to huskyCI API
   - Monitors analysis status
   - Retrieves results

5. **Results Display**:
   - Prints detected languages
   - Displays analysis results
   - Shows vulnerabilities found

6. **Cleanup**:
   - Removes temporary ZIP file

**Supported Languages**:
- Go
- Python
- Ruby
- JavaScript
- Java
- C#
- HCL (Terraform)

**Security Tests Mapping**:
- **Go**: `huskyci/gosec`
- **Python**: `huskyci/bandit`, `huskyci/safety`
- **Ruby**: `huskyci/brakeman`
- **JavaScript**: `huskyci/npmaudit`, `huskyci/yarnaudit`
- **Java**: `huskyci/spotbugs`
- **C#**: `huskyci/securitycodescan`
- **HCL**: `huskyci/tfsec`
- **Generic**: `huskyci/gitleaks` (always included)

**Examples**:
```bash
# Analyze current directory
huskyci run .

# Analyze specific directory
huskyci run ./my-project

# Analyze subdirectory
huskyci run ./src/main
```

**Output Example**:
```
🔍 Scanning code from: /path/to/project

📋 Detected languages:
  ✓ Go
  ✓ Python

📦 Compressing code...
✓ Compressed successfully! Size: 2.5 MB

🚀 Sending code to huskyCI API...
✓ Code sent successfully!

⏳ Checking analysis status...
✓ Analysis check completed!

📊 Analysis Results:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

**Error Scenarios**:
- Path does not exist
- No supported languages found
- File compression errors
- API communication errors
- Permission errors

**Prerequisites**:
- Valid target configured (or environment variables set)
- API endpoint accessible
- Authentication token available (if required)

---

## Authentication

### GitHub OAuth Device Flow

The CLI uses GitHub's OAuth 2.0 Device Authorization Grant flow for authentication. This is designed for headless applications like CLI tools.

### Authentication Flow

1. **Initiation**: `huskyci login` command initiates the flow
2. **Device Code**: GitHub provides a device code and user code
3. **User Authorization**: User enters code in browser at `https://github.com/login/device`
4. **Token Exchange**: CLI exchanges device code for access token
5. **Token Storage**: Token saved to `.huskyci` file

### Token Storage

**File Location**: `.huskyci` in current working directory

**File Format**: Plain text access token

**File Permissions**: `0600` (read/write for owner only)

**Note**: The token file is created in the directory where `huskyci login` is executed.

### GitHub App Requirements

- Device flow must be enabled in GitHub App settings
- App must be registered with GitHub
- User must have authorization permissions

### Troubleshooting Authentication

**404 Error**: Device flow not enabled
- Solution: Enable device flow in GitHub App settings at https://github.com/settings/apps

**Network Errors**: Connection issues
- Solution: Check internet connection and firewall settings

**Authorization Timeout**: User didn't complete authorization
- Solution: Re-run `huskyci login` and complete authorization within 15 minutes

---

## Target Management

### Target Configuration

Targets are stored in the configuration file under the `targets` key. Each target has:

- **name**: Unique identifier (letters, numbers, underscores)
- **endpoint**: Full API URL (scheme + host)
- **current**: Boolean flag indicating active target
- **token-storage**: Storage method for authentication token (optional)

### Current Target Selection

Only one target can be current at a time. The current target is used for all API operations.

### Target Priority

1. **Environment Variables**: Highest priority
   - `HUSKYCI_CLIENT_API_ADDR`: API endpoint
   - `HUSKYCI_CLIENT_TOKEN`: Authentication token

2. **Configuration File**: Used if environment variables not set
   - Target marked with `current: true`

### Target Operations

- **Add**: `huskyci target-add <name> <endpoint>`
- **List**: `huskyci target-list`
- **Set Current**: `huskyci target-set <name>`
- **Remove**: `huskyci target-remove <name>`

---

## Running Security Analysis

### Analysis Workflow

1. **Path Scanning**: Detects languages in codebase
2. **File Collection**: Gathers all code files
3. **Compression**: Creates ZIP archive
4. **Upload**: Sends to huskyCI API
5. **Monitoring**: Polls for analysis status
6. **Results**: Displays vulnerabilities found
7. **Cleanup**: Removes temporary files

### Language Detection

Uses file extension analysis via the `enry` library to detect:
- Programming languages
- File types
- Code structure

### File Filtering

The CLI filters files during compression:
- Includes code files
- Excludes binary files
- Excludes common ignore patterns

### Analysis Results

Results include:
- Detected languages
- Security test results per language
- Vulnerability details (severity, file, line, code)
- Summary statistics

---

## Environment Variables

The CLI supports environment variables for configuration, which take precedence over configuration file settings.

### Available Variables

- `HUSKYCI_CLIENT_API_ADDR`: API endpoint URL
- `HUSKYCI_CLIENT_TOKEN`: Authentication token

### Environment Variable Priority

Environment variables override configuration file settings. If `HUSKYCI_CLIENT_API_ADDR` is set, it will be used regardless of the current target in the config file.

### Example Usage

```bash
export HUSKYCI_CLIENT_API_ADDR="https://api.huskyci.example.com"
export HUSKYCI_CLIENT_TOKEN="your-token-here"
huskyci run ./my-project
```

---

## Error Handling

### Error Format

All errors follow a consistent format:

```
[HUSKYCI] ❌ Error: <error message>

Tip: <helpful tip>
For troubleshooting, visit: https://github.com/huskyci-org/huskyCI/wiki
```

### Error Categories

1. **Configuration Errors**:
   - Missing configuration file
   - Invalid configuration format
   - Target not configured

2. **Validation Errors**:
   - Invalid command arguments
   - Invalid target name format
   - Invalid URL format
   - Path does not exist

3. **Network Errors**:
   - API connection failures
   - Timeout errors
   - HTTP errors (404, 500, etc.)

4. **Authentication Errors**:
   - Device flow not enabled
   - Authorization failures
   - Token expiration

5. **File System Errors**:
   - Permission denied
   - Disk space issues
   - File not found

### Exit Codes

- `0`: Success
- `1`: Error occurred

### Error Recovery

Most errors include helpful tips for resolution:
- Suggested commands to fix issues
- Links to documentation
- Troubleshooting steps

---

## Examples

### Complete Workflow

```bash
# 1. Add API target
huskyci target-add production https://api.huskyci.example.com --set-current

# 2. Authenticate
huskyci login

# 3. List targets
huskyci target-list

# 4. Run analysis
huskyci run ./my-project

# 5. Switch targets
huskyci target-set staging
huskyci run ./my-project
```

### Using Environment Variables

```bash
# Set environment variables
export HUSKYCI_CLIENT_API_ADDR="https://api.huskyci.example.com"
export HUSKYCI_CLIENT_TOKEN="gho_your_token_here"

# Run analysis (bypasses config file)
huskyci run ./my-project
```

### Multiple Targets

```bash
# Add multiple targets
huskyci target-add production https://api.huskyci.example.com
huskyci target-add staging https://staging-api.huskyci.example.com
huskyci target-add local http://localhost:8888

# List all targets
huskyci target-list

# Switch between targets
huskyci target-set production
huskyci run ./project

huskyci target-set staging
huskyci run ./project
```

### Custom Configuration

```bash
# Use custom config file
huskyci --config /path/to/config.yaml target-list

# Run with custom config
huskyci --config /path/to/config.yaml run ./project
```

---

## Troubleshooting

### Common Issues

#### 1. "No targets configured"

**Problem**: No targets are set up in configuration.

**Solution**:
```bash
huskyci target-add <name> <endpoint>
```

#### 2. "Device flow may not be enabled"

**Problem**: GitHub App doesn't have device flow enabled.

**Solution**:
1. Go to https://github.com/settings/apps
2. Select your app
3. Enable "Device Flow" feature
4. Save changes

#### 3. "Path does not exist"

**Problem**: Invalid path provided to `run` command.

**Solution**: Verify path exists and is accessible:
```bash
ls -la /path/to/directory
huskyci run /path/to/directory
```

#### 4. "No supported programming languages found"

**Problem**: Directory doesn't contain supported languages.

**Solution**: Ensure directory contains code in supported languages (Go, Python, Ruby, JavaScript, Java, C#, HCL).

#### 5. "Failed to initiate authentication"

**Problem**: Network or GitHub API issues.

**Solution**:
- Check internet connection
- Verify GitHub API is accessible
- Ensure device flow is enabled for GitHub App

#### 6. Configuration File Errors

**Problem**: Cannot read or write configuration file.

**Solution**:
- Check file permissions: `chmod 644 ~/.huskyci/config.yaml`
- Verify directory exists: `ls -la ~/.huskyci/`
- Check disk space: `df -h ~`

#### 7. Token File Issues

**Problem**: Cannot save authentication token.

**Solution**:
- Check write permissions in current directory
- Verify disk space available
- Ensure `.huskyci` file doesn't exist with wrong permissions

### Debugging

#### Verbose Output

Some commands provide additional information:
- Configuration file location is displayed on startup
- Target operations show confirmation messages
- Analysis shows progress at each step

#### Configuration Inspection

Check configuration file directly:
```bash
cat ~/.huskyci/config.yaml
```

#### Environment Variable Check

Verify environment variables:
```bash
echo $HUSKYCI_CLIENT_API_ADDR
echo $HUSKYCI_CLIENT_TOKEN
```

### Getting Help

- Command help: `huskyci --help` or `huskyci <command> --help`
- Documentation: https://github.com/huskyci-org/huskyCI/wiki
- GitHub Issues: https://github.com/huskyci-org/huskyCI/issues

---

## Additional Notes

### File Locations

- **Config Directory**: `$HOME/.huskyci/`
- **Config File**: `$HOME/.huskyci/config.yaml`
- **Compressed Code**: `$HOME/.huskyci/compressed-code.zip` (temporary)
- **Token File**: `.huskyci` (in current directory, created by `login` command)

### Security Considerations

- Token files have restricted permissions (`0600`)
- Configuration files may contain sensitive information
- Tokens should be kept secure and not shared
- Use environment variables in CI/CD environments

### Performance

- Language detection scans entire directory tree
- Compression time depends on codebase size
- API communication depends on network speed
- Analysis time depends on API processing

### Limitations

- Token file is stored in current directory (not config directory)
- Only one target can be current at a time
- Language detection based on file extensions
- Requires internet connection for API operations

---

## Version Information

Current CLI version: **0.12**

For the latest version and updates, visit: https://github.com/huskyci-org/huskyCI

---

## Support

- **Documentation**: https://github.com/huskyci-org/huskyCI/wiki
- **Issues**: https://github.com/huskyci-org/huskyCI/issues
- **Discussions**: https://github.com/huskyci-org/huskyCI/discussions

---

*Last updated: January 2026*
