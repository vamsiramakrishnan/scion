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
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/scion/pkg/agent"
	"github.com/GoogleCloudPlatform/scion/pkg/api"
	"github.com/GoogleCloudPlatform/scion/pkg/runtime"
	"github.com/GoogleCloudPlatform/scion/pkg/util"
	"github.com/GoogleCloudPlatform/scion/pkg/workflow"
	"github.com/spf13/cobra"
)

var workflowCmd = &cobra.Command{
	Use:   "workflow",
	Short: "Run declarative multi-agent workflows",
	Long: `Define and execute multi-agent workflows from YAML files.

A workflow file describes a set of agent steps with dependencies. Steps with
no dependencies run in parallel. Steps wait for their dependencies to complete
before starting.

Example workflow.yaml:
  name: code-review
  steps:
    - name: reviewer
      template: code-reviewer
      task: "Review code for bugs and quality"
    - name: security
      template: security-reviewer
      task: "Audit for security vulnerabilities"
    - name: summary
      template: default
      task: "Summarize review findings and create a report"
      depends_on: [reviewer, security]`,
}

var workflowRunCmd = &cobra.Command{
	Use:   "run <workflow.yaml>",
	Short: "Execute a workflow from a YAML definition",
	Long: `Execute a multi-agent workflow defined in a YAML file.

Steps run in parallel where possible, respecting dependency ordering.
The command blocks until all steps complete, showing real-time status.

Examples:
  scion workflow run review-pipeline.yaml
  scion workflow run deploy.yaml --format json`,
	Args: cobra.ExactArgs(1),
	RunE: runWorkflowRun,
}

var workflowValidateCmd = &cobra.Command{
	Use:   "validate <workflow.yaml>",
	Short: "Validate a workflow file without executing it",
	Args:  cobra.ExactArgs(1),
	RunE:  runWorkflowValidate,
}

func init() {
	rootCmd.AddCommand(workflowCmd)
	workflowCmd.AddCommand(workflowRunCmd)
	workflowCmd.AddCommand(workflowValidateCmd)
}

func runWorkflowRun(cmd *cobra.Command, args []string) error {
	w, err := workflow.LoadWorkflow(args[0])
	if err != nil {
		return err
	}

	fmt.Printf("%sWorkflow: %s%s\n", util.Bold, w.Name, util.Reset)
	if w.Description != "" {
		fmt.Printf("%s%s%s\n", util.Gray, w.Description, util.Reset)
	}
	fmt.Printf("Steps: %d\n\n", len(w.Steps))

	// Print execution plan
	for i, s := range w.Steps {
		deps := ""
		if len(s.DependsOn) > 0 {
			deps = fmt.Sprintf(" (after: %s)", strings.Join(s.DependsOn, ", "))
		}
		tmpl := s.Template
		if tmpl == "" {
			tmpl = "default"
		}
		fmt.Printf("  %d. %s [%s]%s\n", i+1, s.Name, tmpl, deps)
	}
	fmt.Println()

	// Create launcher
	rt := agent.ResolveRuntime(grovePath, "", profile)
	launcher := &cliAgentLauncher{rt: rt}

	// Create engine with live status output
	engine := workflow.NewEngine(launcher, func(status *workflow.WorkflowStatus) {
		printWorkflowStatus(status)
	})

	ctx := context.Background()
	result, err := engine.Run(ctx, w)

	fmt.Println()
	if err != nil {
		fmt.Printf("%s%sWorkflow failed: %v%s\n", util.Bold, util.Red, err, util.Reset)
		if isJSONOutput() {
			json.NewEncoder(os.Stdout).Encode(result)
		}
		return err
	}

	fmt.Printf("%s%sWorkflow completed successfully%s\n", util.Bold, util.Green, util.Reset)
	if isJSONOutput() {
		json.NewEncoder(os.Stdout).Encode(result)
	}
	return nil
}

func runWorkflowValidate(cmd *cobra.Command, args []string) error {
	w, err := workflow.LoadWorkflow(args[0])
	if err != nil {
		return err
	}
	fmt.Printf("%s%sValid workflow: %s%s (%d steps)\n",
		util.Bold, util.Green, w.Name, util.Reset, len(w.Steps))

	for i, s := range w.Steps {
		deps := ""
		if len(s.DependsOn) > 0 {
			deps = fmt.Sprintf(" -> depends on: %s", strings.Join(s.DependsOn, ", "))
		}
		fmt.Printf("  %d. %s%s\n", i+1, s.Name, deps)
	}
	return nil
}

func printWorkflowStatus(status *workflow.WorkflowStatus) {
	// Clear line and print compact status
	var parts []string
	for _, s := range status.Steps {
		var icon string
		switch s.State {
		case "pending":
			icon = util.Gray + "○" + util.Reset
		case "running":
			icon = util.Yellow + "●" + util.Reset
		case "completed":
			icon = util.Green + "✓" + util.Reset
		case "failed":
			icon = util.Red + "✗" + util.Reset
		case "skipped":
			icon = util.Gray + "⊘" + util.Reset
		}
		parts = append(parts, fmt.Sprintf("%s %s", icon, s.Name))
	}
	fmt.Printf("\r  %s", strings.Join(parts, "  "))
}

// cliAgentLauncher implements workflow.AgentLauncher using the local agent manager.
type cliAgentLauncher struct {
	rt runtime.Runtime
}

func (l *cliAgentLauncher) StartAgent(ctx context.Context, name, task, template, branch, image string, env map[string]string) (string, error) {
	mgr := agent.NewManager(l.rt)
	defer mgr.Close()
	opts := api.StartOptions{
		Name:      name,
		Task:      task,
		Template:  template,
		Branch:    branch,
		Image:     image,
		GrovePath: grovePath,
		Env:       env,
	}
	result, err := mgr.Start(ctx, opts)
	if err != nil {
		return "", err
	}
	// Prefer container ID, fall back to name
	if result.ContainerID != "" {
		return result.ContainerID, nil
	}
	return result.Name, nil
}

func (l *cliAgentLauncher) WaitForAgent(ctx context.Context, agentID string) (string, error) {
	// Poll agent status until terminal
	mgr := agent.NewManager(l.rt)
	defer mgr.Close()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			agents, err := mgr.List(ctx, nil)
			if err != nil {
				continue
			}
			for _, a := range agents {
				if a.ContainerID == agentID || a.Name == agentID {
					activity := strings.ToLower(a.Activity)
					phase := strings.ToLower(a.Phase)
					// Check terminal states via activity or phase
					switch {
					case activity == "completed" || activity == "task_completed":
						return "completed", nil
					case phase == "stopped" || phase == "error":
						return phase, nil
					case activity == "failed" || activity == "error":
						return activity, nil
					}
				}
			}
		}
	}
}

func (l *cliAgentLauncher) StopAgent(ctx context.Context, agentID string) error {
	return l.rt.Stop(ctx, agentID)
}
