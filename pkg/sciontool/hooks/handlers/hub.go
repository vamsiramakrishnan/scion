/*
Copyright 2025 The Scion Authors.
*/

package handlers

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	state "github.com/GoogleCloudPlatform/scion/pkg/agent/state"
	"github.com/GoogleCloudPlatform/scion/pkg/sciontool/hooks"
	"github.com/GoogleCloudPlatform/scion/pkg/sciontool/hub"
	"github.com/GoogleCloudPlatform/scion/pkg/sciontool/log"
)

// HubHandler sends status updates to the Scion Hub.
type HubHandler struct {
	client *hub.Client
}

// NewHubHandler creates a new hub handler.
// Returns nil if the Hub client is not configured.
func NewHubHandler() *HubHandler {
	client := hub.NewClient()
	if client == nil || !client.IsConfigured() {
		return nil
	}
	return &HubHandler{
		client: client,
	}
}

// Handle processes an event and sends a status update to the Hub.
// It mirrors the sticky activity logic from StatusHandler: when the local activity
// is waiting_for_input or completed, non-new-work events won't overwrite it.
func (h *HubHandler) Handle(event *hooks.Event) error {
	if h == nil || h.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var err error
	switch event.Name {
	case hooks.EventSessionStart:
		// Session starting - report running phase with idle activity (clears any sticky)
		log.Debug("Hub: Reporting running/idle (session start)")
		err = h.client.ReportState(ctx, state.PhaseRunning, state.ActivityIdle, "Session started")

	case hooks.EventPromptSubmit, hooks.EventAgentStart:
		// New work events - always clear sticky status
		message := "Processing"
		if event.Data.Prompt != "" {
			message = truncateMessage(event.Data.Prompt, 100)
		}
		log.Debug("Hub: Reporting thinking")
		as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityThinking}
		err = h.client.UpdateStatus(ctx, hub.StatusUpdate{
			Activity: state.ActivityThinking,
			Status:   as.DisplayStatus(),
			Message:  message,
		})

	case hooks.EventModelStart:
		// Model start - report thinking, but respect sticky activity
		if h.isLocalActivitySticky() {
			log.Debug("Hub: Skipping thinking (local activity is sticky)")
			return nil
		}
		message := "Processing"
		if event.Data.Prompt != "" {
			message = truncateMessage(event.Data.Prompt, 100)
		}
		log.Debug("Hub: Reporting thinking (model-start)")
		as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityThinking}
		err = h.client.UpdateStatus(ctx, hub.StatusUpdate{
			Activity: state.ActivityThinking,
			Status:   as.DisplayStatus(),
			Message:  message,
		})

	case hooks.EventToolStart:
		// Claude-specific: ExitPlanMode and AskUserQuestion mean waiting for user
		if event.Dialect == "claude" && (event.Data.ToolName == "ExitPlanMode" || event.Data.ToolName == "AskUserQuestion") {
			message := "Waiting for input"
			if event.Data.ToolName == "ExitPlanMode" {
				message = "Waiting for plan approval"
			}
			log.Debug("Hub: Reporting waiting_for_input (waiting: %s)", event.Data.ToolName)
			as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityWaitingForInput}
			err = h.client.UpdateStatus(ctx, hub.StatusUpdate{
				Activity: state.ActivityWaitingForInput,
				Status:   as.DisplayStatus(),
				Message:  message,
			})
			break
		}

		// Tool-start clears waiting_for_input (user has responded) but
		// preserves completed (tools may fire after task_completed as wrap-up).
		localActivity := readLocalActivity()
		if localActivity == string(state.ActivityCompleted) || localActivity == string(state.ActivityLimitsExceeded) {
			log.Debug("Hub: Skipping executing (completed is sticky, post-completion tool)")
			return nil
		}

		// Agent is executing a tool
		message := "Executing tool"
		if event.Data.ToolName != "" {
			message = "Executing: " + event.Data.ToolName
		}
		log.Debug("Hub: Reporting executing (tool: %s)", event.Data.ToolName)
		as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityExecuting}
		err = h.client.UpdateStatus(ctx, hub.StatusUpdate{
			Activity: state.ActivityExecuting,
			ToolName: event.Data.ToolName,
			Status:   as.DisplayStatus(),
			Message:  message,
		})

	case hooks.EventModelEnd:
		// Model-end carries token usage — always report cost deltas, even if
		// we skip the activity update due to a sticky local activity.
		su := hub.StatusUpdate{}
		applyCostFields(&su, event)

		if !h.isLocalActivitySticky() {
			as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityIdle}
			su.Activity = state.ActivityIdle
			su.Status = as.DisplayStatus()
			su.Message = "Ready"
			log.Debug("Hub: Reporting idle + cost (model-end, input=%d output=%d)",
				event.Data.InputTokens, event.Data.OutputTokens)
		} else {
			log.Debug("Hub: Reporting cost only (model-end, local activity is sticky, input=%d output=%d)",
				event.Data.InputTokens, event.Data.OutputTokens)
		}

		// Only send the update if there is something to report
		if su.Activity != "" || su.InputTokensDelta != nil || su.OutputTokensDelta != nil {
			err = h.client.UpdateStatus(ctx, su)
		}

	case hooks.EventToolEnd, hooks.EventAgentEnd:
		// Check if local activity is sticky before sending idle
		if h.isLocalActivitySticky() {
			log.Debug("Hub: Skipping idle (local activity is sticky)")
			return nil
		}
		log.Debug("Hub: Reporting idle (step completed)")
		as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityIdle}
		err = h.client.UpdateStatus(ctx, hub.StatusUpdate{
			Activity: state.ActivityIdle,
			Status:   as.DisplayStatus(),
			Message:  "Ready",
		})

	case hooks.EventNotification:
		// Agent is waiting for input
		message := "Waiting for input"
		if event.Data.Message != "" {
			message = truncateMessage(event.Data.Message, 100)
		}
		log.Debug("Hub: Reporting waiting_for_input (notification)")
		as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityWaitingForInput}
		err = h.client.UpdateStatus(ctx, hub.StatusUpdate{
			Activity: state.ActivityWaitingForInput,
			Status:   as.DisplayStatus(),
			Message:  message,
		})

	case hooks.EventResponseComplete:
		summary := "Task completed"
		if event.Data.Message != "" {
			summary = truncateMessage(event.Data.Message, 200)
		}
		log.Debug("Hub: Reporting task completed (response-complete)")
		as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityCompleted}
		err = h.client.UpdateStatus(ctx, hub.StatusUpdate{
			Activity:    state.ActivityCompleted,
			Status:      as.DisplayStatus(),
			TaskSummary: summary,
		})

	case hooks.EventSessionEnd:
		// Session ended
		log.Debug("Hub: Reporting stopped (session end)")
		as := state.AgentState{Phase: state.PhaseStopped}
		err = h.client.UpdateStatus(ctx, hub.StatusUpdate{
			Phase:   state.PhaseStopped,
			Status:  as.DisplayStatus(),
			Message: "Session ended",
		})

	default:
		// No status update for this event
		return nil
	}

	if err != nil {
		log.Error("Hub status update failed: %v", err)
		// Don't return error - we don't want Hub failures to break the hook chain
	} else {
		log.Debug("Hub status update sent successfully")
	}

	return nil
}

// isLocalActivitySticky reads the local agent-info.json (written by StatusHandler
// which runs before HubHandler) and returns true if the activity is sticky.
func (h *HubHandler) isLocalActivitySticky() bool {
	activity := readLocalActivity()
	return isStickyActivity(activity)
}

// readLocalActivity reads the current activity from the local agent-info.json file.
func readLocalActivity() string {
	home := os.Getenv("HOME")
	if home == "" {
		home = "/home/scion"
	}
	statusPath := filepath.Join(home, "agent-info.json")

	data, err := os.ReadFile(statusPath)
	if err != nil {
		return ""
	}

	var info map[string]interface{}
	if err := json.Unmarshal(data, &info); err != nil {
		return ""
	}

	activity, _ := info["activity"].(string)
	return activity
}

// ReportWaitingForInput sends a waiting-for-input status to the Hub.
func (h *HubHandler) ReportWaitingForInput(message string) error {
	if h == nil || h.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Debug("Hub: Reporting waiting_for_input (ask_user: %s)", truncateMessage(message, 50))
	as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityWaitingForInput}
	return h.client.UpdateStatus(ctx, hub.StatusUpdate{
		Activity: state.ActivityWaitingForInput,
		Status:   as.DisplayStatus(),
		Message:  message,
	})
}

// ReportTaskCompleted sends a task-completed status to the Hub.
func (h *HubHandler) ReportTaskCompleted(taskSummary string) error {
	if h == nil || h.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Debug("Hub: Reporting task completed: %s", truncateMessage(taskSummary, 50))
	as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityCompleted}
	return h.client.UpdateStatus(ctx, hub.StatusUpdate{
		Activity:    state.ActivityCompleted,
		Status:      as.DisplayStatus(),
		TaskSummary: taskSummary,
	})
}

// ReportLimitsExceeded sends a limits-exceeded status to the Hub.
func (h *HubHandler) ReportLimitsExceeded(message string) error {
	if h == nil || h.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Debug("Hub: Reporting limits_exceeded: %s", truncateMessage(message, 50))
	as := state.AgentState{Phase: state.PhaseRunning, Activity: state.ActivityLimitsExceeded}
	return h.client.UpdateStatus(ctx, hub.StatusUpdate{
		Activity: state.ActivityLimitsExceeded,
		Status:   as.DisplayStatus(),
		Message:  message,
	})
}

// ReportCounts sends current turn and model call counts to the Hub.
func (h *HubHandler) ReportCounts(turnCount, modelCallCount int) error {
	if h == nil || h.client == nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	log.Debug("Hub: Reporting counts (turns=%d, model_calls=%d)", turnCount, modelCallCount)
	return h.client.UpdateStatus(ctx, hub.StatusUpdate{
		CurrentTurns:      &turnCount,
		CurrentModelCalls: &modelCallCount,
	})
}

// truncateMessage truncates a message to the specified length.
func truncateMessage(msg string, maxLen int) string {
	if len(msg) <= maxLen {
		return msg
	}
	return msg[:maxLen-3] + "..."
}

// applyCostFields populates cost-tracking delta fields on a StatusUpdate
// from the token usage data in a model-end event. The model name is read
// from the SCION_MODEL environment variable.
func applyCostFields(su *hub.StatusUpdate, event *hooks.Event) {
	if event.Data.InputTokens <= 0 && event.Data.OutputTokens <= 0 {
		return
	}

	model := os.Getenv("SCION_MODEL")

	if event.Data.InputTokens > 0 {
		v := event.Data.InputTokens
		su.InputTokensDelta = &v
	}
	if event.Data.OutputTokens > 0 {
		v := event.Data.OutputTokens
		su.OutputTokensDelta = &v
	}

	costUSD := estimateCostUSD(model, event.Data.InputTokens, event.Data.OutputTokens)
	if costUSD > 0 {
		su.CostUSDDelta = &costUSD
	}

	if model != "" {
		su.ModelName = model
	}
}

// estimateCostUSD returns an estimated cost in USD for the given token counts
// based on standard model pricing. Prices are per million tokens (MTok).
//
// Pricing sources (approximate, as of 2025):
//
//	Claude Sonnet:    $3/MTok input,   $15/MTok output
//	Claude Opus:      $15/MTok input,  $75/MTok output
//	Claude Haiku:     $0.25/MTok input, $1.25/MTok output
//	Gemini 2.5 Pro:   $1.25/MTok input, $10/MTok output
//	Gemini 2.5 Flash: $0.15/MTok input, $0.60/MTok output
//	GPT-4o:           $2.50/MTok input, $10/MTok output
//	GPT-4o-mini:      $0.15/MTok input, $0.60/MTok output
func estimateCostUSD(model string, inputTokens, outputTokens int64) float64 {
	// Default pricing (Claude Sonnet-class)
	inputPricePerMTok := 3.0
	outputPricePerMTok := 15.0

	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "opus"):
		inputPricePerMTok = 15.0
		outputPricePerMTok = 75.0
	case strings.Contains(m, "sonnet"):
		inputPricePerMTok = 3.0
		outputPricePerMTok = 15.0
	case strings.Contains(m, "haiku"):
		inputPricePerMTok = 0.25
		outputPricePerMTok = 1.25
	case strings.Contains(m, "gemini") && strings.Contains(m, "pro"):
		inputPricePerMTok = 1.25
		outputPricePerMTok = 10.0
	case strings.Contains(m, "gemini") && strings.Contains(m, "flash"):
		inputPricePerMTok = 0.15
		outputPricePerMTok = 0.60
	case strings.Contains(m, "gpt-4o-mini"):
		inputPricePerMTok = 0.15
		outputPricePerMTok = 0.60
	case strings.Contains(m, "gpt-4o"):
		inputPricePerMTok = 2.50
		outputPricePerMTok = 10.0
	case strings.Contains(m, "gpt-4"):
		inputPricePerMTok = 2.50
		outputPricePerMTok = 10.0
	}

	cost := (float64(inputTokens) * inputPricePerMTok / 1_000_000) +
		(float64(outputTokens) * outputPricePerMTok / 1_000_000)
	return cost
}
