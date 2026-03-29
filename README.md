# Scion

**Run a team of AI agents in parallel — each in its own container, with its own workspace, collaborating on your code.**

_sci·on /ˈsīən/ — a young shoot or twig, cut for grafting._

Scion orchestrates AI coding agents (Claude Code, Gemini CLI, Codex, and others) as isolated, concurrent processes. Each agent gets its own container, git branch, and credentials — so they can work on different parts of your project simultaneously without conflicts.

<a href="https://github.com/ptone/scion-athenaeum"><img width="425" alt="Relics of Athenaeum" src="https://github.com/user-attachments/assets/cbee74a3-f3aa-4739-b423-0a83d5dd4c13" /></a>&nbsp;<a href="https://www.youtube.com/watch?v=w16bsh6lFL8"><img width="300" alt="Visualization" src="https://github.com/user-attachments/assets/a615da24-33d8-4882-abe1-95adea4ed79a" /></a>

## Quick Start

```bash
# Install
go install github.com/GoogleCloudPlatform/scion/cmd/scion@latest

# Interactive setup (handles everything)
scion quickstart

# Or manual setup
scion init --machine                      # One-time machine setup
cd my-project && scion init               # Initialize project
export ANTHROPIC_API_KEY="sk-ant-..."     # Set your API key

# Launch your first agent
scion start my-agent "Fix the login bug" --attach

# Launch a team
scion start architect "Design the auth system" --type architect
scion start frontend "Build the login UI" --type fullstack-dev
scion start reviewer "Review all changes" --type code-reviewer
scion list
```

> **Note:** Container images must be built before starting agents. See [Installation](https://googlecloudplatform.github.io/scion/getting-started/install/) for details, or run `scion quickstart` for guided setup.

## Why Scion?

| Problem | How Scion Solves It |
|---------|-------------------|
| Agents step on each other's code | Each agent gets its own **git worktree** — separate branch, no conflicts |
| Can't run multiple agents at once | Agents run in **parallel containers** with independent workspaces |
| Locked into one AI provider | **Harness-agnostic** — Claude Code, Gemini CLI, Codex, or custom agents |
| Hard to specialize agents | **Role templates** — architect, code-reviewer, security-reviewer, docs-writer |
| No visibility into what agents are doing | **Real-time dashboard**, activity feed, cost tracking, terminal access |
| Complex multi-machine setup | **Hub mode** for distributed orchestration across machines and Kubernetes |

## Features

### Agent Lifecycle
```bash
scion start <name> "task"          # Launch an agent
scion start <name> --attach        # Launch and connect to terminal
scion list                          # See all running agents
scion attach <name>                 # Connect to a running agent
scion message <name> "do this"     # Send a message to an agent
scion logs <name> --follow          # Stream agent logs
scion stop <name>                   # Pause an agent
scion resume <name>                 # Resume a paused agent
scion delete <name>                 # Remove agent (with confirmation)
scion status                        # System health dashboard
```

### Role Templates
Pre-built agent personalities with system prompts, skills, and MCP server configurations:

| Template | Role | Capabilities |
|----------|------|-------------|
| `default` | General-purpose coding agent | Flexible, any task |
| `fullstack-dev` | End-to-end developer | GitHub MCP, filesystem MCP, git workflow skill |
| `code-reviewer` | Code review specialist | Review checklist skill, read-focused permissions |
| `security-reviewer` | Security auditor | OWASP Top 10 skill, vulnerability assessment |
| `architect` | System designer | Can spawn sub-agents, implementation planning |
| `docs-writer` | Documentation writer | Documentation standards skill |

```bash
scion start reviewer --type code-reviewer "Review the auth changes"
scion start auditor --type security-reviewer "Audit for vulnerabilities"
scion templates list                  # See all available templates
```

### Marketplace (MCP Servers & Skills)
Install MCP servers and agent skills from real community registries:

```bash
scion marketplace list                           # Browse built-in items
scion marketplace list --type mcp-server         # MCP servers only
scion marketplace install github --template default   # Install GitHub MCP
scion marketplace install notion --template default   # Install Notion MCP
scion marketplace pull https://github.com/openai/skills  # Pull from GitHub
scion marketplace registries                     # See 14+ external registries
```

**Built-in MCP servers**: GitHub, Sentry, Google Drive, Slack, Notion, Linear, PostgreSQL, SQLite, Playwright, Figma, filesystem, memory, Google Maps

**External registries**: Smithery.ai (7,300+ servers), MCP Official Registry (400+), Claude Code Plugin Marketplace, OpenAI Skills Repo, Gemini CLI Extensions Gallery

### Image Management
```bash
scion images list                              # Show installed images
scion images update --harness claude           # Update Claude Code (~30s)
scion images update --all                      # Update all harnesses
scion images build --harness-config my-agent   # Build custom agent image
```

### Web Dashboard
Real-time web UI with SSE-powered live updates:
- **Activity Feed** — "Mission control" view of all agent events
- **Cost Dashboard** — Token usage and cost per model/agent
- **Terminal Access** — Full PTY via browser
- **Cmd+K** — Command palette for quick navigation
- **Agent Detail** — Status, config, logs, limits tracking

### Observability
- **OpenTelemetry** — Normalized telemetry across all harnesses
- **Cost Tracking** — Per-agent token usage and estimated costs
- **Webhooks** — HMAC-signed event delivery to Slack, Discord, or any endpoint
- **pprof** — Profiling endpoints in debug mode

## Architecture

```
┌─────────────────────────────────────────────────┐
│                   scion CLI                      │
│  quickstart · start · list · attach · marketplace│
├─────────────────────────────────────────────────┤
│                  Agent Manager                   │
│  provision · run · message · monitor · delete    │
├──────────┬──────────┬──────────┬────────────────┤
│  Docker  │  Podman  │  Apple   │  Kubernetes    │
│  Runtime │  Runtime │Container │  Runtime       │
├──────────┴──────────┴──────────┴────────────────┤
│              Container (per agent)                │
│  ┌─────────────────────────────────────────┐    │
│  │  sciontool (PID 1)                      │    │
│  │  ├─ UID/GID mapping                     │    │
│  │  ├─ Lifecycle hooks                     │    │
│  │  ├─ Telemetry (OTel)                    │    │
│  │  └─ Status reporting                    │    │
│  ├─────────────────────────────────────────┤    │
│  │  Harness (Claude Code / Gemini / Codex) │    │
│  │  └─ tmux session                        │    │
│  ├─────────────────────────────────────────┤    │
│  │  /workspace (git worktree)              │    │
│  │  ~/.claude/ or ~/.gemini/ (config)      │    │
│  │  Skills & MCP servers                   │    │
│  └─────────────────────────────────────────┘    │
└─────────────────────────────────────────────────┘
```

### Core Concepts

| Concept | What It Is |
|---------|-----------|
| **Agent** | A containerized AI coding assistant with its own workspace and terminal |
| **Grove** | A project workspace (1:1 with a git repo). Holds agents, templates, settings |
| **Template** | A role definition — system prompt + skills + MCP servers + config |
| **Harness** | The AI provider (claude, gemini, codex, opencode) |
| **Profile** | A deployment target (local/docker, remote/kubernetes) |
| **Hub** | Optional control plane for multi-machine orchestration |

## Multi-Agent Patterns

### Fan-Out: Parallel Research
```bash
scion start researcher-1 "Research API design patterns" --type architect
scion start researcher-2 "Research competitor auth flows" --type architect
scion start researcher-3 "Research OAuth2 best practices" --type architect
# Each gets its own branch, works independently
```

### Pipeline: Sequential Stages
```bash
scion start architect "Design the auth system" --type architect --attach
# Wait for completion, then:
scion start impl "Implement the auth middleware" --type fullstack-dev --attach
# Then:
scion start reviewer "Review the implementation" --type code-reviewer --attach
```

### Team: Specialized Roles
```bash
scion start backend "Build REST API for users" --type fullstack-dev
scion start frontend "Build React login page" --type fullstack-dev
scion start security "Audit all code for vulnerabilities" --type security-reviewer
scion start docs "Write API documentation" --type docs-writer
scion list  # Watch them all work
```

## Documentation

**[Full Documentation](https://googlecloudplatform.github.io/scion/)**

| Section | Content |
|---------|---------|
| [Installation](https://googlecloudplatform.github.io/scion/getting-started/install/) | Prerequisites, image build, first setup |
| [Tutorial](https://googlecloudplatform.github.io/scion/getting-started/tutorial/) | Step-by-step first agent walkthrough |
| [Concepts](https://googlecloudplatform.github.io/scion/concepts/) | Agents, groves, harnesses, runtimes |
| [Templates](https://googlecloudplatform.github.io/scion/advanced-local/templates/) | Creating and customizing role templates |
| [CLI Reference](https://googlecloudplatform.github.io/scion/reference/cli/) | All commands with examples |
| [Hub Admin](https://googlecloudplatform.github.io/scion/hub-admin/hub-server/) | Distributed orchestration setup |
| [Kubernetes](https://googlecloudplatform.github.io/scion/hub-admin/kubernetes/) | K8s runtime configuration |
| [Contributing](CONTRIBUTING.md) | Architecture guide for contributors |

## Project Status

| Component | Status |
|-----------|--------|
| Local mode (Docker/Podman) | Stable |
| Hub-based workflows | Production-ready |
| Kubernetes runtime | Early, rough edges |
| Web dashboard | Functional, actively developed |
| Multi-agent orchestration | Primitives in place, workflows emerging |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for development setup, architecture overview, and how to add new harnesses or runtimes.

```bash
make build          # Build the binary
make test           # Run tests
make bench          # Run benchmarks
make ci             # Full CI check
```

## Disclaimers

This is not an officially supported Google product. This project is not eligible for the [Google Open Source Software Vulnerability Rewards Program](https://bughunters.google.com/open-source-security).

## License

Apache License, Version 2.0. See [LICENSE](LICENSE).
