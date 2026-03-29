# Contributing to Scion

Thank you for your interest in contributing to Scion! This guide covers development setup, architecture, and how to add new capabilities.

## Development Setup

### Prerequisites

- Go 1.25+
- Docker or Podman
- Node.js 20+ (for web frontend)
- Git 2.47+ (for worktree support)

### Building

```bash
git clone https://github.com/GoogleCloudPlatform/scion
cd scion

make build          # Build scion binary to ./build/
make install        # Build + install to ~/.local/bin/
make test           # Run all tests
make test-fast      # Run tests without SQLite (faster, less memory)
make bench          # Run performance benchmarks
make lint           # Run go vet
make ci             # Full CI pipeline (fmt + lint + test + build)

# Web frontend
make web            # Build TypeScript/Lit frontend
make web-typecheck  # TypeScript type checking only

# Container images for development
make container-binaries  # Cross-compile for Linux containers
```

### Project Structure

```
scion/
├── cmd/                    # CLI commands (Cobra)
│   ├── scion/              # Main binary entry point
│   ├── sciontool/          # Agent-side helper binary
│   └── *.go                # One file per command group
├── pkg/                    # Core libraries
│   ├── agent/              # Agent lifecycle (provision, run, message)
│   ├── hub/                # Hub server (API, events, auth, webhooks)
│   ├── harness/            # AI provider integrations
│   │   ├── claude_code.go  # Claude Code harness
│   │   ├── gemini_cli.go   # Gemini CLI harness
│   │   ├── codex.go        # Codex harness
│   │   └── */embeds/       # Harness-specific config files
│   ├── runtime/            # Container runtimes (Docker, Podman, K8s)
│   ├── config/             # Settings, templates, marketplace
│   │   ├── embeds/         # Embedded templates and defaults
│   │   └── marketplace.go  # MCP server + skills registry
│   ├── store/              # Persistence (SQLite, Ent ORM)
│   ├── wsprotocol/         # WebSocket protocol
│   ├── sciontool/          # Agent helper (telemetry, hooks, status)
│   └── api/                # Shared types and interfaces
├── web/                    # TypeScript/Lit web frontend
│   └── src/
│       ├── components/     # Lit web components (pages + shared)
│       ├── client/         # State manager, API client, router
│       └── shared/         # Shared types
├── image-build/            # Container image Dockerfiles
│   ├── core-base/          # Foundation (Go, Git, Node, system tools)
│   ├── scion-base/         # Adds scion + sciontool binaries
│   ├── claude/             # Claude Code harness layer
│   ├── gemini/             # Gemini CLI harness layer
│   └── scripts/            # Build scripts
├── .design/                # Architecture decision docs (140+ files)
├── docs-site/              # Documentation site (Starlight)
└── examples/               # Example workflows
```

### Key Interfaces

**Harness** (`pkg/api/harness.go`) — AI provider integration:
```go
type Harness interface {
    Name() string
    GetCommand(task string, resume bool, baseArgs []string) []string
    GetEnv(agentName, agentHome, unixUsername string) map[string]string
    Provision(ctx context.Context, name, dir, home, workspace string) error
    InjectAgentInstructions(home string, content []byte) error
    InjectSystemPrompt(home string, content []byte) error
    SkillsDir() string
    ResolveAuth(auth AuthConfig) (*ResolvedAuth, error)
}
```

**Runtime** (`pkg/runtime/`) — Container execution:
```go
type Runtime interface {
    Name() string
    Run(ctx context.Context, cfg RunConfig) (string, error)
    List(ctx context.Context, filters map[string]string) ([]AgentInfo, error)
    Delete(ctx context.Context, id string) error
    Attach(ctx context.Context, name string) error
    Exec(ctx context.Context, name string, cmd []string) (string, error)
}
```

**Store** (`pkg/store/`) — Data persistence:
```go
type Store interface {
    GetAgent(ctx context.Context, id string) (*Agent, error)
    ListAgents(ctx context.Context, filter AgentFilter, opts ListOptions) (*ListResult[Agent], error)
    CreateAgent(ctx context.Context, agent *Agent) error
    UpdateAgent(ctx context.Context, agent *Agent) error
    // ... (groves, templates, users, policies, etc.)
}
```

## Adding a New Harness

To add support for a new AI provider:

1. **Create the harness implementation** in `pkg/harness/my_harness.go`:
   - Implement the `Harness` interface
   - Define `GetCommand()` to return the CLI command and args
   - Define `GetEnv()` for harness-specific environment variables
   - Implement `Provision()` for config file injection
   - Create `pkg/harness/myharness/embeds/` for config templates

2. **Create the harness config** in `pkg/config/embeds/harness-configs/`:
   - `config.yaml` with `harness: myharness`, `image: scion-myharness:latest`
   - `home/` directory with harness-specific dotfiles

3. **Create the Dockerfile** in `image-build/myharness/`:
   ```dockerfile
   ARG BASE_IMAGE
   FROM ${BASE_IMAGE}
   RUN npm install -g my-harness-cli && npm cache clean --force
   CMD ["my-harness-cli"]
   ```

4. **Register the harness** in `pkg/harness/all.go`

5. **Add to build script** in `image-build/scripts/build-images.sh`

## Adding a New Runtime

To add a new container runtime (beyond Docker/Podman/K8s):

1. **Implement the Runtime interface** in `pkg/runtime/my_runtime.go`
2. **Add to factory** in `pkg/runtime/factory.go`
3. **Add runtime config** to settings schema in `pkg/config/settings_v1.go`

## Adding Templates

Built-in templates live in `pkg/config/embeds/templates/`:

```
my-template/
├── scion-agent.yaml      # Config: description, instructions, env
├── system-prompt.md       # Agent persona and behavior
├── agents.md              # Task workflow instructions
├── skills/                # Skill files (SKILL.md format)
│   └── my-skill.md
└── home/                  # Harness-specific config files
    └── .claude/
        └── settings.json  # MCP servers, permissions
```

Register in `pkg/config/init.go` → `builtInRoleTemplates` array.

## Adding MCP Servers to Marketplace

Edit `pkg/config/marketplace.go` → `BuiltInMarketplace()`:

```go
{
    Name:        "my-server",
    Type:        "mcp-server",
    Description: "What it does",
    Category:    "code",
    Source:      "npm package or source URL",
    Harnesses:   []string{"claude", "gemini"},
    MCPConfig: &MCPServerConfig{
        Command: "npx",
        Args:    []string{"-y", "my-mcp-server"},
        Env:     map[string]string{"API_KEY": "${MY_API_KEY}"},
    },
    RequiredEnv: []string{"MY_API_KEY"},
    Tags:        []string{"tag1", "tag2"},
},
```

## Web Frontend Development

The frontend uses Lit web components with SSE for real-time updates.

```bash
cd web
npm install
npm run dev       # Dev server with hot reload
npm run build     # Production build
npm run typecheck # Type checking
```

**Page component pattern:**
```typescript
@customElement('scion-page-mypage')
export class ScionPageMyPage extends LitElement {
  @property({ type: Object }) pageData: PageData | null = null;
  @state() private loading = true;
  @state() private items: Agent[] = [];

  override connectedCallback() {
    super.connectedCallback();
    stateManager.setScope({ type: 'dashboard' });
    void this.loadData();
    stateManager.addEventListener('agents-updated', this.onUpdate);
  }

  // ... render(), loadData(), etc.
}
```

**Key patterns:**
- SSE via `stateManager` for real-time updates
- `apiFetch()` for REST API calls
- Shoelace components for UI elements (`sl-button`, `sl-icon`, etc.)
- CSS custom properties for theming (`--scion-primary`, `--scion-surface`, etc.)
- Route registration in `web/src/client/main.ts`

## Running Tests

```bash
make test                    # All tests
make test-fast               # Without SQLite (CI-friendly)
make bench                   # Performance benchmarks
make bench-compare           # Save benchmarks for comparison
go test ./pkg/hub/... -v     # Specific package
go test -run TestMyFunc ./pkg/agent/...  # Specific test
```

## Code Style

- **Go**: `gofmt` formatting, `go vet` for static analysis
- **TypeScript**: Prettier + ESLint via npm scripts
- **Commits**: Conventional commits (`feat:`, `fix:`, `perf:`, `docs:`)
- **No unnecessary abstractions**: Three similar lines > premature helper function
- **Error messages**: Always include what went wrong AND how to fix it
- **Tests**: Required for new public APIs; table-driven tests preferred

## Design Documents

Architecture decisions are documented in `.design/` (140+ files). Key ones:

| Document | Topic |
|----------|-------|
| `multi-agent-explore.md` | Multi-agent workflow patterns |
| `hosted/agent-hub-access.md` | Agent-to-agent orchestration |
| `container-update-strategy.md` | Image update architecture |
| `performance-analysis.md` | Performance audit and improvements |
| `server-refactor.md` | Server initialization cleanup |
| `web-frontend-performance.md` | UI optimization strategy |
