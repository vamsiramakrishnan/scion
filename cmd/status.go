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
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/GoogleCloudPlatform/scion/pkg/agent"
	"github.com/GoogleCloudPlatform/scion/pkg/config"
	"github.com/GoogleCloudPlatform/scion/pkg/runtime"
	"github.com/GoogleCloudPlatform/scion/pkg/util"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show system health, running agents, and configuration at a glance",
	Long: `Display a single dashboard showing:
  - System health (container runtime, credentials, image registry)
  - Current grove context
  - Running agents with their current status
  - Warnings and issues that need attention

This is like 'git status' for your entire scion setup.`,
	RunE: runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	fmt.Println()

	// ─── System Health ──────────────────────────────────────────────
	fmt.Printf("%sScion Status%s\n\n", util.Bold, util.Reset)

	// Container runtime
	runtimeName := "none"
	for _, rt := range []string{"docker", "podman"} {
		if _, err := exec.LookPath(rt); err == nil {
			out, err := exec.Command(rt, "info", "--format", "{{.ServerVersion}}").Output()
			if err == nil {
				runtimeName = fmt.Sprintf("%s %s", rt, strings.TrimSpace(string(out)))
			} else {
				runtimeName = rt + " (not responding)"
			}
			break
		}
	}
	printStatusItem("Runtime", runtimeName, runtimeName != "none" && !strings.Contains(runtimeName, "not responding"))

	// Image registry
	globalDir, _ := config.GetGlobalDir()
	registry := ""
	if vs, _, err := config.LoadEffectiveSettings(globalDir); err == nil && vs != nil {
		registry = vs.ResolveImageRegistry("")
	}
	if registry != "" {
		printStatusItem("Images", registry, true)
	} else {
		printStatusItem("Images", "not configured — run: scion config set --global image_registry <reg>", false)
	}

	// Credentials
	creds := []string{}
	credChecks := map[string]string{
		"ANTHROPIC_API_KEY": "Claude",
		"GEMINI_API_KEY":    "Gemini",
		"OPENAI_API_KEY":    "Codex",
	}
	for env, name := range credChecks {
		if os.Getenv(env) != "" {
			creds = append(creds, name)
		}
	}
	if home, _ := os.UserHomeDir(); home != "" {
		if _, err := os.Stat(home + "/.config/gcloud/application_default_credentials.json"); err == nil {
			creds = append(creds, "GCP ADC")
		}
	}
	if len(creds) > 0 {
		printStatusItem("Credentials", strings.Join(creds, ", "), true)
	} else {
		printStatusItem("Credentials", "none — export ANTHROPIC_API_KEY, GEMINI_API_KEY, or OPENAI_API_KEY", false)
	}

	fmt.Println()

	// ─── Grove Context ──────────────────────────────────────────────
	projectDir, ok := config.FindProjectRoot()
	if ok {
		groveName := config.GetGroveName(projectDir)
		fmt.Printf("%sGrove:%s %s\n", util.Bold, util.Reset, groveName)
		fmt.Printf("  Path: %s\n\n", projectDir)
	} else {
		fmt.Printf("%sGrove:%s %snot in a grove — run: scion init%s\n\n", util.Bold, util.Reset, util.Yellow, util.Reset)
	}

	// ─── Running Agents ─────────────────────────────────────────────
	if ok {
		rt := runtime.GetRuntime(grovePath, profile)
		mgr := agent.NewManager(rt)

		agents, err := mgr.List(context.Background(), map[string]string{
			"scion.agent": "true",
		})
		if err == nil && len(agents) > 0 {
			fmt.Printf("%sAgents:%s\n", util.Bold, util.Reset)
			for _, a := range agents {
				name := a.Labels["scion.name"]
				if name == "" {
					name = a.Name
				}
				status := a.ContainerStatus
				phase := a.Labels["scion.phase"]
				if phase == "" {
					phase = a.Phase
				}

				icon := "\u25CB" // circle
				color := util.Dim
				if strings.HasPrefix(strings.ToLower(status), "up") || status == "running" {
					icon = "\u25CF" // filled circle
					color = util.Green
				} else if strings.Contains(strings.ToLower(status), "exit") {
					icon = "\u25CF"
					color = util.Red
				}

				template := a.Labels["scion.template"]
				fmt.Printf("  %s%s%s %-20s %-12s %s\n", color, icon, util.Reset, name, phase, template)
			}
		} else {
			fmt.Printf("%sAgents:%s none running\n", util.Bold, util.Reset)
			fmt.Printf("  Start one: %sscion start my-agent \"task\" --attach%s\n", util.Bold, util.Reset)
		}
	}

	fmt.Println()
	return nil
}

func printStatusItem(label, value string, ok bool) {
	if ok {
		fmt.Printf("  %s\u2713%s %-14s %s\n", util.Green, util.Reset, label, value)
	} else {
		fmt.Printf("  %s\u2717%s %-14s %s%s%s\n", util.Red, util.Reset, label, util.Yellow, value, util.Reset)
	}
}
