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

package hub

import (
	"encoding/json"
	"net/http"

	"github.com/GoogleCloudPlatform/scion/pkg/api"
	"github.com/GoogleCloudPlatform/scion/pkg/store"
)

// handleTasks routes requests under /api/v1/groves/{groveId}/tasks[/{taskId}].
//
// Routes:
//
//	POST   /api/v1/groves/{groveId}/tasks           — Create task
//	GET    /api/v1/groves/{groveId}/tasks           — List tasks
//	GET    /api/v1/groves/{groveId}/tasks/{taskId}  — Get task
//	PATCH  /api/v1/groves/{groveId}/tasks/{taskId}  — Update task
//	DELETE /api/v1/groves/{groveId}/tasks/{taskId}  — Delete task
func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request, groveID, taskPath string) {
	// Require authentication
	identity := GetIdentityFromContext(r.Context())
	if identity == nil {
		Unauthorized(w)
		return
	}

	// For agent identities, enforce grove isolation
	if agentIdentity := GetAgentIdentityFromContext(r.Context()); agentIdentity != nil {
		if agentIdentity.GroveID() != groveID {
			Forbidden(w)
			return
		}
	}

	if taskPath == "" {
		// Collection endpoint
		switch r.Method {
		case http.MethodGet:
			s.handleListTasks(w, r, groveID)
		case http.MethodPost:
			s.handleCreateTask(w, r, groveID)
		default:
			MethodNotAllowed(w)
		}
		return
	}

	// Individual task endpoint
	taskID := taskPath
	switch r.Method {
	case http.MethodGet:
		s.handleGetTask(w, r, groveID, taskID)
	case http.MethodPatch:
		s.handleUpdateTask(w, r, groveID, taskID)
	case http.MethodDelete:
		s.handleDeleteTask(w, r, groveID, taskID)
	default:
		MethodNotAllowed(w)
	}
}

// handleCreateTask handles POST /api/v1/groves/{groveId}/tasks
func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request, groveID string) {
	var req struct {
		Title      string            `json:"title"`
		WorkflowID string            `json:"workflowId,omitempty"`
		AssignedTo string            `json:"assignedTo,omitempty"`
		Branch     string            `json:"branch,omitempty"`
		DependsOn  []string          `json:"dependsOn,omitempty"`
		Input      map[string]string `json:"input,omitempty"`
	}
	if err := readJSON(r, &req); err != nil {
		BadRequest(w, "Invalid request body: "+err.Error())
		return
	}

	if req.Title == "" {
		ValidationError(w, "title is required", nil)
		return
	}

	// Determine creator identity
	createdBy := ""
	if identity := GetIdentityFromContext(r.Context()); identity != nil {
		createdBy = identity.ID()
	}

	task := &store.Task{
		ID:         api.NewUUID(),
		GroveID:    groveID,
		WorkflowID: req.WorkflowID,
		Title:      req.Title,
		Status:     "pending",
		CreatedBy:  createdBy,
		AssignedTo: req.AssignedTo,
		Branch:     req.Branch,
		DependsOn:  req.DependsOn,
		Input:      req.Input,
	}

	if err := s.store.CreateTask(r.Context(), task); err != nil {
		writeErrorFromErr(w, err, "")
		return
	}

	writeJSON(w, http.StatusCreated, task)
}

// handleListTasks handles GET /api/v1/groves/{groveId}/tasks
func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request, groveID string) {
	query := r.URL.Query()

	filter := store.TaskFilter{
		WorkflowID: query.Get("workflowId"),
		Status:     query.Get("status"),
		AssignedTo: query.Get("assignedTo"),
	}

	tasks, err := s.store.ListTasks(r.Context(), groveID, filter)
	if err != nil {
		writeErrorFromErr(w, err, "")
		return
	}

	if tasks == nil {
		tasks = []store.Task{}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"tasks": tasks,
		"count": len(tasks),
	})
}

// handleGetTask handles GET /api/v1/groves/{groveId}/tasks/{taskId}
func (s *Server) handleGetTask(w http.ResponseWriter, r *http.Request, groveID, taskID string) {
	task, err := s.store.GetTask(r.Context(), taskID)
	if err != nil {
		writeErrorFromErr(w, err, "")
		return
	}

	// Verify the task belongs to the requested grove
	if task.GroveID != groveID {
		NotFound(w, "Task")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// handleUpdateTask handles PATCH /api/v1/groves/{groveId}/tasks/{taskId}
func (s *Server) handleUpdateTask(w http.ResponseWriter, r *http.Request, groveID, taskID string) {
	task, err := s.store.GetTask(r.Context(), taskID)
	if err != nil {
		writeErrorFromErr(w, err, "")
		return
	}

	// Verify the task belongs to the requested grove
	if task.GroveID != groveID {
		NotFound(w, "Task")
		return
	}

	var req struct {
		Title      *string           `json:"title,omitempty"`
		Status     *string           `json:"status,omitempty"`
		AssignedTo *string           `json:"assignedTo,omitempty"`
		AgentID    *string           `json:"agentId,omitempty"`
		Branch     *string           `json:"branch,omitempty"`
		DependsOn  []string          `json:"dependsOn,omitempty"`
		Input      map[string]string `json:"input,omitempty"`
		Output     map[string]string `json:"output,omitempty"`
		Summary    *string           `json:"summary,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		BadRequest(w, "Invalid request body: "+err.Error())
		return
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Status != nil {
		task.Status = *req.Status
	}
	if req.AssignedTo != nil {
		task.AssignedTo = *req.AssignedTo
	}
	if req.AgentID != nil {
		task.AgentID = *req.AgentID
	}
	if req.Branch != nil {
		task.Branch = *req.Branch
	}
	if req.DependsOn != nil {
		task.DependsOn = req.DependsOn
	}
	if req.Input != nil {
		task.Input = req.Input
	}
	if req.Output != nil {
		task.Output = req.Output
	}
	if req.Summary != nil {
		task.Summary = *req.Summary
	}

	if err := s.store.UpdateTask(r.Context(), task); err != nil {
		writeErrorFromErr(w, err, "")
		return
	}

	writeJSON(w, http.StatusOK, task)
}

// handleDeleteTask handles DELETE /api/v1/groves/{groveId}/tasks/{taskId}
func (s *Server) handleDeleteTask(w http.ResponseWriter, r *http.Request, groveID, taskID string) {
	// Verify task exists and belongs to the grove
	task, err := s.store.GetTask(r.Context(), taskID)
	if err != nil {
		writeErrorFromErr(w, err, "")
		return
	}
	if task.GroveID != groveID {
		NotFound(w, "Task")
		return
	}

	if err := s.store.DeleteTask(r.Context(), taskID); err != nil {
		writeErrorFromErr(w, err, "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
