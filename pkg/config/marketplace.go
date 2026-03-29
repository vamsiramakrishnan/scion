// Copyright 2026 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package config

// MarketplaceItem represents an installable item from the marketplace.
type MarketplaceItem struct {
	// Name is the unique identifier (e.g., "github", "notion", "owasp-checklist")
	Name string `json:"name" yaml:"name"`
	// Type is "mcp-server", "skill", or "template"
	Type string `json:"type" yaml:"type"`
	// Description is a short human-readable description
	Description string `json:"description" yaml:"description"`
	// Category groups related items (e.g., "productivity", "code", "security")
	Category string `json:"category" yaml:"category"`
	// Harnesses lists which harnesses support this item (empty = all)
	Harnesses []string `json:"harnesses,omitempty" yaml:"harnesses,omitempty"`
	// Source is where the item comes from
	Source string `json:"source" yaml:"source"`
	// MCPConfig is the MCP server configuration (for type=mcp-server)
	MCPConfig *MCPServerConfig `json:"mcpConfig,omitempty" yaml:"mcpConfig,omitempty"`
	// SkillContent is the skill markdown content (for type=skill)
	SkillContent string `json:"skillContent,omitempty" yaml:"skillContent,omitempty"`
	// RequiredEnv lists environment variables the user must set
	RequiredEnv []string `json:"requiredEnv,omitempty" yaml:"requiredEnv,omitempty"`
	// Tags for search/filtering
	Tags []string `json:"tags,omitempty" yaml:"tags,omitempty"`
}

// MCPServerConfig is the configuration for an MCP server.
type MCPServerConfig struct {
	// Command is the executable to run
	Command string `json:"command" yaml:"command"`
	// Args are the command arguments
	Args []string `json:"args" yaml:"args"`
	// Env are environment variables for the server process
	Env map[string]string `json:"env,omitempty" yaml:"env,omitempty"`
}

// ExternalRegistries lists known registries where MCP servers and skills
// can be discovered. These are real, live registries maintained by the community.
type ExternalRegistry struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Type        string `json:"type"` // "mcp", "skills", "templates"
}

// KnownRegistries returns the list of external registries that SCION
// can pull from. These are real sources maintained by the MCP and AI community.
func KnownRegistries() []ExternalRegistry {
	return []ExternalRegistry{
		{
			Name:        "MCP Official Registry",
			URL:         "https://modelcontextprotocol.io/registry",
			Description: "Official MCP server registry with 400+ verified servers — searchable catalog with installation instructions",
			Type:        "mcp",
		},
		{
			Name:        "MCP Reference Servers",
			URL:         "https://github.com/modelcontextprotocol/servers",
			Description: "Reference MCP server implementations (filesystem, memory, sequential-thinking, git)",
			Type:        "mcp",
		},
		{
			Name:        "Smithery",
			URL:         "https://smithery.ai",
			Description: "Community MCP server registry with 2000+ servers — install via `npx @smithery/cli install <server>`",
			Type:        "mcp",
		},
		{
			Name:        "Awesome MCP Servers",
			URL:         "https://github.com/punkpeye/awesome-mcp-servers",
			Description: "Curated list of community MCP servers organized by category (databases, APIs, dev tools, AI, etc.)",
			Type:        "mcp",
		},
		{
			Name:        "Claude Code Skills",
			URL:         "https://github.com/anthropics/claude-code",
			Description: "Official Claude Code skills in .claude/skills/ format — markdown instruction files for specialized tasks",
			Type:        "skills",
		},
		{
			Name:        "Gemini CLI Extensions",
			URL:         "https://github.com/google-gemini/gemini-cli",
			Description: "Gemini CLI supports MCP servers in .gemini/settings.json and custom tool extensions",
			Type:        "mcp",
		},
		{
			Name:        "MCP.run",
			URL:         "https://www.mcp.run",
			Description: "Serverless MCP server hosting — run MCP servers without local installation via WebAssembly",
			Type:        "mcp",
		},
		{
			Name:        "Glama MCP Directory",
			URL:         "https://glama.ai/mcp/servers",
			Description: "MCP server directory with search, filtering, and installation instructions",
			Type:        "mcp",
		},
		// ─── Skills Registries ─────────────────────────────────────────
		{
			Name:        "Claude Code Plugin Marketplace",
			URL:         "https://claude.com/plugins",
			Description: "Official Claude Code marketplace — 30+ plugins including GitHub, Figma, Linear, Notion, Sentry, Slack, plus LSP servers. Install with /plugin install <name>",
			Type:        "skills",
		},
		{
			Name:        "OpenAI Skills Repo",
			URL:         "https://github.com/openai/skills",
			Description: "Official Codex skills repo (15k+ stars) — system, curated, and experimental skills in Agent Skills standard format (SKILL.md + YAML frontmatter)",
			Type:        "skills",
		},
		{
			Name:        "Gemini CLI Extensions Gallery",
			URL:         "https://github.com/gemini-cli-extensions",
			Description: "40+ official and community Gemini CLI extensions with MCP servers, context files, and slash commands",
			Type:        "skills",
		},
		{
			Name:        "Awesome Claude Skills",
			URL:         "https://github.com/travisvn/awesome-claude-skills",
			Description: "Curated community Claude Code skills — TDD, debugging, webapp-testing, security, design, plus skills from obra/superpowers",
			Type:        "skills",
		},
		{
			Name:        "Cross-Platform Agent Skills",
			URL:         "https://github.com/alirezarezvani/claude-skills",
			Description: "192+ cross-platform skills for Claude Code, Codex, Gemini CLI, and Cursor",
			Type:        "skills",
		},
		{
			Name:        "Agent Skills Standard",
			URL:         "https://agentskills.io",
			Description: "Open standard for agent skills (SKILL.md + YAML frontmatter) — shared by Claude Code, Codex, and compatible with Gemini CLI",
			Type:        "skills",
		},
	}
}

// BuiltInMarketplace returns the curated list of marketplace items.
// These are sourced from:
// - Official MCP server registry (github.com/modelcontextprotocol/servers)
// - Claude Code official marketplace
// - Community MCP servers
// - Agent skill best practices from Claude, Gemini, and Codex ecosystems
func BuiltInMarketplace() []MarketplaceItem {
	return []MarketplaceItem{
		// ─── MCP Servers: Code & Development ───────────────────────────
		{
			Name:        "github",
			Type:        "mcp-server",
			Description: "GitHub API — repos, issues, PRs, code search, file operations",
			Category:    "code",
			Source:      "github/github-mcp-server (official, replaces deprecated @modelcontextprotocol/server-github)",
			Harnesses:   []string{"claude", "gemini"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@anthropic-ai/github-mcp-server"},
				Env:     map[string]string{"GITHUB_PERSONAL_ACCESS_TOKEN": "${GITHUB_TOKEN}"},
			},
			RequiredEnv: []string{"GITHUB_TOKEN"},
			Tags:        []string{"git", "pr", "issues", "code-review"},
		},
		{
			Name:        "sentry",
			Type:        "mcp-server",
			Description: "Sentry error tracking — search issues, view stack traces, resolve errors",
			Category:    "code",
			Source:      "@sentry/mcp-server (official Sentry)",
			Harnesses:   []string{"claude", "gemini"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@sentry/mcp-server"},
				Env:     map[string]string{"SENTRY_AUTH_TOKEN": "${SENTRY_AUTH_TOKEN}"},
			},
			RequiredEnv: []string{"SENTRY_AUTH_TOKEN"},
			Tags:        []string{"errors", "monitoring", "debugging"},
		},

		// ─── MCP Servers: Productivity ─────────────────────────────────
		{
			Name:        "google-drive",
			Type:        "mcp-server",
			Description: "Google Drive — search, read, and create documents and spreadsheets",
			Category:    "productivity",
			Source:      "modelcontextprotocol/servers (official)",
			Harnesses:   []string{"claude"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-gdrive"},
			},
			Tags: []string{"docs", "sheets", "google-workspace", "gws"},
		},
		{
			Name:        "google-maps",
			Type:        "mcp-server",
			Description: "Google Maps — geocoding, directions, place search",
			Category:    "data",
			Source:      "modelcontextprotocol/servers (official)",
			Harnesses:   []string{"claude"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-google-maps"},
				Env:     map[string]string{"GOOGLE_MAPS_API_KEY": "${GOOGLE_MAPS_API_KEY}"},
			},
			RequiredEnv: []string{"GOOGLE_MAPS_API_KEY"},
			Tags:        []string{"maps", "geo", "location"},
		},
		{
			Name:        "slack",
			Type:        "mcp-server",
			Description: "Slack — read/send messages, manage channels, search conversations",
			Category:    "productivity",
			Source:      "modelcontextprotocol/servers (official)",
			Harnesses:   []string{"claude"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-slack"},
				Env:     map[string]string{"SLACK_BOT_TOKEN": "${SLACK_BOT_TOKEN}"},
			},
			RequiredEnv: []string{"SLACK_BOT_TOKEN"},
			Tags:        []string{"chat", "messaging", "team"},
		},
		{
			Name:        "notion",
			Type:        "mcp-server",
			Description: "Notion — search, read, create, and update pages and databases",
			Category:    "productivity",
			Source:      "@notionhq/notion-mcp-server (official Notion)",
			Harnesses:   []string{"claude", "gemini"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@notionhq/notion-mcp-server"},
				Env:     map[string]string{"OPENAPI_MCP_HEADERS": `{"Authorization": "Bearer ${NOTION_API_KEY}", "Notion-Version": "2022-06-28"}`},
			},
			RequiredEnv: []string{"NOTION_API_KEY"},
			Tags:        []string{"docs", "wiki", "knowledge-base"},
		},
		{
			Name:        "linear",
			Type:        "mcp-server",
			Description: "Linear — issues, projects, teams, and cycle tracking",
			Category:    "productivity",
			Source:      "community",
			Harnesses:   []string{"claude"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-linear"},
				Env:     map[string]string{"LINEAR_API_KEY": "${LINEAR_API_KEY}"},
			},
			RequiredEnv: []string{"LINEAR_API_KEY"},
			Tags:        []string{"issues", "project-management", "agile"},
		},

		// ─── MCP Servers: Data & Infrastructure ────────────────────────
		{
			Name:        "filesystem",
			Type:        "mcp-server",
			Description: "Filesystem — read, write, search files with directory access control",
			Category:    "data",
			Source:      "modelcontextprotocol/servers (official)",
			Harnesses:   []string{"claude"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "/workspace"},
			},
			Tags: []string{"files", "fs", "local"},
		},
		{
			Name:        "postgres",
			Type:        "mcp-server",
			Description: "PostgreSQL — query databases, inspect schemas, run SQL",
			Category:    "data",
			Source:      "modelcontextprotocol/servers (official)",
			Harnesses:   []string{"claude"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-postgres", "${DATABASE_URL}"},
			},
			RequiredEnv: []string{"DATABASE_URL"},
			Tags:        []string{"sql", "database", "query"},
		},
		{
			Name:        "sqlite",
			Type:        "mcp-server",
			Description: "SQLite — query local SQLite databases",
			Category:    "data",
			Source:      "modelcontextprotocol/servers (official)",
			Harnesses:   []string{"claude"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-sqlite", "${SQLITE_DB_PATH}"},
			},
			RequiredEnv: []string{"SQLITE_DB_PATH"},
			Tags:        []string{"sql", "database", "local"},
		},
		{
			Name:        "playwright",
			Type:        "mcp-server",
			Description: "Playwright — browser automation via accessibility snapshots, screenshots, testing",
			Category:    "data",
			Source:      "@playwright/mcp (official Microsoft, replaces Puppeteer)",
			Harnesses:   []string{"claude", "gemini"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@playwright/mcp"},
			},
			Tags: []string{"browser", "testing", "automation", "e2e"},
		},
		{
			Name:        "memory",
			Type:        "mcp-server",
			Description: "Persistent memory — knowledge graph for long-term agent memory",
			Category:    "data",
			Source:      "modelcontextprotocol/servers (official)",
			Harnesses:   []string{"claude"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "@modelcontextprotocol/server-memory"},
			},
			Tags: []string{"memory", "knowledge", "persistence", "context"},
		},

		// ─── MCP Servers: Design ───────────────────────────────────────
		{
			Name:        "figma",
			Type:        "mcp-server",
			Description: "Figma — read design files, inspect components, extract design tokens",
			Category:    "design",
			Source:      "community",
			Harnesses:   []string{"claude"},
			MCPConfig: &MCPServerConfig{
				Command: "npx",
				Args:    []string{"-y", "figma-mcp-server"},
				Env:     map[string]string{"FIGMA_ACCESS_TOKEN": "${FIGMA_ACCESS_TOKEN}"},
			},
			RequiredEnv: []string{"FIGMA_ACCESS_TOKEN"},
			Tags:        []string{"design", "ui", "components"},
		},

		// ─── Skills: Security ──────────────────────────────────────────
		{
			Name:        "owasp-top10",
			Type:        "skill",
			Description: "OWASP Top 10 security checklist — systematic vulnerability assessment",
			Category:    "security",
			Source:      "scion built-in",
			SkillContent: `# OWASP Top 10 Security Checklist
When auditing code, check each category:
- A01: Broken Access Control (IDOR, missing auth checks)
- A02: Cryptographic Failures (weak algorithms, plaintext secrets)
- A03: Injection (SQL, XSS, command injection)
- A04: Insecure Design (missing rate limiting, business logic flaws)
- A05: Security Misconfiguration (default creds, verbose errors)
- A06: Vulnerable Components (outdated dependencies with CVEs)
- A07: Authentication Failures (weak passwords, missing MFA)
- A08: Data Integrity (unsigned updates, deserialization flaws)
- A09: Logging Failures (secrets in logs, missing audit trail)
- A10: SSRF (user-controlled URLs in server requests)`,
			Tags: []string{"security", "audit", "vulnerability"},
		},
		{
			Name:        "code-review-checklist",
			Type:        "skill",
			Description: "Structured code review checklist — correctness, security, performance, readability",
			Category:    "code",
			Source:      "scion built-in",
			SkillContent: `# Code Review Checklist
## Correctness
- Logic handles edge cases and error paths
- Concurrency safety (no data races)
- Resource cleanup (files, connections)
## Security
- Input validation on all user data
- Parameterized queries (no SQL injection)
- Auth checks on sensitive operations
## Performance
- No N+1 queries or unbounded loops
- Appropriate indexing for database queries
## Readability
- Clear names and single-responsibility functions
- Tests cover critical paths`,
			Tags: []string{"review", "quality", "standards"},
		},
		{
			Name:        "api-design-standards",
			Type:        "skill",
			Description: "REST API design standards — naming, versioning, error handling, pagination",
			Category:    "code",
			Source:      "scion built-in",
			SkillContent: `# API Design Standards
## Naming
- Use plural nouns for resources: /users, /orders
- Use kebab-case for multi-word paths: /user-profiles
- Version via path prefix: /api/v1/
## Methods
- GET: Read (idempotent), POST: Create, PUT: Replace, PATCH: Update, DELETE: Remove
## Errors
- Use standard HTTP status codes (400, 401, 403, 404, 409, 500)
- Return structured error bodies: {"code": "...", "message": "..."}
## Pagination
- Use cursor-based pagination for large datasets
- Return next_cursor in response, accept cursor parameter
## Security
- Require authentication on all mutating endpoints
- Rate limit by API key or IP
- Validate Content-Type header`,
			Tags: []string{"api", "rest", "design", "standards"},
		},
		{
			Name:        "git-workflow",
			Type:        "skill",
			Description: "Git workflow for isolated worktrees — branching, commits, conflict resolution",
			Category:    "code",
			Source:      "scion built-in",
			SkillContent: `# Git Workflow for Scion Agents
## Worktree Rules
- You are on your own branch. Never checkout main.
- Compare: git diff main...HEAD
- View main files: git show main:path/to/file
- Rebase: git rebase main
## Commits
- Use conventional commits: feat:, fix:, refactor:, test:, docs:
- Always: git commit -m "message" (never open an editor)
## Conflicts
- git status to find conflicts
- Edit files, then: git add <files> && GIT_EDITOR=true git rebase --continue`,
			Tags: []string{"git", "workflow", "branching"},
		},
		{
			Name:        "testing-strategy",
			Type:        "skill",
			Description: "Testing strategy guide — unit, integration, e2e test patterns and when to use each",
			Category:    "code",
			Source:      "scion built-in",
			SkillContent: `# Testing Strategy
## Test Pyramid
- Unit tests: Fast, isolated, test one function. Cover edge cases.
- Integration tests: Test component interactions (DB, API).
- E2E tests: Test full user flows. Keep these minimal and stable.
## When to Write Tests
- Always: new public API methods, bug fixes (regression test)
- Usually: complex business logic, data transformations
- Skip: simple getters/setters, framework boilerplate
## Test Patterns
- Arrange-Act-Assert (AAA) structure
- Table-driven tests for multiple input/output combos
- Use test fixtures, not production data
- Mock external services, not internal modules`,
			Tags: []string{"testing", "quality", "tdd"},
		},
		{
			Name:        "documentation-standards",
			Type:        "skill",
			Description: "Documentation writing standards — audience, structure, examples, style",
			Category:    "docs",
			Source:      "scion built-in",
			SkillContent: `# Documentation Standards
## Audience First
- Tutorials: step-by-step for beginners
- How-to: solve a specific problem (intermediate)
- Reference: exhaustive API docs (advanced)
- Architecture: system design for contributors
## Structure
- Lead with a code example, explain after
- Use headers and bullet points for scanability
- Keep paragraphs short (3-4 sentences max)
## Style
- Active voice, present tense
- "Run the command" not "The command should be run"
- Include copy-pasteable code blocks
- Link to related docs, don't duplicate`,
			Tags: []string{"docs", "writing", "technical-writing"},
		},
	}
}

// FilterMarketplace returns items matching the given type and/or category.
func FilterMarketplace(items []MarketplaceItem, itemType, category, query string) []MarketplaceItem {
	var result []MarketplaceItem
	for _, item := range items {
		if itemType != "" && item.Type != itemType {
			continue
		}
		if category != "" && item.Category != category {
			continue
		}
		if query != "" {
			found := false
			queryLower := toLower(query)
			if contains(toLower(item.Name), queryLower) || contains(toLower(item.Description), queryLower) {
				found = true
			}
			for _, tag := range item.Tags {
				if contains(toLower(tag), queryLower) {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}
		result = append(result, item)
	}
	return result
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || containsImpl(s, sub))
}

func containsImpl(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
