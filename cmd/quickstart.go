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
	"strings"

	"github.com/GoogleCloudPlatform/scion/pkg/config"
	"github.com/GoogleCloudPlatform/scion/pkg/harness"
	"github.com/GoogleCloudPlatform/scion/pkg/util"
	"github.com/spf13/cobra"
)

var quickstartCmd = &cobra.Command{
	Use:   "quickstart",
	Short: "Guided setup for new users — get running in 60 seconds",
	Long: `Interactive guided setup that handles machine init, grove setup,
credential discovery, and launches your first agent.

This is the recommended way to get started with scion. It combines
'scion init --machine', 'scion init', and 'scion start' into a
single guided flow with helpful explanations at each step.`,
	RunE: runQuickstart,
}

func init() {
	rootCmd.AddCommand(quickstartCmd)
}

func runQuickstart(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Header
	fmt.Println()
	fmt.Printf("%s%s  Scion Quickstart%s\n", util.Bold, util.Cyan, util.Reset)
	fmt.Printf("%s  Multi-agent orchestration for AI coding assistants%s\n\n", util.Dim, util.Reset)

	// Step 1: Check prerequisites
	fmt.Printf("%s[1/5]%s Checking prerequisites...\n", util.Bold, util.Reset)
	prereqs := checkPrerequisites()
	if !prereqs.allPassed {
		fmt.Printf("\n%s%s  Some prerequisites are missing:%s\n", util.Bold, util.Red, util.Reset)
		for _, msg := range prereqs.failures {
			fmt.Printf("  %s%s  %s%s\n", util.Bold, util.Red, msg, util.Reset)
		}
		fmt.Println()
		return fmt.Errorf("fix prerequisites and try again")
	}
	for _, msg := range prereqs.passed {
		fmt.Printf("  %s%s  %s%s\n", util.Bold, util.Green, msg, util.Reset)
	}
	fmt.Println()

	// Step 2: Machine setup (if needed)
	globalDir, _ := config.GetGlobalDir()
	needsMachineInit := false
	if _, err := os.Stat(globalDir); os.IsNotExist(err) {
		needsMachineInit = true
	}

	if needsMachineInit {
		fmt.Printf("%s[2/5]%s Setting up scion on this machine...\n", util.Bold, util.Reset)
		if err := config.InitMachine(harness.All(), config.InitMachineOpts{}); err != nil {
			return fmt.Errorf("machine init failed: %w", err)
		}
		fmt.Printf("  %s%s  Created %s%s\n", util.Bold, util.Green, globalDir, util.Reset)
	} else {
		fmt.Printf("%s[2/5]%s Machine already set up %s%s%s\n", util.Bold, util.Reset, util.Dim, globalDir, util.Reset)
	}
	fmt.Println()

	// Step 3: Detect credentials
	fmt.Printf("%s[3/5]%s Detecting credentials...\n", util.Bold, util.Reset)
	creds := detectCredentials()
	if len(creds) == 0 {
		fmt.Printf("  %s%s  No API keys found%s\n", util.Bold, util.Yellow, util.Reset)
		fmt.Printf("  Set one of: ANTHROPIC_API_KEY, GEMINI_API_KEY, OPENAI_API_KEY\n")
		fmt.Printf("  Or configure Vertex AI / Application Default Credentials\n\n")
	} else {
		for _, c := range creds {
			fmt.Printf("  %s%s  %s%s\n", util.Bold, util.Green, c, util.Reset)
		}
		fmt.Println()
	}

	// Step 4: Grove setup (if needed)
	fmt.Printf("%s[4/5]%s Checking project grove...\n", util.Bold, util.Reset)
	if _, ok := config.FindProjectRoot(); !ok {
		fmt.Printf("  No grove found in current directory.\n")
		fmt.Printf("  Initialize grove here? [Y/n] ")
		answer, _ := reader.ReadString('\n')
		answer = strings.TrimSpace(strings.ToLower(answer))
		if answer == "" || answer == "y" || answer == "yes" {
			if err := config.InitProject(config.InitProjectOpts{}); err != nil {
				return fmt.Errorf("grove init failed: %w", err)
			}
			fmt.Printf("  %s%s  Grove initialized%s\n", util.Bold, util.Green, util.Reset)
		}
	} else {
		fmt.Printf("  %s%s  Grove already initialized%s\n", util.Bold, util.Green, util.Reset)
	}
	fmt.Println()

	// Step 5: Show available templates and offer to start an agent
	fmt.Printf("%s[5/5]%s Ready to go!\n\n", util.Bold, util.Reset)
	fmt.Printf("  %sAvailable templates:%s\n", util.Bold, util.Reset)
	fmt.Printf("    default          General-purpose coding agent\n")
	fmt.Printf("    fullstack-dev    Fullstack developer (implements features end-to-end)\n")
	fmt.Printf("    code-reviewer    Reviews code for bugs, security, and quality\n")
	fmt.Printf("    security-reviewer  Security audit (OWASP Top 10, CVE checks)\n")
	fmt.Printf("    architect        System design and implementation planning\n")
	fmt.Printf("    docs-writer      Documentation and technical writing\n")
	fmt.Println()
	fmt.Printf("  %sQuick commands:%s\n", util.Bold, util.Reset)
	fmt.Printf("    scion start my-agent \"Fix the login bug\"          %s# Start with default template%s\n", util.Dim, util.Reset)
	fmt.Printf("    scion start my-agent --type code-reviewer --attach %s# Start a code reviewer%s\n", util.Dim, util.Reset)
	fmt.Printf("    scion list                                          %s# See running agents%s\n", util.Dim, util.Reset)
	fmt.Printf("    scion attach my-agent                               %s# Connect to agent terminal%s\n", util.Dim, util.Reset)
	fmt.Printf("    scion templates list                                %s# See all templates%s\n", util.Dim, util.Reset)
	fmt.Println()

	return nil
}

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
		result.failures = append(result.failures, "git not found — install git (https://git-scm.com)")
	} else {
		result.passed = append(result.passed, "git found")
	}

	// Check tmux
	if _, err := exec.LookPath("tmux"); err != nil {
		// tmux is needed inside containers, not on host — just warn
		result.passed = append(result.passed, "tmux not found (OK — only needed in agent containers)")
	} else {
		result.passed = append(result.passed, "tmux found")
	}

	// Check container runtime
	runtimes := []string{"docker", "podman"}
	foundRuntime := false
	for _, rt := range runtimes {
		if _, err := exec.LookPath(rt); err == nil {
			result.passed = append(result.passed, rt+" found")
			foundRuntime = true
			break
		}
	}
	if !foundRuntime {
		result.allPassed = false
		result.failures = append(result.failures, "No container runtime found — install Docker or Podman")
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
		{"GOOGLE_APPLICATION_CREDENTIALS", "Google Application Default Credentials"},
		{"GOOGLE_CLOUD_PROJECT", "Google Cloud project configured"},
	}
	for _, c := range checks {
		if os.Getenv(c.env) != "" {
			found = append(found, c.name)
		}
	}

	// Check for ADC file
	home, _ := os.UserHomeDir()
	if home != "" {
		adcPath := home + "/.config/gcloud/application_default_credentials.json"
		if _, err := os.Stat(adcPath); err == nil {
			found = append(found, "Google ADC credentials file")
		}
	}

	return found
}
