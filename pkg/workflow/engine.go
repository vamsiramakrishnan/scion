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
	"context"
	"fmt"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// LoadWorkflow parses a workflow YAML file.
func LoadWorkflow(path string) (*Workflow, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read workflow: %w", err)
	}
	var w Workflow
	if err := yaml.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("parse workflow: %w", err)
	}
	if w.Name == "" {
		return nil, fmt.Errorf("workflow name is required")
	}
	if len(w.Steps) == 0 {
		return nil, fmt.Errorf("workflow must have at least one step")
	}
	// Validate dependencies exist
	stepNames := make(map[string]bool)
	for _, s := range w.Steps {
		if s.Name == "" {
			return nil, fmt.Errorf("all steps must have a name")
		}
		if stepNames[s.Name] {
			return nil, fmt.Errorf("duplicate step name: %s", s.Name)
		}
		stepNames[s.Name] = true
	}
	for _, s := range w.Steps {
		for _, dep := range s.DependsOn {
			if !stepNames[dep] {
				return nil, fmt.Errorf("step %q depends on unknown step %q", s.Name, dep)
			}
		}
	}
	// Check for cycles
	if err := detectCycles(&w); err != nil {
		return nil, err
	}
	return &w, nil
}

// detectCycles checks for circular dependencies using DFS.
func detectCycles(w *Workflow) error {
	depMap := make(map[string][]string)
	for _, s := range w.Steps {
		depMap[s.Name] = s.DependsOn
	}
	visited := make(map[string]int) // 0=unvisited, 1=visiting, 2=done
	var visit func(name string) error
	visit = func(name string) error {
		if visited[name] == 1 {
			return fmt.Errorf("circular dependency detected involving step %q", name)
		}
		if visited[name] == 2 {
			return nil
		}
		visited[name] = 1
		for _, dep := range depMap[name] {
			if err := visit(dep); err != nil {
				return err
			}
		}
		visited[name] = 2
		return nil
	}
	for _, s := range w.Steps {
		if err := visit(s.Name); err != nil {
			return err
		}
	}
	return nil
}

// AgentLauncher is the interface the engine uses to start agents.
type AgentLauncher interface {
	// StartAgent launches an agent and returns its ID.
	StartAgent(ctx context.Context, name, task, template, branch, image string, env map[string]string) (string, error)
	// WaitForAgent blocks until the agent reaches a terminal state.
	// Returns the final status ("completed", "failed", "error", etc.)
	WaitForAgent(ctx context.Context, agentID string) (string, error)
}

// Engine executes a workflow by launching agents according to the dependency graph.
type Engine struct {
	launcher AgentLauncher
	status   *WorkflowStatus
	mu       sync.Mutex
	onUpdate func(*WorkflowStatus) // optional callback on status change
}

// NewEngine creates a workflow execution engine.
func NewEngine(launcher AgentLauncher, onUpdate func(*WorkflowStatus)) *Engine {
	return &Engine{
		launcher: launcher,
		onUpdate: onUpdate,
	}
}

// Run executes the workflow, respecting dependencies.
// Steps with no dependencies run in parallel. Steps wait for their dependencies.
func (e *Engine) Run(ctx context.Context, w *Workflow) (*WorkflowStatus, error) {
	e.mu.Lock()
	e.status = &WorkflowStatus{
		Name:  w.Name,
		State: "running",
		Steps: make([]StepStatus, len(w.Steps)),
	}
	stepIndex := make(map[string]int)
	for i, s := range w.Steps {
		e.status.Steps[i] = StepStatus{Name: s.Name, State: "pending"}
		stepIndex[s.Name] = i
	}
	e.mu.Unlock()
	e.notify()

	// Channel-based completion tracking
	done := make(map[string]chan struct{})
	for _, s := range w.Steps {
		done[s.Name] = make(chan struct{})
	}

	var wg sync.WaitGroup
	var firstErr error
	var errOnce sync.Once

	for i, step := range w.Steps {
		wg.Add(1)
		go func(idx int, s Step) {
			defer wg.Done()
			defer close(done[s.Name])

			// Wait for dependencies
			for _, dep := range s.DependsOn {
				select {
				case <-done[dep]:
					// Check if dependency failed
					e.mu.Lock()
					depState := e.status.Steps[stepIndex[dep]].State
					e.mu.Unlock()
					if depState == "failed" || depState == "error" {
						e.mu.Lock()
						e.status.Steps[idx].State = "skipped"
						e.status.Steps[idx].Error = fmt.Sprintf("dependency %q failed", dep)
						e.mu.Unlock()
						e.notify()
						return
					}
				case <-ctx.Done():
					return
				}
			}

			// Merge workflow-level and step-level env
			env := make(map[string]string)
			for k, v := range w.Env {
				env[k] = v
			}
			for k, v := range s.Env {
				env[k] = v
			}

			// Launch agent
			e.mu.Lock()
			e.status.Steps[idx].State = "running"
			e.mu.Unlock()
			e.notify()

			agentID, err := e.launcher.StartAgent(ctx, s.Name, s.Task, s.Template, s.Branch, s.Image, env)
			if err != nil {
				e.mu.Lock()
				e.status.Steps[idx].State = "failed"
				e.status.Steps[idx].Error = err.Error()
				e.mu.Unlock()
				e.notify()
				errOnce.Do(func() { firstErr = fmt.Errorf("step %q failed to start: %w", s.Name, err) })
				return
			}

			e.mu.Lock()
			e.status.Steps[idx].AgentID = agentID
			e.mu.Unlock()
			e.notify()

			// Wait for completion
			finalState, err := e.launcher.WaitForAgent(ctx, agentID)
			if err != nil {
				e.mu.Lock()
				e.status.Steps[idx].State = "failed"
				e.status.Steps[idx].Error = err.Error()
				e.mu.Unlock()
				e.notify()
				errOnce.Do(func() { firstErr = fmt.Errorf("step %q failed: %w", s.Name, err) })
				return
			}

			e.mu.Lock()
			if finalState == "completed" || finalState == "task_completed" {
				e.status.Steps[idx].State = "completed"
			} else {
				e.status.Steps[idx].State = "failed"
				e.status.Steps[idx].Error = "agent ended with status: " + finalState
				errOnce.Do(func() { firstErr = fmt.Errorf("step %q ended with status: %s", s.Name, finalState) })
			}
			e.mu.Unlock()
			e.notify()
		}(i, step)
	}

	wg.Wait()

	e.mu.Lock()
	if firstErr != nil {
		e.status.State = "failed"
	} else {
		e.status.State = "completed"
	}
	result := *e.status
	e.mu.Unlock()
	e.notify()

	return &result, firstErr
}

func (e *Engine) notify() {
	if e.onUpdate != nil {
		e.mu.Lock()
		s := *e.status
		e.mu.Unlock()
		e.onUpdate(&s)
	}
}

// Status returns the current workflow status.
func (e *Engine) Status() *WorkflowStatus {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.status == nil {
		return nil
	}
	s := *e.status
	return &s
}
