# dg — Deployment Guard (Go)

[中文](README-zh.md)

dg is a lightweight and practical deployment guard tool developed in Go, designed to provide reliable guarantees for project deployment processes. It can real-time detect Docker image updates and Git repository changes, trigger the execution of custom scripts through cron scheduled tasks, and has core capabilities such as daily log rotation and PID-based secure concurrency control, helping developers automate deployment management and reduce manual intervention costs.

## Core Features

dg is designed around three core goals: "reliable detection, secure execution, and convenient management", with the following core features:

- **Multi-dimensional Update Detection**: Supports update triggering in three dimensions: Docker images (comparing remote and local image digests), Git branches (detecting head changes), and Git tags (detecting new tags).

- **Ordered Script Execution**: Executes custom scripts in the configured order, uniformly captures stdout/stderr output, and aborts execution and records errors when a script exits with a non-zero code.

- **Intelligent Log Management**: Rotates logs daily in the "YYYY-MM-DD.log" format by default, and supports specifying log retention days through configuration (automatically cleans up outdated logs).

- **Secure Concurrency Control**: Implements concurrency protection through PID semantics in `.dg/state.yml` — PID=0 indicates idle and executable; PID>0 and the corresponding process exists indicates running, and the current execution will be skipped automatically.

- **Flexible Cron Integration**: When installing/uninstalling cron rules, it strictly filters through `-config <absolute path>`, only operates on the scheduled tasks of the target project, and does not affect other rules.

- **Clear CLI Interaction**: Provides 5 core commands: `run` (manually trigger execution), `install` (install cron rules), `uninstall` (uninstall cron rules), `help` (view help), and `version` (view version), with intuitive operations.

- **Secure Credential Handling**: Does not record any secrets. Docker repository authentication calls the system's default Docker keychain (`~/.docker/config.json`) through `go-containerregistry`; Git HTTPS credentials are only used in memory (Note: The constructed URL may be briefly visible in the `ps` process list. It is recommended to use SSH or system credential helpers first).

## Quick Start Guide

The following is the complete process from installation to deployment, suitable for users using dg for the first time.

### 1. Install dg

You can install or update dg to `/usr/local/bin` (system-level path, ensure permission) through a one-click script:

```bash
curl -fsSL https://raw.githubusercontent.com/indiefun/deploy-guard/main/install.sh | bash
```

### 2. Project Configuration

In the root directory of the target project, you need to create a `.dg` directory and configure two core files: `config.yml` (main configuration) and custom scripts (e.g., `script1.sh`).

#### 2.1 Configure config.yml

Creation path: `<project>/.dg/config.yml`. The configuration example is as follows (required/optional fields are marked):

```yaml
# Scheduled execution rule (required, 5 fields: minute hour day month weekday, does not support second-level precision; use an external scheduler if second-level precision is needed)
cron: '*/1 * * * *'        
# Monitoring configuration (enable at least one: docker.images/git.branches/git.tags)
watchs:
  docker:
    # List of Docker images to monitor (example)
    images:                
      - postgres:17
      - redis:8
  git:
    # List of Git branches to monitor (example)
    branches: [main, dev]  
    # Whether to monitor new Git tags (true/false)
    tags: true             
    # Git remote repository name (optional, default: origin)
    remote: origin         
    # Git HTTPS username (optional, only required for HTTPS protocol)
    username: myuser       
    # Git HTTPS password (optional, only required for HTTPS protocol)
    password: mypass       
# List of scripts to execute after detecting updates (required, at least one script, supports absolute/relative paths)
scripts:                  
  - /absolute/path/script1.sh  # Absolute path: directly points to the script
  - ./relative/to/config.yml/dir/script2.sh  # Relative path: relative to the directory where config.yml is located
# Log configuration (optional)
logs:
  # Log retention days (optional, default: no cleanup; logs exceeding the days will be deleted automatically)
  retain_days: 7          
```

**Configuration Rule Explanation**:

- Relative path resolution: All relative paths of scripts, logs, and state files are based on the directory where `config.yml` is located (i.e., `<project>/.dg`).

- Paths of logs and state files:

- Logs: `<project>/.dg/logs/YYYY-MM-DD.log` (rotated by date)

- State: `<project>/.dg/state.yml` (records PID, execution timestamp, and result)

Mandatory verification: The `cron` field, `scripts` list, and at least one monitoring item under `watchs` are mandatory; none of them can be missing.

#### 2.2 Write Custom Scripts

Taking `script1.sh` as an example, creation path: `<project>/.dg/script1.sh`. The script content can be customized according to business needs (example as follows):

```bash
# Example: Output execution completion information
echo 'dg detected an update, script execution completed!'
# In actual scenarios, you can add: image pulling, service restart, deployment verification and other logic
```

### 3. Test Run

After configuration, you can manually trigger a run first to verify if the configuration is correct:

```bash
# Enter the project root directory (or directly specify the config.yml path, e.g., dg run -config /path/to/.dg/config.yml)
cd <project>
# Manually run dg
dg run
# View run logs (verify execution results)
cat .dg/logs/$(date +%Y-%m-%d).log
```

### 4. Install Cron Scheduled Task

After the test passes, configure dg as a cron scheduled task to realize automatic detection and execution:

```bash
# Install cron rules (automatically read the cron expression in config.yml)
dg install
# Verify if the cron rule takes effect (view the current user's cron list)
crontab -l
```

Cron rule format: `<cron expression> /usr/local/bin/dg -config <project>/.dg/config.yml`

### 5. Uninstall Cron Scheduled Task

If you need to stop automatic execution, you can uninstall the corresponding cron rule (only remove the dg task of the current project, without affecting other cron rules):

```bash
# Uninstall cron rules
dg uninstall
# Verify uninstallation result
crontab -l
```

## Development & Advanced Instructions

This section is suitable for developers who need secondary development, custom construction, or understanding the internal operation mechanism of dg.

### 1. Local Build &amp; Installation

dg provides a Makefile to simplify the build process. Ensure that the Go 1.18+ environment is installed locally first.

#### 1.1 Core Make Commands

```bash
# 1. Build the binary file (output to bin/dg in the project root directory)
make build

# 2. Install to system path (/usr/local/bin/dg, requires sudo permission)
sudo make install

# 3. Uninstall dg from the system (delete /usr/local/bin/dg)
sudo make uninstall
```

#### 1.2 Version Control & Release

dg provides version management commands that support Semantic Versioning (SemVer) upgrades. The version constant is defined in `internal/version/version.go`.

##### 1.2.1 Version Upgrade Commands

```bash
# Patch version upgrade (vX.Y.Z → vX.Y.(Z+1), e.g., v1.0.0 → v1.0.1)
make bump-patch

# Minor version upgrade (vX.Y.Z → vX.(Y+1).0, e.g., v1.0.1 → v1.1.0)
make bump-minor

# Major version upgrade (vX.Y.Z → v(X+1).0.0, e.g., v1.1.0 → v2.0.0)
make bump-major

# Custom version release (specify the version number, automatically update version.go, commit code, tag, and push)
make release VERSION=v1.0.0
```

##### 1.2.2 GitHub Releases Automatic Build (CI)

When a tag `vX.Y.Z` is pushed to the GitHub repository, the CI process will be triggered automatically:

- Build Linux tarballs suitable for`amd64` and `arm64` architectures (e.g., `dg_v1.0.0_linux_amd64.tar.gz`).

- Generate release notes through `changelogithub` and upload the tarballs to the GitHub Releases page.

### 2. Core Operation Mechanism

Understanding the operation behavior of dg helps troubleshoot problems and customize script logic:

#### 2.1 Concurrency Control Logic

- Before each execution, dg will check the PID in `<project>/.dg/state.yml`:

- If PID=0: Indicates idle and can be executed normally.

- If PID>0 and the corresponding process exists: Indicates running, and the current execution will be skipped automatically (to avoid concurrency conflicts).

#### 2.2 Update Detection Logic

- **Docker Image Detection**: Compare the digest of the image in the remote repository with the digest of the locally pulled image; if they are inconsistent, it is determined as an update.

- **Git Branch Detection**: Pull the latest commit of the remote branch and compare it with the last commit recorded locally; if they are inconsistent, it is determined as an update.

- **Git Tag Detection**: Pull all tags from the remote and compare them with the last tag list recorded locally; if there are new tags, it is determined as an update.

#### 2.3 Script Execution & Signal Handling

- If any update is detected, dg will execute the scripts in the order of `scripts` in `config.yml`; if the previous script exits with a non-zero code, the subsequent scripts will be aborted.

- When receiving SIGINT (Ctrl+C) or SIGTERM signal, dg will gracefully stop the current execution and write the state back to`state.yml`.

- After execution (success/failure), `state.yml` will be updated: PID is set to 0, the current execution timestamp is recorded, and the execution result (success/failure) is marked.

#### 2.4 Log Recording Rules

- Log format: `[timestamp] [log level] [module prefix (optional)] log content`, e.g., `2024-05-20T14:30:00Z INFO docker-detector Image postgres:17 has no update`.

- Log rotation: Generate log files by date (`YYYY-MM-DD.log`) by default, and automatically switch to a new file at 00:00 every day.

- Log cleanup: If`logs.retain_days` is configured in `config.yml`, logs exceeding the retention days will be deleted automatically every day.

### 3. Dependencies Description

dg depends on the following third-party libraries and tools, which will be pulled automatically during construction:

- **YAML Parsing**: `gopkg.in/yaml.v3` (handles the parsing and generation of `config.yml` and `state.yml`).

- **Docker Image Digest Acquisition**: `github.com/google/go-containerregistry` (connects to Docker remote repositories to obtain image digests).

- **System Tool Dependence**: Docker CLI (local image detection needs to obtain the local image digest through the `docker image inspect` command; ensure that Docker is installed and executable on the system).

## License

dg is open-source under the **Apache License 2.0**. See the `LICENSE` file in the project root directory for details.