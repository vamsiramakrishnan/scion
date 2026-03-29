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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/GoogleCloudPlatform/scion/pkg/config"
	"github.com/GoogleCloudPlatform/scion/pkg/harness"
	"github.com/GoogleCloudPlatform/scion/pkg/util"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Interactive setup wizard — zero to first agent in one command",
	Long: `Fully guided interactive setup that takes you from zero to running agents.

Handles everything:
  1. Prerequisites check (git, container runtime)
  2. Machine initialization
  3. Container image setup (build locally or use existing)
  4. API key configuration (secure input with validation)
  5. Project grove initialization
  6. Shell completion installation
  7. Launch your first agent

This is the recommended way to get started with scion.`,
	RunE: runQuickstart,
}

func init() {
	rootCmd.AddCommand(quickstartCmd)
}

func runQuickstart(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	clearScreen()
	printBanner()

	// ─── Step 1: Prerequisites ──────────────────────────────────────
	printStep(1, 7, "Checking prerequisites")
	prereqs := checkPrerequisites()

	if !prereqs.allPassed {
		for _, msg := range prereqs.failures {
			printFail(msg)
		}
		fmt.Println()
		printHint("Fix the issues above and run 'scion quickstart' again.")
		return fmt.Errorf("prerequisites not met")
	}
	for _, msg := range prereqs.passed {
		printOK(msg)
	}
	fmt.Println()

	// ─── Step 2: Machine Init ───────────────────────────────────────
	printStep(2, 7, "Machine setup")
	globalDir, _ := config.GetGlobalDir()

	if _, err := os.Stat(globalDir); os.IsNotExist(err) {
		if err := config.InitMachine(harness.All(), config.InitMachineOpts{}); err != nil {
			return fmt.Errorf("machine init failed: %w", err)
		}
		printOK(fmt.Sprintf("Created %s with templates and harness configs", globalDir))
	} else {
		printOK("Already initialized")
	}
	fmt.Println()

	// ─── Step 3: Container Images ───────────────────────────────────
	printStep(3, 7, "Container images")

	registry := getSettingValue(globalDir, "image_registry")
	if registry == "" {
		fmt.Printf("  Scion runs agents in containers. You need container images.\n\n")
		fmt.Printf("  %sOptions:%s\n", util.Bold, util.Reset)
		fmt.Printf("    %s1%s  I have images in a registry (ghcr.io/myorg, etc.)\n", util.Bold, util.Reset)
		fmt.Printf("    %s2%s  Build images locally now (requires Docker, ~15 min)\n", util.Bold, util.Reset)
		fmt.Printf("    %s3%s  Build via GitHub Actions (fork + workflow, ~5 min)\n", util.Bold, util.Reset)
		fmt.Printf("    %s4%s  Skip for now (configure later)\n\n", util.Bold, util.Reset)

		choice := prompt(reader, "Choose [1-4]", "4")

		switch choice {
		case "1":
			reg := prompt(reader, "Image registry path (e.g., ghcr.io/yourname)", "")
			if reg != "" {
				if err := config.UpdateSetting(globalDir, "image_registry", reg, true); err != nil {
					printFail(fmt.Sprintf("Failed to save: %v", err))
				} else {
					registry = reg
					printOK(fmt.Sprintf("Registry set to %s", reg))
				}
			}

		case "2":
			reg := prompt(reader, "Registry to push to (e.g., ghcr.io/yourname)", "")
			if reg == "" {
				printWarn("Registry required for image build. Skipping.")
			} else {
				fmt.Printf("\n  Building images (this takes ~15 minutes)...\n")
				fmt.Printf("  %sYou can continue with other work while this runs.%s\n\n", util.Dim, util.Reset)

				buildScript := findBuildScript()
				if buildScript == "" {
					printFail("Build script not found. Clone the scion repo and try again.")
					printHint("git clone https://github.com/GoogleCloudPlatform/scion && cd scion")
				} else {
					buildCmd := exec.Command(buildScript, "--registry", reg, "--push")
					buildCmd.Stdout = os.Stdout
					buildCmd.Stderr = os.Stderr
					if err := buildCmd.Run(); err != nil {
						printFail(fmt.Sprintf("Build failed: %v", err))
						printHint("Check Docker is running and you're authenticated to " + reg)
					} else {
						if err := config.UpdateSetting(globalDir, "image_registry", reg, true); err == nil {
							registry = reg
							printOK("Images built and registry configured")
						}
					}
				}
			}

		case "3":
			fmt.Println()
			fmt.Printf("  %sGitHub Actions build:%s\n", util.Bold, util.Reset)
			fmt.Printf("  1. Fork github.com/GoogleCloudPlatform/scion\n")
			fmt.Printf("  2. Go to Actions tab > 'Build Scion Images'\n")
			fmt.Printf("  3. Run workflow with your registry (e.g., ghcr.io/yourname)\n")
			fmt.Printf("  4. Wait ~5 minutes, then run:\n")
			fmt.Printf("     %sscion config set --global image_registry ghcr.io/yourname%s\n\n", util.Bold, util.Reset)
			printWarn("Skipping image setup — configure manually after build completes")

		default:
			printWarn("Skipping — run 'scion config set --global image_registry <reg>' when ready")
		}
	} else {
		printOK(fmt.Sprintf("Image registry: %s", registry))
	}
	fmt.Println()

	// ─── Step 4: API Keys ───────────────────────────────────────────
	printStep(4, 7, "API credentials")

	creds := detectCredentials()
	if len(creds) > 0 {
		for _, c := range creds {
			printOK(c)
		}
	} else {
		fmt.Printf("  No API keys detected. Let's set one up.\n\n")
		fmt.Printf("  %sWhich AI provider?%s\n", util.Bold, util.Reset)
		fmt.Printf("    %s1%s  Anthropic (Claude)        — ANTHROPIC_API_KEY\n", util.Bold, util.Reset)
		fmt.Printf("    %s2%s  Google (Gemini)            — GEMINI_API_KEY\n", util.Bold, util.Reset)
		fmt.Printf("    %s3%s  OpenAI (Codex)             — OPENAI_API_KEY\n", util.Bold, util.Reset)
		fmt.Printf("    %s4%s  Google Cloud (Vertex AI)   — Application Default Credentials\n", util.Bold, util.Reset)
		fmt.Printf("    %s5%s  Skip (configure later)\n\n", util.Bold, util.Reset)

		choice := prompt(reader, "Choose [1-5]", "5")

		var envVar, providerName, getKeyURL string
		switch choice {
		case "1":
			envVar, providerName, getKeyURL = "ANTHROPIC_API_KEY", "Anthropic", "https://console.anthropic.com/settings/keys"
		case "2":
			envVar, providerName, getKeyURL = "GEMINI_API_KEY", "Google Gemini", "https://aistudio.google.com/apikey"
		case "3":
			envVar, providerName, getKeyURL = "OPENAI_API_KEY", "OpenAI", "https://platform.openai.com/api-keys"
		case "4":
			fmt.Println()
			fmt.Printf("  Run: %sgcloud auth application-default login%s\n", util.Bold, util.Reset)
			fmt.Printf("  Then: %sexport GOOGLE_CLOUD_PROJECT=your-project%s\n", util.Bold, util.Reset)
			fmt.Printf("  Then: %sexport GOOGLE_CLOUD_REGION=us-central1%s\n\n", util.Bold, util.Reset)
		}

		if envVar != "" {
			fmt.Printf("\n  Get your %s API key: %s%s%s\n", providerName, util.Dim, getKeyURL, util.Reset)
			fmt.Printf("  Paste your API key (input hidden): ")

			keyBytes, err := term.ReadPassword(int(syscall.Stdin))
			fmt.Println() // newline after hidden input
			if err == nil && len(keyBytes) > 0 {
				key := strings.TrimSpace(string(keyBytes))
				if key != "" {
					os.Setenv(envVar, key)
					printOK(fmt.Sprintf("%s set for this session", envVar))

					// Offer to persist
					shell := detectShell()
					rcFile := shellRCFile(shell)
					if rcFile != "" {
						persist := prompt(reader, fmt.Sprintf("Add to %s for future sessions? [Y/n]", rcFile), "y")
						if persist == "y" || persist == "yes" || persist == "" {
							line := fmt.Sprintf("\nexport %s=\"%s\"\n", envVar, key)
							if f, err := os.OpenFile(rcFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600); err == nil {
								f.WriteString(line)
								f.Close()
								printOK(fmt.Sprintf("Added to %s", rcFile))
							}
						}
					}
				}
			}
		}
	}
	fmt.Println()

	// ─── Step 5: Project Grove ──────────────────────────────────────
	printStep(5, 7, "Project setup")

	if _, ok := config.FindProjectRoot(); !ok {
		doInit := prompt(reader, "Initialize grove in current directory? [Y/n]", "y")
		if doInit == "y" || doInit == "yes" || doInit == "" {
			if err := config.InitProject("", harness.All()); err != nil {
				printFail(fmt.Sprintf("Grove init failed: %v", err))
			} else {
				printOK("Grove initialized")
			}
		} else {
			printWarn("Skipped — run 'scion init' in your project directory later")
		}
	} else {
		printOK("Grove already initialized")
	}
	fmt.Println()

	// ─── Step 6: Shell Completions ──────────────────────────────────
	printStep(6, 7, "Shell completions")

	shell := detectShell()
	if shell != "" {
		installed := installCompletions(shell)
		if installed {
			printOK(fmt.Sprintf("Installed %s completions", shell))
		} else {
			printWarn(fmt.Sprintf("Run 'scion completion %s --help' for manual setup", shell))
		}
	} else {
		printWarn("Shell not detected — run 'scion completion --help' for setup")
	}
	fmt.Println()

	// ─── Step 7: Summary ────────────────────────────────────────────
	printStep(7, 7, "You're all set!")
	fmt.Println()

	// Show status summary
	hasImages := registry != ""
	hasCreds := len(detectCredentials()) > 0
	hasGrove := false
	if _, ok := config.FindProjectRoot(); ok {
		hasGrove = true
	}

	if hasImages && hasCreds && hasGrove {
		fmt.Printf("  %s%sEverything is configured. Start your first agent:%s\n\n", util.Bold, util.Green, util.Reset)
		fmt.Printf("    %sscion start my-agent \"Explore this codebase and suggest improvements\" --attach%s\n\n", util.Bold, util.Reset)
	} else {
		fmt.Printf("  %sSetup status:%s\n", util.Bold, util.Reset)
		printStatusLine("Container images", hasImages, "scion config set --global image_registry <reg>")
		printStatusLine("API credentials", hasCreds, "export ANTHROPIC_API_KEY=... (or GEMINI/OPENAI)")
		printStatusLine("Project grove", hasGrove, "cd your-project && scion init")
		fmt.Println()
	}

	fmt.Printf("  %sUseful commands:%s\n", util.Bold, util.Reset)
	fmt.Printf("    scion start <name> \"<task>\"    Start an agent\n")
	fmt.Printf("    scion list                      See running agents\n")
	fmt.Printf("    scion attach <name>             Connect to agent terminal\n")
	fmt.Printf("    scion marketplace list          Browse MCP servers & skills\n")
	fmt.Printf("    scion doctor                    Check system health\n")
	fmt.Println()

	return nil
}

// ─── UI Helpers ─────────────────────────────────────────────────────

func clearScreen() {
	fmt.Print("\033[2J\033[H")
}

func printBanner() {
	fmt.Println()
	fmt.Printf("  %s%s╔══════════════════════════════════════╗%s\n", util.Bold, util.Cyan, util.Reset)
	fmt.Printf("  %s%s║         Scion Setup Wizard           ║%s\n", util.Bold, util.Cyan, util.Reset)
	fmt.Printf("  %s%s║   Multi-Agent AI Orchestration       ║%s\n", util.Bold, util.Cyan, util.Reset)
	fmt.Printf("  %s%s╚══════════════════════════════════════╝%s\n\n", util.Bold, util.Cyan, util.Reset)
}

func printStep(n, total int, title string) {
	fmt.Printf("  %s%s[%d/%d]%s %s\n\n", util.Bold, util.Cyan, n, total, util.Reset, title)
}

func printOK(msg string) {
	fmt.Printf("  %s\u2713%s %s\n", util.Green, util.Reset, msg)
}

func printFail(msg string) {
	fmt.Printf("  %s\u2717%s %s\n", util.Red, util.Reset, msg)
}

func printWarn(msg string) {
	fmt.Printf("  %s!%s %s\n", util.Yellow, util.Reset, msg)
}

func printHint(msg string) {
	fmt.Printf("  %s%s%s\n", util.Dim, msg, util.Reset)
}

func printStatusLine(label string, ok bool, fix string) {
	if ok {
		fmt.Printf("    %s\u2713%s %s\n", util.Green, util.Reset, label)
	} else {
		fmt.Printf("    %s\u2717%s %s — %s%s%s\n", util.Red, util.Reset, label, util.Dim, fix, util.Reset)
	}
}

func prompt(reader *bufio.Reader, question, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", question, defaultVal)
	} else {
		fmt.Printf("  %s: ", question)
	}
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(answer)
	if answer == "" {
		return strings.ToLower(defaultVal)
	}
	return strings.ToLower(answer)
}

// ─── Detection Helpers ──────────────────────────────────────────────

type prereqResult struct {
	allPassed bool
	passed    []string
	failures  []string
}

func checkPrerequisites() prereqResult {
	result := prereqResult{allPassed: true}

	// Check git
	if _, err := exec.LookPath("git"); err != nil {
		result.allPassed = false
		result.failures = append(result.failures, "git not found — install from https://git-scm.com")
	} else {
		out, _ := exec.Command("git", "--version").Output()
		result.passed = append(result.passed, strings.TrimSpace(string(out)))
	}

	// Check container runtime
	runtimes := []string{"docker", "podman"}
	if runtime.GOOS == "darwin" {
		runtimes = append([]string{"container"}, runtimes...)
	}
	foundRuntime := false
	for _, rt := range runtimes {
		if _, err := exec.LookPath(rt); err == nil {
			result.passed = append(result.passed, rt+" available")
			foundRuntime = true
			break
		}
	}
	if !foundRuntime {
		result.allPassed = false
		result.failures = append(result.failures, "No container runtime — install Docker (docker.com) or Podman")
	}

	return result
}

func detectCredentials() []string {
	var found []string
	checks := []struct {
		env  string
		name string
	}{
		{"ANTHROPIC_API_KEY", "Anthropic API key (Claude)"},
		{"GEMINI_API_KEY", "Gemini API key"},
		{"GOOGLE_API_KEY", "Google API key"},
		{"OPENAI_API_KEY", "OpenAI API key (Codex)"},
	}
	for _, c := range checks {
		if v := os.Getenv(c.env); v != "" {
			// Show masked key
			masked := v
			if len(v) > 8 {
				masked = v[:4] + "..." + v[len(v)-4:]
			}
			found = append(found, fmt.Sprintf("%s (%s)", c.name, masked))
		}
	}

	// Check for ADC
	home, _ := os.UserHomeDir()
	if home != "" {
		adcPath := home + "/.config/gcloud/application_default_credentials.json"
		if _, err := os.Stat(adcPath); err == nil {
			found = append(found, "Google Application Default Credentials")
		}
	}

	return found
}

func detectShell() string {
	shell := os.Getenv("SHELL")
	if strings.Contains(shell, "zsh") {
		return "zsh"
	}
	if strings.Contains(shell, "bash") {
		return "bash"
	}
	if strings.Contains(shell, "fish") {
		return "fish"
	}
	return ""
}

func shellRCFile(shell string) string {
	home, _ := os.UserHomeDir()
	if home == "" {
		return ""
	}
	switch shell {
	case "zsh":
		return filepath.Join(home, ".zshrc")
	case "bash":
		rc := filepath.Join(home, ".bashrc")
		if runtime.GOOS == "darwin" {
			rc = filepath.Join(home, ".bash_profile")
		}
		return rc
	case "fish":
		return filepath.Join(home, ".config", "fish", "config.fish")
	}
	return ""
}

func installCompletions(shell string) bool {
	home, _ := os.UserHomeDir()
	if home == "" {
		return false
	}

	switch shell {
	case "zsh":
		// Write to a local completions dir
		dir := filepath.Join(home, ".zsh", "completions")
		os.MkdirAll(dir, 0755)
		out, err := exec.Command("scion", "completion", "zsh").Output()
		if err != nil {
			return false
		}
		return os.WriteFile(filepath.Join(dir, "_scion"), out, 0644) == nil

	case "bash":
		dir := "/etc/bash_completion.d"
		if runtime.GOOS == "darwin" {
			dir = "/usr/local/etc/bash_completion.d"
		}
		// Try user-local first
		localDir := filepath.Join(home, ".local", "share", "bash-completion", "completions")
		os.MkdirAll(localDir, 0755)
		out, err := exec.Command("scion", "completion", "bash").Output()
		if err != nil {
			return false
		}
		return os.WriteFile(filepath.Join(localDir, "scion"), out, 0644) == nil

	case "fish":
		dir := filepath.Join(home, ".config", "fish", "completions")
		os.MkdirAll(dir, 0755)
		out, err := exec.Command("scion", "completion", "fish").Output()
		if err != nil {
			return false
		}
		return os.WriteFile(filepath.Join(dir, "scion.fish"), out, 0644) == nil
	}
	return false
}

func findBuildScript() string {
	// Check common locations
	paths := []string{
		"image-build/scripts/build-images.sh",
		"../scion/image-build/scripts/build-images.sh",
	}
	// Also check GOPATH
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		paths = append(paths, filepath.Join(gopath, "src", "github.com", "GoogleCloudPlatform", "scion", "image-build", "scripts", "build-images.sh"))
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func getSettingValue(globalDir, key string) string {
	vs, _, err := config.LoadEffectiveSettings(globalDir)
	if err != nil || vs == nil {
		return ""
	}
	if key == "image_registry" {
		return vs.ResolveImageRegistry("")
	}
	return ""
}
