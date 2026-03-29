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
	"fmt"
	"log/slog"
	"net/http"
	"time"
)

// ActivityFeedItem is a unified activity event for the live feed.
// It combines agent status events, creation/deletion events, and broker
// events into a single stream for the "mission control" view.
type ActivityFeedItem struct {
	ID        string          `json:"id"`
	Timestamp string          `json:"timestamp"`
	Type      string          `json:"type"`              // "agent_status", "agent_created", "agent_deleted", "broker_status"
	AgentID   string          `json:"agentId,omitempty"` // Agent ID (if agent event)
	AgentName string          `json:"agentName,omitempty"`
	GroveID   string          `json:"groveId,omitempty"`
	Phase     string          `json:"phase,omitempty"`
	Activity  string          `json:"activity,omitempty"`
	ToolName  string          `json:"toolName,omitempty"`
	Message   string          `json:"message,omitempty"`
	Subject   string          `json:"subject"` // Raw event subject
	Data      json.RawMessage `json:"data"`    // Full event payload
}

// handleActivityFeed serves a Server-Sent Events stream of all agent activity
// in a grove. This is the backend for the "mission control" live activity feed.
//
// Route: GET /api/v1/groves/{groveId}/activity-feed
//
// Query parameters:
//   - grove_id (required): The grove to subscribe to
//
// The stream sends events in SSE format with the following event types:
//   - activity: A new activity feed item
//   - heartbeat: Periodic keepalive (every 15 seconds)
func (s *Server) handleActivityFeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, ErrCodeMethodNotAllowed, "GET required", nil)
		return
	}

	groveID := r.URL.Query().Get("grove_id")
	if groveID == "" {
		writeError(w, http.StatusBadRequest, ErrCodeInvalidRequest, "grove_id parameter required", nil)
		return
	}

	// Verify grove exists
	if _, err := s.store.GetGrove(r.Context(), groveID); err != nil {
		NotFound(w, "Grove")
		return
	}

	eventPub, ok := s.events.(*ChannelEventPublisher)
	if !ok || eventPub == nil {
		writeError(w, http.StatusServiceUnavailable, ErrCodeInternalError, "Event system not available", nil)
		return
	}

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, ErrCodeInternalError, "Streaming not supported", nil)
		return
	}

	// Subscribe to all events in this grove
	ch, unsub := eventPub.Subscribe(
		fmt.Sprintf("grove.%s.>", groveID),
	)
	defer unsub()

	ctx := r.Context()
	heartbeat := time.NewTicker(15 * time.Second)
	defer heartbeat.Stop()

	slog.Debug("Activity feed started", "grove_id", groveID)

	for {
		select {
		case <-ctx.Done():
			return

		case evt, ok := <-ch:
			if !ok {
				return
			}

			item := ActivityFeedItem{
				ID:        fmt.Sprintf("af_%d", time.Now().UnixNano()),
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Subject:   evt.Subject,
				Data:      json.RawMessage(evt.Data),
			}

			// Enrich based on event type
			enrichActivityItem(&item, evt)

			data, err := json.Marshal(item)
			if err != nil {
				continue
			}

			fmt.Fprintf(w, "event: activity\ndata: %s\n\n", data)
			flusher.Flush()

		case <-heartbeat.C:
			fmt.Fprintf(w, "event: heartbeat\ndata: {\"time\":\"%s\"}\n\n",
				time.Now().UTC().Format(time.RFC3339))
			flusher.Flush()
		}
	}
}

// enrichActivityItem extracts structured fields from the raw event data.
func enrichActivityItem(item *ActivityFeedItem, evt Event) {
	// Try to parse as agent status event
	var status AgentStatusEvent
	if err := json.Unmarshal(evt.Data, &status); err == nil && status.AgentID != "" {
		item.Type = "agent_status"
		item.AgentID = status.AgentID
		item.GroveID = status.GroveID
		item.Phase = status.Phase
		item.Activity = status.Activity
		if status.Detail != nil {
			item.ToolName = status.Detail.ToolName
			item.Message = status.Detail.Message
		}
		return
	}

	// Try as agent created event
	var created AgentCreatedEvent
	if err := json.Unmarshal(evt.Data, &created); err == nil && created.AgentID != "" && created.Name != "" {
		item.Type = "agent_created"
		item.AgentID = created.AgentID
		item.AgentName = created.Name
		item.GroveID = created.GroveID
		item.Phase = created.Phase
		return
	}

	// Try as agent deleted event
	var deleted AgentDeletedEvent
	if err := json.Unmarshal(evt.Data, &deleted); err == nil && deleted.AgentID != "" {
		item.Type = "agent_deleted"
		item.AgentID = deleted.AgentID
		item.GroveID = deleted.GroveID
		return
	}

	// Try as broker event
	var broker BrokerGroveEvent
	if err := json.Unmarshal(evt.Data, &broker); err == nil && broker.BrokerID != "" {
		item.Type = "broker_status"
		item.GroveID = broker.GroveID
		item.Message = fmt.Sprintf("Broker %s is %s", broker.BrokerName, broker.Status)
		return
	}

	// Unknown event type
	item.Type = "unknown"
}
