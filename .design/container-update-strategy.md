# Container Image Update Strategy

## Problem

SCION bakes harness CLIs (Claude Code, Gemini CLI, Codex) into container images
at build time. When upstream ships a new version (daily for Claude Code), users
must rebuild and push images to stay current.

## Current Architecture (Correct)

```
core-base (~2GB, changes rarely)
  └─ scion-base (~100MB added, changes on scion release)
       ├─ scion-claude (~100MB added, changes on Claude Code release)
       ├─ scion-gemini (~50MB added, changes on Gemini CLI release)
       ├─ scion-codex (~50MB added, changes on Codex release)
       └─ scion-opencode (~30MB added)
```

This is intentionally build-time because:
1. **Deterministic** — same environment every time
2. **No network at runtime** — agents work air-gapped
3. **Full OS available** — harnesses need git, node, python, gcc, make, etc.
4. **Agents install packages** — `npm install`, `pip install` must work inside container
5. **Fast cold start** — no 30-60s install delay

## Solution: Three-Tier Update Strategy

### Tier 1: Harness-Only Rebuild (30 seconds)

The harness layer is THIN — just `npm install -g <package>`. Rebuilding only
the harness layer takes ~30 seconds because Docker caches the scion-base layer.

New command: `scion images update [--harness claude|gemini|codex|all]`

```bash
# Rebuild only the Claude harness image (30s, reuses cached scion-base)
scion images update --harness claude

# Rebuild all harness images (2 min total)
scion images update --harness all
```

Implementation: run `docker build` with `--cache-from` pointing to existing
scion-base, only replacing the harness npm install layer.

### Tier 2: Version Pinning + Auto-Check

Track upstream harness versions and notify when updates are available:

```yaml
# ~/.scion/settings.yaml
harness_versions:
  claude: "2.1.87"     # currently installed
  gemini: "1.3.2"
  codex: "0.9.1"

auto_update_check: true  # check for new versions on scion start
```

```bash
$ scion start my-agent "task"
  Note: Claude Code 2.2.0 available (installed: 2.1.87)
  Run: scion images update --harness claude
Starting agent 'my-agent'...
```

### Tier 3: Arbitrary Harness Support (ADK, Custom Agents)

For ADK agents or any custom harness, support a `Dockerfile` field in the
harness-config that builds on top of scion-base:

```yaml
# ~/.scion/harness-configs/my-adk-agent/config.yaml
harness: custom
image: my-adk-agent:latest
dockerfile: |
  ARG BASE_IMAGE
  FROM ${BASE_IMAGE}
  RUN pip install google-adk-agent my-custom-package
  CMD ["python", "-m", "my_agent"]
```

```bash
# Build custom harness image from inline Dockerfile
scion images build --harness-config my-adk-agent
```

This reuses scion-base (cached) and adds only the custom layer.

### Tier 4: Volume-Mounted Harness Override (Development Only)

Already exists via `SCION_DEV_BINARIES`:
```bash
# Override harness binary without rebuilding image
export SCION_DEV_BINARIES=/path/to/local/binaries
scion start my-agent "task"  # Uses local binary instead of image's
```

Extend this to support harness-specific overrides:
```bash
# Mount a specific npm global directory
export SCION_HARNESS_OVERRIDE=/usr/local/lib/node_modules/@anthropic-ai/claude-code
```

## What to Build

### Phase 1: `scion images` command group
- `scion images list` — show installed images with harness versions
- `scion images update --harness <name>` — rebuild harness layer only (30s)
- `scion images update --all` — rebuild all harness layers
- `scion images check` — check for upstream version updates

### Phase 2: Version tracking
- Store installed harness versions in ~/.scion/harness-versions.json
- Check npm registry for latest versions on `scion start` (with 24h cache)
- Show update notification (non-blocking) when newer version available

### Phase 3: Custom harness Dockerfile support
- `dockerfile` field in harness-config config.yaml
- `scion images build --harness-config <name>` builds from inline Dockerfile
- Supports ADK agents, Python agents, Rust agents, anything

## Why NOT Install-at-Runtime

1. **Network required** — agents are designed to work air-gapped
2. **Non-deterministic** — different agents could get different versions
3. **Slow cold start** — npm install adds 30-60s per agent start
4. **Package install breaks** — if npm registry is down, agents can't start
5. **The tools matter** — harnesses need git, gcc, make, python, etc. These
   MUST be in the base image. A minimal container would break the harness
   AND the agent's ability to install packages and build code.
