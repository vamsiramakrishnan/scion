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

package workflow

// Workflow is the top-level workflow definition parsed from YAML.
type Workflow struct {
	Name        string            `yaml:"name" json:"name"`
	Description string            `yaml:"description,omitempty" json:"description,omitempty"`
	Env         map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
	Steps       []Step            `yaml:"steps" json:"steps"`
}

// Step defines a single agent step in a workflow.
type Step struct {
	Name      string            `yaml:"name" json:"name"`
	Template  string            `yaml:"template,omitempty" json:"template,omitempty"`
	Task      string            `yaml:"task" json:"task"`
	DependsOn []string          `yaml:"depends_on,omitempty" json:"dependsOn,omitempty"`
	Branch    string            `yaml:"branch,omitempty" json:"branch,omitempty"`
	Image     string            `yaml:"image,omitempty" json:"image,omitempty"`
	Env       map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

// StepStatus tracks runtime state of a workflow step.
type StepStatus struct {
	Name    string `json:"name"`
	AgentID string `json:"agentId,omitempty"`
	State   string `json:"state"` // pending, running, completed, failed, skipped
	Error   string `json:"error,omitempty"`
}

// WorkflowStatus tracks overall workflow execution state.
type WorkflowStatus struct {
	Name  string       `json:"name"`
	State string       `json:"state"` // running, completed, failed
	Steps []StepStatus `json:"steps"`
}
