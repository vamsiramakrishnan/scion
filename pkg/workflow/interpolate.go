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

import (
	"regexp"
	"strings"
)

var stepVarPattern = regexp.MustCompile(`\$\{steps\.([a-zA-Z0-9_-]+)\.([a-zA-Z0-9_.]+)\}`)

// InterpolateTask resolves ${steps.X.Y} variables in a task string
// using outputs from completed steps.
func InterpolateTask(task string, state *WorkflowState) string {
	if state == nil || len(state.Outputs) == 0 {
		return task
	}

	return stepVarPattern.ReplaceAllStringFunc(task, func(match string) string {
		parts := stepVarPattern.FindStringSubmatch(match)
		if len(parts) != 3 {
			return match
		}
		stepName, field := parts[1], parts[2]

		output, ok := state.Outputs[stepName]
		if !ok {
			return match // step not found, leave variable as-is
		}

		// Handle nested artifact references: artifacts.key
		if strings.HasPrefix(field, "artifacts.") {
			key := strings.TrimPrefix(field, "artifacts.")
			if val, ok := output.Artifacts[key]; ok {
				return val
			}
			return match
		}

		// Handle metadata references: metadata.key
		if strings.HasPrefix(field, "metadata.") {
			key := strings.TrimPrefix(field, "metadata.")
			if val, ok := output.Metadata[key]; ok {
				return val
			}
			return match
		}

		switch field {
		case "branch":
			return output.Branch
		case "summary":
			return output.Summary
		case "status":
			return output.Status
		case "agent_id":
			return output.AgentID
		case "step_name":
			return output.StepName
		default:
			return match
		}
	})
}
