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

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/GoogleCloudPlatform/scion/pkg/config"
	"github.com/GoogleCloudPlatform/scion/pkg/util"
	"github.com/spf13/cobra"
)

var marketplaceCmd = &cobra.Command{
	Use:     "marketplace",
	Aliases: []string{"market", "mp"},
	Short:   "Browse and install MCP servers, skills, and templates from the marketplace",
	Long: `Browse the curated marketplace of MCP servers, agent skills, and templates.

Items are sourced from:
  - Official MCP Server Registry (modelcontextprotocol/servers)
  - Claude Code Marketplace
  - Community MCP servers (Notion, Linear, Figma, etc.)
  - Agent skill best practices

Use 'scion marketplace list' to browse and 'scion marketplace install' to add
items to your templates.`,
}

var mpListCmd = &cobra.Command{
	Use:   "list [query]",
	Short: "List available marketplace items",
	Long: `List MCP servers, skills, and templates available for installation.

Examples:
  scion marketplace list                    # List all items
  scion marketplace list --type mcp-server  # Only MCP servers
  scion marketplace list --category code    # Only code-related items
  scion marketplace list github             # Search for "github"
  scion marketplace list --type skill       # Only skills`,
	RunE: runMPList,
}

var mpInstallCmd = &cobra.Command{
	Use:   "install <name> [--template <template>]",
	Short: "Install a marketplace item into a template",
	Long: `Install an MCP server or skill into an agent template.

For MCP servers, this adds the server configuration to the template's
Claude Code settings.json (or equivalent for other harnesses).

For skills, this creates a skill file in the template's skills/ directory.

Examples:
  scion marketplace install github --template code-reviewer
  scion marketplace install owasp-top10 --template security-reviewer
  scion marketplace install notion --template default
  scion marketplace install postgres --template fullstack-dev`,
	RunE: runMPInstall,
}

var mpInfoCmd = &cobra.Command{
	Use:   "info <name>",
	Short: "Show detailed information about a marketplace item",
	RunE:  runMPInfo,
}

var mpRegistriesCmd = &cobra.Command{
	Use:   "registries",
	Short: "List external registries for discovering MCP servers and skills",
	Long: `Show known external registries where you can find more MCP servers,
agent skills, and templates beyond the built-in marketplace.

These registries include:
  - Official MCP Server Registry (modelcontextprotocol/servers)
  - Smithery.ai (2000+ community MCP servers)
  - Awesome MCP Servers (curated community list)
  - Claude Code Skills (official skill format)
  - Gemini CLI Extensions
  - MCP.run (serverless MCP hosting)
  - Glama MCP Directory`,
	RunE: runMPRegistries,
}

var mpPullCmd = &cobra.Command{
	Use:   "pull <github-url> [--template <template>]",
	Short: "Pull skills from a GitHub repo into a template",
	Long: `Clone a GitHub repository containing agent skills (SKILL.md files)
and install them into a template's skills directory.

Works with repos following the Agent Skills standard (agentskills.io):
  - Claude Code skills (.claude/skills/)
  - Codex skills (.agents/skills/)
  - Generic skills (skills/ or root SKILL.md files)

Examples:
  scion marketplace pull https://github.com/openai/skills --template default
  scion marketplace pull https://github.com/travisvn/awesome-claude-skills --template code-reviewer
  scion marketplace pull https://github.com/alirezarezvani/claude-skills --template fullstack-dev`,
	RunE: runMPPull,
}

var (
	mpTypeFilter     string
	mpCategoryFilter string
	mpTemplate       string
	mpFormatJSON     bool
)

func init() {
	rootCmd.AddCommand(marketplaceCmd)
	marketplaceCmd.AddCommand(mpListCmd)
	marketplaceCmd.AddCommand(mpInstallCmd)
	marketplaceCmd.AddCommand(mpInfoCmd)
	marketplaceCmd.AddCommand(mpRegistriesCmd)
	marketplaceCmd.AddCommand(mpPullCmd)

	mpListCmd.Flags().StringVar(&mpTypeFilter, "type", "", "Filter by type (mcp-server, skill, template)")
	mpListCmd.Flags().StringVar(&mpCategoryFilter, "category", "", "Filter by category (code, productivity, security, data, design, docs)")
	mpListCmd.Flags().BoolVar(&mpFormatJSON, "json", false, "Output as JSON")

	mpInstallCmd.Flags().StringVar(&mpTemplate, "template", "default", "Target template to install into")
	mpPullCmd.Flags().StringVar(&mpTemplate, "template", "default", "Target template to install skills into")
}

func runMPList(cmd *cobra.Command, args []string) error {
	query := ""
	if len(args) > 0 {
		query = strings.Join(args, " ")
	}

	items := config.FilterMarketplace(config.BuiltInMarketplace(), mpTypeFilter, mpCategoryFilter, query)

	if mpFormatJSON {
		return json.NewEncoder(os.Stdout).Encode(items)
	}

	if len(items) == 0 {
		fmt.Println("No items found matching your criteria.")
		return nil
	}

	// Group by type
	mcpServers := []config.MarketplaceItem{}
	skills := []config.MarketplaceItem{}
	for _, item := range items {
		switch item.Type {
		case "mcp-server":
			mcpServers = append(mcpServers, item)
		case "skill":
			skills = append(skills, item)
		}
	}

	if len(mcpServers) > 0 {
		fmt.Printf("\n%sMCP Servers%s (%d available)\n", util.Bold, util.Reset, len(mcpServers))
		fmt.Printf("%-20s %-12s %s\n", "NAME", "CATEGORY", "DESCRIPTION")
		fmt.Printf("%-20s %-12s %s\n", "────", "────────", "───────────")
		for _, item := range mcpServers {
			fmt.Printf("%-20s %-12s %s\n", item.Name, item.Category, item.Description)
		}
	}

	if len(skills) > 0 {
		fmt.Printf("\n%sSkills%s (%d available)\n", util.Bold, util.Reset, len(skills))
		fmt.Printf("%-25s %-12s %s\n", "NAME", "CATEGORY", "DESCRIPTION")
		fmt.Printf("%-25s %-12s %s\n", "────", "────────", "───────────")
		for _, item := range skills {
			fmt.Printf("%-25s %-12s %s\n", item.Name, item.Category, item.Description)
		}
	}

	fmt.Printf("\n%sInstall:%s scion marketplace install <name> --template <template>\n", util.Dim, util.Reset)
	fmt.Printf("%sDetails:%s scion marketplace info <name>\n\n", util.Dim, util.Reset)

	return nil
}

func runMPInstall(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("item name required. Run 'scion marketplace list' to see available items")
	}
	name := args[0]

	// Find the item
	items := config.BuiltInMarketplace()
	var item *config.MarketplaceItem
	for i := range items {
		if items[i].Name == name {
			item = &items[i]
			break
		}
	}
	if item == nil {
		return fmt.Errorf("item %q not found. Run 'scion marketplace list' to see available items", name)
	}

	// Resolve template directory
	globalDir, err := config.GetGlobalDir()
	if err != nil {
		return fmt.Errorf("failed to get global dir: %w", err)
	}
	templateDir := filepath.Join(globalDir, "templates", mpTemplate)
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return fmt.Errorf("template %q not found. Run 'scion templates list' to see available templates", mpTemplate)
	}

	switch item.Type {
	case "mcp-server":
		return installMCPServer(item, templateDir)
	case "skill":
		return installSkill(item, templateDir)
	default:
		return fmt.Errorf("unsupported item type: %s", item.Type)
	}
}

func installMCPServer(item *config.MarketplaceItem, templateDir string) error {
	if item.MCPConfig == nil {
		return fmt.Errorf("item %q has no MCP configuration", item.Name)
	}

	// Write to Claude Code settings.json
	settingsDir := filepath.Join(templateDir, "home", ".claude")
	settingsPath := filepath.Join(settingsDir, "settings.json")

	// Read existing settings or create new
	var settings map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		if err := json.Unmarshal(data, &settings); err != nil {
			settings = make(map[string]interface{})
		}
	} else {
		settings = map[string]interface{}{
			"permissions": map[string]interface{}{"allow": []string{"*"}},
		}
	}

	// Get or create mcpServers map
	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	// Add the MCP server config
	serverConfig := map[string]interface{}{
		"command": item.MCPConfig.Command,
		"args":    item.MCPConfig.Args,
	}
	if len(item.MCPConfig.Env) > 0 {
		serverConfig["env"] = item.MCPConfig.Env
	}
	mcpServers[item.Name] = serverConfig
	settings["mcpServers"] = mcpServers

	// Write back
	if err := os.MkdirAll(settingsDir, 0755); err != nil {
		return fmt.Errorf("create settings dir: %w", err)
	}
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return fmt.Errorf("write settings: %w", err)
	}

	fmt.Printf("%s%s  Installed MCP server %q into template %q (Claude Code)%s\n", util.Bold, util.Green, item.Name, mpTemplate, util.Reset)

	// Also install into Gemini settings.json if it exists
	geminiSettingsDir := filepath.Join(templateDir, "home", ".gemini")
	geminiSettingsPath := filepath.Join(geminiSettingsDir, "settings.json")
	if _, err := os.Stat(geminiSettingsPath); err == nil {
		installMCPServerGemini(item, geminiSettingsPath)
	} else {
		// Create Gemini config with MCP server too
		installMCPServerGemini(item, geminiSettingsPath)
	}

	if len(item.RequiredEnv) > 0 {
		fmt.Printf("\n%sRequired environment variables:%s\n", util.Bold, util.Reset)
		for _, env := range item.RequiredEnv {
			fmt.Printf("  %s=%s\n", env, os.Getenv(env))
			if os.Getenv(env) == "" {
				fmt.Printf("    %s%s(not set — configure before starting agents)%s\n", util.Bold, util.Yellow, util.Reset)
			}
		}
	}
	return nil
}

// installMCPServerGemini adds an MCP server to Gemini's settings.json.
func installMCPServerGemini(item *config.MarketplaceItem, settingsPath string) {
	var settings map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	mcpServers, ok := settings["mcpServers"].(map[string]interface{})
	if !ok {
		mcpServers = make(map[string]interface{})
	}

	serverConfig := map[string]interface{}{
		"type":    "stdio",
		"command": item.MCPConfig.Command,
		"args":    item.MCPConfig.Args,
	}
	if len(item.MCPConfig.Env) > 0 {
		serverConfig["env"] = item.MCPConfig.Env
	}
	mcpServers[item.Name] = serverConfig
	settings["mcpServers"] = mcpServers

	settingsDir := filepath.Dir(settingsPath)
	os.MkdirAll(settingsDir, 0755)
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return
	}
	if err := os.WriteFile(settingsPath, data, 0644); err != nil {
		return
	}
	fmt.Printf("%s%s  Installed MCP server %q into template %q (Gemini CLI)%s\n", util.Bold, util.Green, item.Name, mpTemplate, util.Reset)
}

func installSkill(item *config.MarketplaceItem, templateDir string) error {
	skillsDir := filepath.Join(templateDir, "skills")
	if err := os.MkdirAll(skillsDir, 0755); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}

	skillPath := filepath.Join(skillsDir, item.Name+".md")
	if err := os.WriteFile(skillPath, []byte(item.SkillContent), 0644); err != nil {
		return fmt.Errorf("write skill: %w", err)
	}

	fmt.Printf("%s%s  Installed skill %q into template %q%s\n", util.Bold, util.Green, item.Name, mpTemplate, util.Reset)
	return nil
}

func runMPInfo(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("item name required")
	}
	name := args[0]

	items := config.BuiltInMarketplace()
	for _, item := range items {
		if item.Name == name {
			fmt.Printf("\n%s%s%s\n", util.Bold, item.Name, util.Reset)
			fmt.Printf("  Type:        %s\n", item.Type)
			fmt.Printf("  Category:    %s\n", item.Category)
			fmt.Printf("  Source:      %s\n", item.Source)
			fmt.Printf("  Description: %s\n", item.Description)
			if len(item.Harnesses) > 0 {
				fmt.Printf("  Harnesses:   %s\n", strings.Join(item.Harnesses, ", "))
			}
			if len(item.Tags) > 0 {
				fmt.Printf("  Tags:        %s\n", strings.Join(item.Tags, ", "))
			}
			if len(item.RequiredEnv) > 0 {
				fmt.Printf("  Required:    %s\n", strings.Join(item.RequiredEnv, ", "))
			}
			if item.MCPConfig != nil {
				fmt.Printf("  Command:     %s %s\n", item.MCPConfig.Command, strings.Join(item.MCPConfig.Args, " "))
			}
			fmt.Println()
			return nil
		}
	}
	return fmt.Errorf("item %q not found", name)
}

func runMPPull(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("GitHub URL required. Example: scion marketplace pull https://github.com/openai/skills --template default")
	}
	repoURL := args[0]

	// Resolve template directory
	globalDir, err := config.GetGlobalDir()
	if err != nil {
		return fmt.Errorf("failed to get global dir: %w", err)
	}
	templateDir := filepath.Join(globalDir, "templates", mpTemplate)
	if _, err := os.Stat(templateDir); os.IsNotExist(err) {
		return fmt.Errorf("template %q not found", mpTemplate)
	}

	// Create a temp directory for cloning
	tmpDir, err := os.MkdirTemp("", "scion-marketplace-pull-*")
	if err != nil {
		return fmt.Errorf("create temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	fmt.Printf("Cloning %s...\n", repoURL)

	// Shallow clone the repo
	cloneCmd := exec.Command("git", "clone", "--depth=1", repoURL, tmpDir)
	cloneCmd.Stdout = os.Stdout
	cloneCmd.Stderr = os.Stderr
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	// Search for SKILL.md files in common locations
	skillDirs := []string{
		filepath.Join(tmpDir, ".claude", "skills"),
		filepath.Join(tmpDir, ".agents", "skills"),
		filepath.Join(tmpDir, "skills"),
		tmpDir, // root-level SKILL.md files
	}

	targetSkillsDir := filepath.Join(templateDir, "skills")
	if err := os.MkdirAll(targetSkillsDir, 0755); err != nil {
		return fmt.Errorf("create skills dir: %w", err)
	}

	installed := 0
	for _, dir := range skillDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() {
				// Check for SKILL.md inside the directory
				skillFile := filepath.Join(dir, entry.Name(), "SKILL.md")
				if _, err := os.Stat(skillFile); err == nil {
					destDir := filepath.Join(targetSkillsDir, entry.Name())
					os.MkdirAll(destDir, 0755)
					data, _ := os.ReadFile(skillFile)
					if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), data, 0644); err == nil {
						fmt.Printf("  %s%s  %s%s\n", util.Bold, util.Green, entry.Name(), util.Reset)
						installed++
					}
				}
			} else if strings.HasSuffix(entry.Name(), ".md") && entry.Name() != "README.md" && entry.Name() != "CHANGELOG.md" {
				// Copy standalone .md skill files
				data, _ := os.ReadFile(filepath.Join(dir, entry.Name()))
				if len(data) > 0 {
					destName := strings.TrimSuffix(entry.Name(), ".md")
					destDir := filepath.Join(targetSkillsDir, destName)
					os.MkdirAll(destDir, 0755)
					if err := os.WriteFile(filepath.Join(destDir, "SKILL.md"), data, 0644); err == nil {
						fmt.Printf("  %s%s  %s%s\n", util.Bold, util.Green, destName, util.Reset)
						installed++
					}
				}
			}
		}
	}

	if installed == 0 {
		fmt.Println("No skills found in repository. Expected SKILL.md files in .claude/skills/, .agents/skills/, or skills/")
	} else {
		fmt.Printf("\n%sInstalled %d skills into template %q%s\n", util.Bold, installed, mpTemplate, util.Reset)
	}

	return nil
}

func runMPRegistries(cmd *cobra.Command, args []string) error {
	registries := config.KnownRegistries()

	fmt.Printf("\n%sExternal Registries%s — discover more MCP servers and skills\n\n", util.Bold, util.Reset)

	for _, r := range registries {
		fmt.Printf("  %s%s%s [%s]\n", util.Bold, r.Name, util.Reset, r.Type)
		fmt.Printf("  %s\n", r.Description)
		fmt.Printf("  %s%s%s\n\n", util.Dim, r.URL, util.Reset)
	}

	fmt.Printf("%sTo install from Smithery:%s\n", util.Bold, util.Reset)
	fmt.Printf("  npx @smithery/cli install <server-name>\n\n")

	fmt.Printf("%sTo install from MCP official:%s\n", util.Bold, util.Reset)
	fmt.Printf("  scion marketplace install github --template <template>\n\n")

	fmt.Printf("%sTo manually add any MCP server:%s\n", util.Bold, util.Reset)
	fmt.Printf("  Edit your template's home/.claude/settings.json or home/.gemini/settings.json\n")
	fmt.Printf("  and add the server to the mcpServers object.\n\n")

	return nil
}
