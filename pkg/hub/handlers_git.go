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
	"strings"
)

// GitDiffResponse is the response for the git diff API.
type GitDiffResponse struct {
	Diff    string `json:"diff"`
	AgentID string `json:"agentId"`
	Base    string `json:"base"`
	Error   string `json:"error,omitempty"`
}

// GitStatusResponse is the response for the git status API.
type GitStatusResponse struct {
	Status  string `json:"status"`
	Commits string `json:"commits"`
	Branch  string `json:"branch"`
	AgentID string `json:"agentId"`
	Error   string `json:"error,omitempty"`
}

// handleAgentGitDiff returns the git diff for an agent's worktree.
// Route: GET /api/v1/agents/{id}/git-diff[?base=main]
func (s *Server) handleAgentGitDiff(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "GET required", nil)
		return
	}

	agentID := extractIDFromPath(r.URL.Path, "/api/v1/agents/", "/git-diff")
	if agentID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid agent ID", nil)
		return
	}

	agent, err := s.store.GetAgent(r.Context(), agentID)
	if err != nil {
		NotFound(w, "Agent")
		return
	}

	// Check authorization
	if user := GetUserIdentityFromContext(r.Context()); user != nil {
		decision := s.authzService.CheckAccess(r.Context(), user, agentResource(agent), ActionRead)
		if !decision.Allowed {
			writeError(w, http.StatusForbidden, ErrCodeForbidden, "Access denied", nil)
			return
		}
	}

	base := r.URL.Query().Get("base")
	if base == "" {
		base = "main"
	}

	dispatcher := s.getDispatcher()
	if dispatcher == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternalError, "Dispatcher not available", nil)
		return
	}

	diff, err := dispatcher.DispatchAgentGitDiff(r.Context(), agent, base)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GitDiffResponse{
			AgentID: agentID,
			Base:    base,
			Error:   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GitDiffResponse{
		Diff:    diff,
		AgentID: agentID,
		Base:    base,
	})
}

// handleAgentGitStatus returns the git status for an agent's worktree.
// Route: GET /api/v1/agents/{id}/git-status
func (s *Server) handleAgentGitStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "GET required", nil)
		return
	}

	agentID := extractIDFromPath(r.URL.Path, "/api/v1/agents/", "/git-status")
	if agentID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid agent ID", nil)
		return
	}

	agent, err := s.store.GetAgent(r.Context(), agentID)
	if err != nil {
		NotFound(w, "Agent")
		return
	}

	// Check authorization
	if user := GetUserIdentityFromContext(r.Context()); user != nil {
		decision := s.authzService.CheckAccess(r.Context(), user, agentResource(agent), ActionRead)
		if !decision.Allowed {
			writeError(w, http.StatusForbidden, ErrCodeForbidden, "Access denied", nil)
			return
		}
	}

	dispatcher := s.getDispatcher()
	if dispatcher == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternalError, "Dispatcher not available", nil)
		return
	}

	result, err := dispatcher.DispatchAgentGitStatus(r.Context(), agent)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(GitStatusResponse{
			AgentID: agentID,
			Error:   err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(GitStatusResponse{
		Status:  result["status"],
		Commits: result["commits"],
		Branch:  result["branch"],
		AgentID: agentID,
	})
}

// extractIDFromPath extracts an ID from URL paths like /prefix/{id}/suffix.
func extractIDFromPath(path, prefix, suffix string) string {
	if !strings.HasPrefix(path, prefix) || !strings.HasSuffix(path, suffix) {
		return ""
	}
	path = strings.TrimPrefix(path, prefix)
	path = strings.TrimSuffix(path, suffix)
	return path
}
