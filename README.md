# dg — Deployment Guard (Go)

[中文](README-CN.md)

A simple, pragmatic deployment guard written in Go. It detects updates (Docker image digests) and runs your scripts via cron, with daily logs and safe pid-based concurrency.

## Features
- Docker image update detection (remote digest vs local image).
- Ordered script execution with unified logging (stdout/stderr).
- Daily log rotation and retention cleanup.
- Safe concurrency via pid semantics in `.dg/state.yml` (pid=0 idle; pid>0 and process exists → running).
- Cron integration: install/uninstall rules strictly filtered by `-config <abs path>`.
- Clear CLI: `run`, `install`, `uninstall`, `help`, `version`.
- Git monitoring: detects branch head changes and new tags.

## Installation

**Linux**

You can easily install or update to the latest version of `dg` using the one-click installation script. This will download the binary and install it to `/usr/local/bin`.

```bash
curl -fsSL https://raw.githubusercontent.com/indiefun/deploy-guard/main/install.sh | bash
```

## Quick Start
```bash
# Build
make build
# Or
go build -o dg ./cmd/dg

# Install to system (requires sudo)
make install
# Verify
dg version

# Run with a config
dg run -config /absolute/path/to/project/.dg/config.yml

# Install cron rule for this config
dg install -config /absolute/path/to/project/.dg/config.yml

# Uninstall cron rule for this config
dg uninstall -config /absolute/path/to/project/.dg/config.yml

# Uninstall from system
make uninstall
```

## Configuration
Place a YAML config at `<project>/.dg/config.yml`:
```yaml
cron: '*/1 * * * *'
watchs:
  docker:
    images:
      - postgres:17
      - redis:8
  git:
    remote: origin         # optional, default: origin
    branches: [main, dev]  # optional
    tags: true             # optional, check for new tags
    username: myuser       # optional, for HTTPS only
    password: mypass       # optional, for HTTPS only
scripts:
  - /absolute/path/script1.sh
  - ./relative/to/config.yml/dir/script2.sh
logs:
  retain_days: 7
```

- All relative paths are resolved against the directory of `config.yml`.
- Logs and state live beside the config:
  - Logs: `<config-dir>/logs/YYYY-MM-DD.log`
  - State: `<config-dir>/state.yml`
- Cron expression must be 5 fields (minute hour day month weekday). If you need seconds, use an external scheduler.
- `scripts` list must not be empty.
- At least one watch (`docker` or `git`) must be enabled.

## Run Behavior
- If `<config-dir>/state.yml` has `pid>0` and the process exists, the current run is skipped.
- Docker check: compares remote registry digest to local image digest.
- If any update is detected, scripts are executed in order. Non-zero exit aborts and logs error.
- Signals SIGINT/SIGTERM gracefully stop and write back state.
- On completion, writes `pid=0`, timestamps, and last result.

## Cron Integration
- Install: reads `cron` from config and writes a rule using the current `dg` binary path:
  - `<cron> /absolute/path/to/dg -config <abs path to config.yml>`
- Uninstall: removes only rules containing `-config <abs path>`; comments and other rules are left untouched.

## GitHub Releases (CI)
- Push a tag `vX.Y.Z` to trigger CI.
- CI builds Linux tarballs for `amd64` and `arm64`:
  - `dg_<tag>_linux_amd64.tar.gz`
  - `dg_<tag>_linux_arm64.tar.gz`
- CI creates release notes via `changelogithub` and uploads tarballs to the release.

## Versioning & Release
- Version constant: `internal/version/version.go`.
- Makefile targets:
```bash
make release VERSION=v1.0.0  # bump version.go, commit, tag, push
make bump-patch              # vX.Y.Z → vX.Y.(Z+1)
make bump-minor              # vX.Y.Z → vX.(Y+1).0
make bump-major              # vX.Y.Z → v(X+1).0.0
```

## Logging
- Format: `time level message` with module prefixes when relevant.
- Daily rotation by filename; retention cleanup according to `retain_days`.

## Security Notes
- No secrets are logged.
- Registry auth uses Docker keychain (`~/.docker/config.json`) via `go-containerregistry` default keychain.
- Git HTTPS credentials in config are used only in memory, but the constructed URL might be briefly visible in process list (`ps`). Use SSH or system credentials helper if this is a concern.

## Dependencies
- `gopkg.in/yaml.v3` — YAML parsing.
- `github.com/google/go-containerregistry` — registry digest.
- System Docker CLI — local image inspect via `docker image inspect`.

## Roadmap
- Optional checksums and signed release artifacts.

## License
Apache-2.0. See `LICENSE`.
