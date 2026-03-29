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
	"context"
	"encoding/json"
	"net/http"

	"github.com/GoogleCloudPlatform/scion/pkg/store"
)

// CostSummary represents aggregated cost data for a grove or globally.
type CostSummary struct {
	TotalCostUSD      float64          `json:"totalCostUsd"`
	TotalInputTokens  int64            `json:"totalInputTokens"`
	TotalOutputTokens int64            `json:"totalOutputTokens"`
	TotalTokens       int64            `json:"totalTokens"`
	AgentCount        int              `json:"agentCount"`
	ByModel           []ModelCostBreak `json:"byModel,omitempty"`
	ByAgent           []AgentCostBreak `json:"byAgent,omitempty"`
}

// ModelCostBreak shows cost breakdown by model.
type ModelCostBreak struct {
	Model        string  `json:"model"`
	CostUSD      float64 `json:"costUsd"`
	InputTokens  int64   `json:"inputTokens"`
	OutputTokens int64   `json:"outputTokens"`
	AgentCount   int     `json:"agentCount"`
}

// AgentCostBreak shows cost breakdown by agent.
type AgentCostBreak struct {
	AgentID      string  `json:"agentId"`
	AgentName    string  `json:"agentName"`
	Model        string  `json:"model"`
	CostUSD      float64 `json:"costUsd"`
	InputTokens  int64   `json:"inputTokens"`
	OutputTokens int64   `json:"outputTokens"`
	Turns        int     `json:"turns"`
}

// handleCostSummary returns aggregated cost data for a grove.
//
// Route: GET /api/v1/groves/{groveId}/cost-summary
//
// Returns a CostSummary with breakdowns by model and agent.
func (s *Server) handleCostSummary(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "GET required", nil)
		return
	}

	groveID := extractGroveIDFromPath(r.URL.Path, "/api/v1/groves/", "/cost-summary")
	if groveID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "Invalid grove ID", nil)
		return
	}

	ctx := r.Context()
	summary, err := s.computeCostSummary(ctx, groveID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, "Failed to compute cost summary", nil)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

// computeCostSummary aggregates cost data from all agents in a grove.
func (s *Server) computeCostSummary(ctx context.Context, groveID string) (*CostSummary, error) {
	result, err := s.store.ListAgents(ctx, store.AgentFilter{
		GroveID:        groveID,
		IncludeDeleted: false,
	}, store.ListOptions{Limit: 200})
	if err != nil {
		return nil, err
	}

	summary := &CostSummary{}
	modelMap := make(map[string]*ModelCostBreak)

	for _, agent := range result.Items {
		summary.TotalCostUSD += agent.CostUSD
		summary.TotalInputTokens += agent.InputTokens
		summary.TotalOutputTokens += agent.OutputTokens
		summary.TotalTokens += agent.TotalTokens
		summary.AgentCount++

		// Per-agent breakdown
		summary.ByAgent = append(summary.ByAgent, AgentCostBreak{
			AgentID:      agent.ID,
			AgentName:    agent.Name,
			Model:        agent.ModelName,
			CostUSD:      agent.CostUSD,
			InputTokens:  agent.InputTokens,
			OutputTokens: agent.OutputTokens,
			Turns:        agent.CurrentTurns,
		})

		// Per-model aggregation
		model := agent.ModelName
		if model == "" {
			model = "unknown"
		}
		mb, ok := modelMap[model]
		if !ok {
			mb = &ModelCostBreak{Model: model}
			modelMap[model] = mb
		}
		mb.CostUSD += agent.CostUSD
		mb.InputTokens += agent.InputTokens
		mb.OutputTokens += agent.OutputTokens
		mb.AgentCount++
	}

	for _, mb := range modelMap {
		summary.ByModel = append(summary.ByModel, *mb)
	}

	return summary, nil
}

// extractGroveIDFromPath extracts a grove ID from URL paths like
// /api/v1/groves/{groveId}/cost-summary.
func extractGroveIDFromPath(path, prefix, suffix string) string {
	if len(path) <= len(prefix)+len(suffix) {
		return ""
	}
	path = path[len(prefix):]
	if idx := len(path) - len(suffix); idx > 0 && path[idx:] == suffix {
		return path[:idx]
	}
	return ""
}
