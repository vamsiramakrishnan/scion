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
	Outputs   []string          `yaml:"outputs,omitempty" json:"outputs,omitempty"`
	Gate      string            `yaml:"gate,omitempty" json:"gate,omitempty"` // "approval", "review", or "" (auto)
}

// StepStatus tracks runtime state of a workflow step.
type StepStatus struct {
	Name    string `json:"name"`
	AgentID string `json:"agentId,omitempty"`
	State   string `json:"state"` // pending, running, completed, failed, skipped
	Error   string `json:"error,omitempty"`
}

// StepOutput captures the output of a completed workflow step.
type StepOutput struct {
	StepName  string            `json:"step_name" yaml:"step_name"`
	AgentID   string            `json:"agent_id,omitempty" yaml:"agent_id,omitempty"`
	Status    string            `json:"status" yaml:"status"`
	Branch    string            `json:"branch,omitempty" yaml:"branch,omitempty"`
	Summary   string            `json:"summary,omitempty" yaml:"summary,omitempty"`
	Artifacts map[string]string `json:"artifacts,omitempty" yaml:"artifacts,omitempty"`
	Metadata  map[string]string `json:"metadata,omitempty" yaml:"metadata,omitempty"`
}

// WorkflowState is the shared state for a workflow execution.
type WorkflowState struct {
	Outputs map[string]StepOutput `json:"outputs"` // keyed by step name
}

// WorkflowStatus tracks overall workflow execution state.
type WorkflowStatus struct {
	Name    string                `json:"name"`
	State   string                `json:"state"` // running, completed, failed
	Steps   []StepStatus          `json:"steps"`
	Outputs map[string]StepOutput `json:"outputs,omitempty"` // accumulated step outputs
}
