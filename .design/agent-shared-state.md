# Agent-to-Agent Shared State & Task Orchestration

**Status:** IMPLEMENTING
**Created:** 2026-03-29

## Overview

This document describes the shared state and task orchestration system that enables
Scion agents to collaborate. It builds on the existing infrastructure:

- Agent JWT scopes (`grove:agent:create`, `grove:agent:lifecycle`) — **implemented**
- Hub handler auth for agent callers — **implemented**
- Message broker with pub/sub — **implemented**
- Structured messages (`pkg/messages/types.go`) — **implemented**
- Event publisher with NATS-style subscriptions — **implemented**
- Workflow engine with DAG execution — **implemented**

What's missing: **the coordination layer** — structured tasks, output handoff,
shared state between agents, and human approval gates.

---

## Design Principles

1. **Git as coordination substrate.** Agents already share a repository. Task definitions
   and artifacts should be committable files, not database-only entities. This makes
   workflows auditable, diffable, and reproducible.

2. **Hub as real-time layer.** While git handles durable artifacts, the Hub provides
   real-time signaling (status events, completion notifications, message delivery).

3. **Temporal-inspired task model.** Tasks have lifecycle states, inputs, outputs,
   and parent-child relationships. Like Temporal activities, tasks are the unit of work.

4. **LangGraph-inspired state passing.** Workflow state flows between steps as a
   typed dictionary. Each step can read previous state and append to it.

5. **GitHub Actions-inspired artifact passing.** Step outputs become available to
   dependent steps via named artifact references.

---

## 1. Task Artifact System

### 1.1 Task Definition (YAML)

Tasks are the unit of coordination between agents. Stored in `.scion/tasks/` in
the repository or as Hub entities in hosted mode.

```yaml
# .scion/tasks/task-001.yaml
id: task-001
workflow: code-review-pipeline
title: "Review authentication middleware for security issues"
created_by: planner-agent
assigned_to: security-reviewer
status: pending  # pending | assigned | running | completed | failed | blocked | approved | rejected
branch: feature/auth-middleware
depends_on: []
blocks: [task-003]

# Structured input — available to the assigned agent as context
input:
  files_to_review:
    - pkg/hub/auth.go
    - pkg/hub/agenttoken.go
  focus_areas:
    - JWT validation
    - Token refresh security
  parent_findings: ""  # Will be populated by preceding task

# Structured output — written by the agent on completion
output: {}

# Acceptance criteria — used by review/approval gates
acceptance_criteria:
  - "No critical vulnerabilities found"
  - "All findings documented with severity ratings"

# Metadata
created_at: "2026-03-29T10:00:00Z"
updated_at: "2026-03-29T10:00:00Z"
```

### 1.2 Task Store API

New Hub endpoints for task CRUD:

```
POST   /api/v1/groves/{groveId}/tasks          — Create task
GET    /api/v1/groves/{groveId}/tasks           — List tasks (filter by workflow, status, assignee)
GET    /api/v1/groves/{groveId}/tasks/{taskId}  — Get task
PATCH  /api/v1/groves/{groveId}/tasks/{taskId}  — Update task (status, output)
DELETE /api/v1/groves/{groveId}/tasks/{taskId}  — Delete task
```

Authorization: User callers need grove write access. Agent callers need
`grove:agent:create` scope and grove isolation is enforced.

### 1.3 Task Events

When a task status changes, the Hub publishes:
```
grove.{groveId}.task.{taskId}.status  →  {taskId, status, assignedTo, output}
```

Agents and workflows can subscribe to these events for completion signaling.

---

## 2. Workflow State (Output Handoff)

### 2.1 Workflow State Model

Each workflow execution has a shared state dictionary. Steps can read state from
completed dependencies and write their own output to it.

```go
// WorkflowState is the shared state for a workflow execution.
type WorkflowState struct {
    WorkflowID string                 `json:"workflow_id"`
    Outputs    map[string]StepOutput  `json:"outputs"`  // keyed by step name
}

// StepOutput is the output of a completed workflow step.
type StepOutput struct {
    StepName  string            `json:"step_name"`
    AgentID   string            `json:"agent_id"`
    Status    string            `json:"status"`
    Branch    string            `json:"branch,omitempty"`
    Artifacts map[string]string `json:"artifacts,omitempty"`  // name → value/path
    Summary   string            `json:"summary,omitempty"`
    Metadata  map[string]string `json:"metadata,omitempty"`
}
```

### 2.2 State Flow in Workflows

The workflow YAML gets new fields:

```yaml
name: review-pipeline
steps:
  - name: implement
    template: fullstack-dev
    task: "Implement user auth middleware"
    outputs:
      - branch    # auto-captured: the git branch this agent worked on
      - summary   # auto-captured: agent's task summary at completion

  - name: review
    template: code-reviewer
    task: |
      Review the auth middleware implementation.
      Branch: ${steps.implement.branch}
      Implementation summary: ${steps.implement.summary}
    depends_on: [implement]
    outputs:
      - review_report

  - name: security
    template: security-reviewer
    task: |
      Security audit of auth middleware.
      Branch: ${steps.implement.branch}
    depends_on: [implement]

  - name: fix
    template: fullstack-dev
    task: |
      Address review findings:
      ${steps.review.review_report}
      Security findings:
      ${steps.security.summary}
    depends_on: [review, security]
    gate: approval  # PAUSE here for human approval before starting
```

### 2.3 Variable Interpolation

The workflow engine resolves `${steps.<name>.<output>}` references before
launching each step. Available variables:

- `${steps.<name>.branch}` — Git branch from completed step
- `${steps.<name>.summary}` — Agent's task summary at completion
- `${steps.<name>.status}` — Final status (completed, failed)
- `${steps.<name>.agent_id}` — Agent ID for cross-referencing
- `${steps.<name>.artifacts.<key>}` — Named artifacts from step output

---

## 3. Completion Signaling

### 3.1 Event-Driven Completion

When an agent reaches a terminal state (completed, failed, stopped), the Hub
publishes an event. The workflow engine subscribes to this instead of polling.

Flow:
1. Workflow engine creates agent via `launcher.StartAgent()`
2. Engine subscribes to `agent.{agentID}.status` events
3. When event with terminal phase/activity arrives, engine proceeds
4. Fall back to polling if SSE connection fails

### 3.2 Agent Status Capture

When an agent completes, the engine captures its state as `StepOutput`:

```go
func captureStepOutput(agent *store.Agent) StepOutput {
    return StepOutput{
        AgentID:   agent.ID,
        Status:    agent.Activity,
        Branch:    agent.AppliedConfig.Branch,
        Summary:   agent.TaskSummary,
        Metadata: map[string]string{
            "model":        agent.ModelName,
            "cost_usd":     fmt.Sprintf("%.4f", agent.CostUSD),
            "input_tokens": fmt.Sprintf("%d", agent.InputTokens),
        },
    }
}
```

---

## 4. Human-in-the-Loop Approval Gates

### 4.1 Gate Mechanism

Workflow steps can declare `gate: approval`. When the engine reaches a gated
step, it:

1. Pauses the workflow
2. Updates workflow status to `awaiting_approval`
3. Publishes a notification to the workflow creator
4. Waits for `POST /api/v1/workflows/{id}/approve` or `/reject`

### 4.2 Gate Types

```yaml
gate: approval           # Human must approve before step runs
gate: review             # Human reviews output of previous step
gate: auto               # No gate (default)
```

### 4.3 Approval API

```
POST /api/v1/workflows/{workflowId}/gates/{stepName}/approve
POST /api/v1/workflows/{workflowId}/gates/{stepName}/reject
GET  /api/v1/workflows/{workflowId}/gates                    # List pending gates
```

---

## 5. Implementation Plan

### Phase 1: Task Artifacts + Store
- Create `pkg/hub/tasks.go` — Task CRUD handlers
- Create `pkg/store/task.go` — Task model and store interface
- Create `pkg/store/sqlite/tasks.go` — SQLite implementation
- Add migration for tasks table
- Wire routes in `server.go`

### Phase 2: Workflow State + Output Handoff
- Extend `pkg/workflow/types.go` with `WorkflowState`, `StepOutput`
- Add `outputs` field to `Step` struct
- Add variable interpolation in task strings (`${steps.X.Y}`)
- Capture agent state on completion → `StepOutput`
- Pass state between steps

### Phase 3: Approval Gates
- Add `gate` field to `Step` struct
- Add gate channels in workflow engine
- Create approval API endpoints
- Wire notifications for pending gates

### Phase 4: CLI Integration
- `scion tasks list/create/update` — Task management
- `scion workflow approve/reject` — Gate management
- Agent-side: `scion` CLI reads `SCION_HUB_TOKEN` for Hub API access
