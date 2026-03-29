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
	"strings"
	"text/tabwriter"

	"github.com/GoogleCloudPlatform/scion/pkg/config"
	"github.com/GoogleCloudPlatform/scion/pkg/util"
	"github.com/spf13/cobra"
)

var imagesCmd = &cobra.Command{
	Use:   "images",
	Short: "Manage container images for agent harnesses",
	Long: `List, update, and build container images for agent harnesses.

The image hierarchy is:
  core-base     → Foundation (Go, Git, Node, Python, system tools)
  scion-base    → Adds scion + sciontool binaries
  scion-claude  → Adds Claude Code CLI
  scion-gemini  → Adds Gemini CLI
  scion-codex   → Adds Codex CLI

Updating a harness image only rebuilds the thin top layer (~30 seconds)
because the base layers are cached.`,
}

var imagesListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed scion container images with versions",
	RunE:  runImagesList,
}

var imagesUpdateCmd = &cobra.Command{
	Use:   "update [--harness <name>]",
	Short: "Rebuild harness image with latest upstream version (~30 seconds)",
	Long: `Rebuild a harness container image to pick up the latest upstream CLI version.

This only rebuilds the thin harness layer on top of the cached scion-base,
so it completes in ~30 seconds (not 15 minutes).

Examples:
  scion images update --harness claude    # Update Claude Code to latest
  scion images update --harness gemini    # Update Gemini CLI to latest
  scion images update --all               # Update all harnesses`,
	RunE: runImagesUpdate,
}

var imagesBuildCmd = &cobra.Command{
	Use:   "build --harness-config <name>",
	Short: "Build a custom harness image from a harness-config Dockerfile",
	Long: `Build a container image for a custom harness (ADK agents, Python agents, etc.)
using the Dockerfile defined in the harness-config.

The image is built on top of scion-base, inheriting all system tools
(Git, Node, Python, Go, tmux) needed for agent operation.

Example harness-config with custom Dockerfile:
  # ~/.scion/harness-configs/my-adk-agent/config.yaml
  harness: custom
  image: my-adk-agent:latest
  dockerfile: |
    ARG BASE_IMAGE
    FROM ${BASE_IMAGE}
    RUN pip install google-adk-agent
    CMD ["python", "-m", "my_agent"]

Then build: scion images build --harness-config my-adk-agent`,
	RunE: runImagesBuild,
}

var (
	imagesHarness    string
	imagesAll        bool
	imagesHarnessConfig string
)

func init() {
	rootCmd.AddCommand(imagesCmd)
	imagesCmd.AddCommand(imagesListCmd)
	imagesCmd.AddCommand(imagesUpdateCmd)
	imagesCmd.AddCommand(imagesBuildCmd)

	imagesUpdateCmd.Flags().StringVar(&imagesHarness, "harness", "", "Harness to update (claude, gemini, codex, opencode)")
	imagesUpdateCmd.Flags().BoolVar(&imagesAll, "all", false, "Update all harness images")

	imagesBuildCmd.Flags().StringVar(&imagesHarnessConfig, "harness-config", "", "Harness config name to build")
	imagesBuildCmd.MarkFlagRequired("harness-config")
}

func runImagesList(cmd *cobra.Command, args []string) error {
	globalDir, _ := config.GetGlobalDir()
	registry := ""
	if vs, _, err := config.LoadEffectiveSettings(globalDir); err == nil && vs != nil {
		registry = vs.ResolveImageRegistry("")
	}

	if registry == "" {
		fmt.Printf("%sNo image registry configured.%s\n", util.Yellow, util.Reset)
		fmt.Printf("Run: scion config set --global image_registry <registry>\n\n")
		return nil
	}

	// Check which images exist locally
	harnesses := []struct {
		name  string
		image string
	}{
		{"claude", registry + "/scion-claude:latest"},
		{"gemini", registry + "/scion-gemini:latest"},
		{"codex", registry + "/scion-codex:latest"},
		{"opencode", registry + "/scion-opencode:latest"},
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "HARNESS\tIMAGE\tSTATUS\tSIZE\tCREATED")
	for _, h := range harnesses {
		status := "not found"
		size := "-"
		created := "-"

		// Check if image exists locally via docker inspect
		out, err := exec.Command("docker", "inspect", "--format",
			`{{.Size}}|{{.Created}}`, h.image).Output()
		if err == nil {
			parts := strings.SplitN(strings.TrimSpace(string(out)), "|", 2)
			status = "available"
			if len(parts) >= 1 {
				if s, err := fmt.Sscanf(parts[0], "%s", &size); err == nil && s > 0 {
					// Convert bytes to human-readable
					var sizeBytes int64
					fmt.Sscanf(parts[0], "%d", &sizeBytes)
					size = formatBytes(sizeBytes)
				}
			}
			if len(parts) >= 2 && len(parts[1]) > 10 {
				created = parts[1][:10] // Just the date
			}
		}

		statusColor := util.Red
		if status == "available" {
			statusColor = util.Green
		}

		fmt.Fprintf(w, "%s\t%s\t%s%s%s\t%s\t%s\n",
			h.name, h.image, statusColor, status, util.Reset, size, created)
	}
	w.Flush()

	fmt.Printf("\n%sRegistry:%s %s\n", util.Bold, util.Reset, registry)
	fmt.Printf("%sUpdate:%s scion images update --harness <name>\n\n", util.Dim, util.Reset)
	return nil
}

func runImagesUpdate(cmd *cobra.Command, args []string) error {
	if !imagesAll && imagesHarness == "" {
		return fmt.Errorf("specify --harness <name> or --all\n\nAvailable: claude, gemini, codex, opencode")
	}

	globalDir, _ := config.GetGlobalDir()
	registry := ""
	if vs, _, err := config.LoadEffectiveSettings(globalDir); err == nil && vs != nil {
		registry = vs.ResolveImageRegistry("")
	}
	if registry == "" {
		return fmt.Errorf("image_registry not configured. Run: scion config set --global image_registry <registry>")
	}

	harnesses := []string{imagesHarness}
	if imagesAll {
		harnesses = []string{"claude", "gemini", "codex", "opencode"}
	}

	baseImage := registry + "/scion-base:latest"

	for _, h := range harnesses {
		fmt.Printf("\n%sUpdating %s...%s\n", util.Bold, h, util.Reset)

		// Build a minimal Dockerfile that just updates the harness npm package
		var npmPkg string
		switch h {
		case "claude":
			npmPkg = "@anthropic-ai/claude-code"
		case "gemini":
			npmPkg = "@google/gemini-cli"
		case "codex":
			npmPkg = "@openai/codex"
		case "opencode":
			npmPkg = "opencode-ai"
		default:
			fmt.Printf("  Unknown harness: %s\n", h)
			continue
		}

		dockerfile := fmt.Sprintf(
			"FROM %s\nRUN npm install -g %s@latest && npm cache clean --force\n",
			baseImage, npmPkg)

		targetImage := fmt.Sprintf("%s/scion-%s:latest", registry, h)

		// Build using docker build with stdin Dockerfile
		buildCmd := exec.Command("docker", "build", "-t", targetImage, "-f", "-", ".")
		buildCmd.Stdin = strings.NewReader(dockerfile)
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr

		if err := buildCmd.Run(); err != nil {
			fmt.Printf("  %sFailed: %v%s\n", util.Red, err, util.Reset)
			continue
		}

		fmt.Printf("  %s%s Updated %s%s\n", util.Bold, util.Green, targetImage, util.Reset)

		// Push if registry is remote
		if strings.Contains(registry, "/") || strings.Contains(registry, ".") {
			fmt.Printf("  Pushing %s...\n", targetImage)
			pushCmd := exec.Command("docker", "push", targetImage)
			pushCmd.Stdout = os.Stdout
			pushCmd.Stderr = os.Stderr
			if err := pushCmd.Run(); err != nil {
				fmt.Printf("  %sPush failed: %v%s\n", util.Yellow, err, util.Reset)
				fmt.Printf("  Image built locally. Push manually: docker push %s\n", targetImage)
			}
		}
	}

	fmt.Printf("\n%sDone.%s\n", util.Bold, util.Reset)
	return nil
}

func runImagesBuild(cmd *cobra.Command, args []string) error {
	globalDir, _ := config.GetGlobalDir()
	registry := ""
	if vs, _, err := config.LoadEffectiveSettings(globalDir); err == nil && vs != nil {
		registry = vs.ResolveImageRegistry("")
	}
	if registry == "" {
		return fmt.Errorf("image_registry not configured")
	}

	// Load the harness config
	hcDir := findHarnessConfigDirLocal(imagesHarnessConfig, globalDir)
	if hcDir == "" {
		return fmt.Errorf("harness-config %q not found", imagesHarnessConfig)
	}

	// Read config.yaml for dockerfile field
	configPath := hcDir + "/config.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	// Simple extraction of dockerfile field (avoid full YAML parse dependency)
	var dockerfileContent string
	lines := strings.Split(string(data), "\n")
	inDockerfile := false
	for _, line := range lines {
		if strings.HasPrefix(line, "dockerfile:") {
			inDockerfile = true
			// Check for inline value
			val := strings.TrimPrefix(line, "dockerfile:")
			val = strings.TrimSpace(val)
			if val != "" && val != "|" {
				dockerfileContent = val
				inDockerfile = false
			}
			continue
		}
		if inDockerfile {
			if len(line) > 0 && line[0] != ' ' && line[0] != '\t' {
				break // End of block
			}
			dockerfileContent += strings.TrimPrefix(strings.TrimPrefix(line, "  "), "\t") + "\n"
		}
	}

	if dockerfileContent == "" {
		return fmt.Errorf("harness-config %q has no 'dockerfile' field in config.yaml\n\nAdd a dockerfile field:\n  dockerfile: |\n    ARG BASE_IMAGE\n    FROM ${BASE_IMAGE}\n    RUN pip install my-package\n    CMD [\"my-command\"]", imagesHarnessConfig)
	}

	// Extract image name from config
	imageName := ""
	for _, line := range lines {
		if strings.HasPrefix(line, "image:") {
			imageName = strings.TrimSpace(strings.TrimPrefix(line, "image:"))
			break
		}
	}
	if imageName == "" {
		imageName = fmt.Sprintf("scion-%s:latest", imagesHarnessConfig)
	}

	// Ensure BASE_IMAGE arg is set
	baseImage := registry + "/scion-base:latest"
	if !strings.Contains(dockerfileContent, "BASE_IMAGE") {
		dockerfileContent = fmt.Sprintf("ARG BASE_IMAGE=%s\n%s", baseImage, dockerfileContent)
	}

	fmt.Printf("Building %s from harness-config %q...\n", imageName, imagesHarnessConfig)

	buildCmd := exec.Command("docker", "build",
		"-t", imageName,
		"--build-arg", "BASE_IMAGE="+baseImage,
		"-f", "-", ".")
	buildCmd.Stdin = strings.NewReader(dockerfileContent)
	buildCmd.Stdout = os.Stdout
	buildCmd.Stderr = os.Stderr

	if err := buildCmd.Run(); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("%s%s Built %s%s\n", util.Bold, util.Green, imageName, util.Reset)
	return nil
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}

// FindHarnessConfigDir looks up a harness config directory by name.
// This is added to config package below but duplicated here for the command.
func findHarnessConfigDirLocal(name, globalDir string) string {
	// Check grove-level first, then global
	if projectDir, _ := config.GetResolvedProjectDir(grovePath); projectDir != "" {
		dir := projectDir + "/harness-configs/" + name
		if _, err := os.Stat(dir); err == nil {
			return dir
		}
	}
	dir := globalDir + "/harness-configs/" + name
	if _, err := os.Stat(dir); err == nil {
		return dir
	}
	return ""
}

