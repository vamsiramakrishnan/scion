# SCION Performance Analysis: Top 10 Improvement Areas

**Date**: 2026-03-29
**Status**: Analysis Complete

## Context

SCION is a multi-agent orchestration platform (~253K LOC Go, ~76 TypeScript files)
managing containerized AI coding agents across Docker, Podman, Kubernetes, and macOS
runtimes. This document identifies the top 10 performance improvement areas, evaluates
microkernel decomposition for faster startup, and assesses a potential Rust rewrite.

---

## Executive Summary

| Rank | Area | Impact | Effort | Estimated Benefit |
|------|------|--------|--------|-------------------|
| 1 | Web UI Blocking Pattern | **High** | Low | Perceived latency: 5-10s to <200ms |
| 2 | Binary Size & Startup Time | **High** | Medium | 40-60% binary size reduction |
| 3 | SQLite Database Layer | **High** | Medium | 3-5x read throughput under load |
| 4 | JSON/Base64 WebSocket Protocol | Medium | Medium | ~30% PTY bandwidth savings |
| 5 | Server Initialization | Medium | Medium | 30-50% faster startup |
| 6 | Memory Allocation (Hot Paths) | Medium | Low | 20-40% less GC pressure |
| 7 | Event System Backpressure | Medium | Low | Zero dropped events |
| 8 | WebSocket Buffer Sizing | Low | Low | 10-20% PTY throughput gain |
| 9 | Benchmarking Infrastructure | Low | Medium | Enables data-driven optimization |
| 10 | Dependency Tree Optimization | Low | High | 30-50% faster compile times |

**Microkernel verdict**: Yes, beneficial via build tags for deployment-specific binaries.
**Rust rewrite verdict**: No. SCION is I/O-bound; Go optimizations deliver more value.

---

## Detailed Analysis

### #1 — Web UI Blocking Pattern (High Impact, Low Effort)

**Current state**: Every mutating action (start, stop, delete agent) sets
`this.loading = true`, awaits a synchronous full-data reload from the API, then
re-enables the UI. Observed delays of 5-10 seconds during multi-agent operations.

**Root cause**: Components in `web/src/components/pages/` use a blocking pattern:
```
action() → set loading=true → await API call → await full reload → set loading=false
```

**Proposed solution** (already designed in `.design/web-frontend-performance.md`):
- **Layer 1**: Optimistic local state updates (instant visual feedback)
- **Layer 2**: Background refresh without blocking spinner
- **Layer 3**: SSE-driven real-time status rendering

**Files**: `web/src/components/pages/agents.ts`, `agent-detail.ts`, `grove-detail.ts`

---

### #2 — Binary Size & Startup Time (High Impact, Medium Effort)

**Current state**: Single fat binary includes K8s client, GCP SDKs, Rclone, OTel,
gRPC, and embedded web assets. Build does not use `-s -w` ldflags for stripping
debug symbols. All dependencies compiled in regardless of deployment mode.

**Proposed solution**:
- Add `-s -w` to LDFLAGS in Makefile (one-line change, ~20-30% size reduction)
- Use build tags to create deployment profiles:
  - `local` — excludes K8s, GCP, Rclone
  - `hub` — excludes container runtime libraries
  - `broker` — excludes full API framework
- Lazy-load heavy subsystems (scheduler, GCP integrations)

**Files**: `Makefile:26`, `cmd/server.go`

---

### #3 — SQLite Database Layer (High Impact, Medium Effort)

**Current state**: Single connection (`SetMaxOpenConns(1)`), 40 sequential migrations
at startup, no explicit indices beyond schema defaults, no prepared statements for
hot queries.

**Why single connection**: Comment says "SQLite serializes writes anyway" — true for
writes, but WAL mode allows concurrent reads. The single connection serializes reads
too, unnecessarily.

**Proposed solution**:
- **Separate read pool**: Open a second `*sql.DB` with `SetMaxOpenConns(4)` for reads
  (WAL mode allows concurrent readers alongside a single writer)
- **Batch migrations**: Group migrations into version ranges to reduce transaction count
- **Add indices**: On hot query paths — agent lookups by grove, status filters,
  broker queries by state
- **Prepared statements**: For frequently-executed queries in agent/grove listing

**Files**: `pkg/store/sqlite/sqlite.go`, migration strings in same file

---

### #4 — JSON/Base64 WebSocket Protocol (Medium Impact, Medium Effort)

**Current state**: All WebSocket messages are JSON-encoded including binary payloads.
PTY data and stream frames use Base64 encoding inside JSON, adding ~33% overhead.
`pkg/wsprotocol/protocol.go` defines the message types; `pkg/hub/controlchannel.go`
handles routing.

**Proposed solution**:
- Use binary WebSocket message type for PTY data and stream frames
- Keep JSON for control messages (connect, ping, error, etc.)
- Consider MessagePack for high-frequency structured messages
- Eliminate Base64 for binary payloads entirely

**Files**: `pkg/wsprotocol/protocol.go`, `pkg/wsprotocol/connection.go`,
`pkg/hub/controlchannel.go`, `pkg/hub/pty_handlers.go`

---

### #5 — Server Initialization (Medium Impact, Medium Effort)

**Current state**: `cmd/server.go` is ~2300 lines with a ~1040-line
`runServerStart()` function. Initialization is sequential: logging, config, database,
Hub server, web server, broker, dispatcher. A refactoring design doc exists at
`.design/server-refactor.md` proposing splitting into 11 focused files.

**Proposed solution**:
- Identify independent initialization steps and run them in parallel via `errgroup`
- Lazy-initialize rarely-used subsystems (scheduler, GCP integrations, notifications)
- Split `runServerStart()` per the existing design doc

**Files**: `cmd/server.go`, `.design/server-refactor.md`

---

### #6 — Memory Allocation on Hot Paths (Medium Impact, Low Effort)

**Current state**: Zero `sync.Pool` usage in the entire codebase. JSON
marshal/unmarshal allocates fresh buffers on every WebSocket message. PTY data
chunks (up to 32KB per message) are allocated and discarded per-read.

**Proposed solution**:
- Add `sync.Pool` for byte buffers in WebSocket read/write paths
- Pool `bytes.Buffer` instances for JSON encoding
- Reuse PTY data buffers (32KB chunks)
- Pool `json.Decoder` instances where applicable

**Files**: `pkg/wsprotocol/connection.go`, `pkg/hub/pty_handlers.go`,
`pkg/hub/controlchannel.go`

---

### #7 — Event System Backpressure (Medium Impact, Low Effort)

**Current state**: `pkg/hub/events.go` uses 64-element buffered channels per SSE
subscriber. Sends are non-blocking (`select { case ch <- event: default: }`) —
events are silently dropped when a subscriber is slow. No metrics track dropped
events.

**Proposed solution**:
- Implement ring buffer with overflow counter
- Add Prometheus/OTel metrics for dropped events per subscriber
- Event coalescing for rapid state changes (e.g., multiple agent status updates
  within 100ms → single event)
- Adaptive buffer sizing based on subscriber consumption rate

**Files**: `pkg/hub/events.go`

---

### #8 — WebSocket Buffer Sizing (Low Impact, Low Effort)

**Current state**: Fixed 4KB read/write buffers for all WebSocket connections
(`pkg/wsprotocol/connection.go:30-35`). 64KB max message size. 256-element stream
data channels.

**Proposed solution**:
- Increase PTY connection buffers (16-32KB for read, 8KB for write)
- Keep control channel buffers at 4KB (small messages)
- Make buffer sizes configurable via connection options
- Use `bufio.Writer` pooling for write batching

**Files**: `pkg/wsprotocol/connection.go`

---

### #9 — Benchmarking & Profiling Infrastructure (Low Impact, Medium Effort)

**Current state**: Zero `Benchmark*` functions in the entire codebase (278 test
files, none with benchmarks). No pprof endpoints. No load testing framework. No
performance regression detection.

**Proposed solution**:
- Add `Benchmark*` tests for critical paths:
  - WebSocket message routing throughput
  - JSON serialization/deserialization
  - SQLite query patterns (agent listing, grove lookup)
  - PTY relay throughput
  - Agent provisioning time
- Add `net/http/pprof` endpoints to Hub server (debug mode)
- Create synthetic multi-agent load test

**Files**: New `*_test.go` files alongside existing packages, `cmd/server.go` (pprof)

---

### #10 — Dependency Tree Optimization (Low Impact, High Effort)

**Current state**: 63 direct dependencies in `go.mod` including full K8s client-go,
GCP SDK suite, Rclone, gRPC, and OTel exporters. All compiled into every binary
regardless of deployment mode.

**Proposed solution**:
- Build tags for deployment profiles:
  - `-tags local` — excludes K8s, GCP, cloud storage
  - `-tags hub` — excludes container runtime, PTY
  - `-tags broker` — excludes policy engine, full API
- Interface-based dependency injection (already partially done) to enable
  compile-time exclusion
- Evaluate lighter alternatives for edge cases (e.g., minimal K8s client)

**Files**: `go.mod`, package-level `*_no_*.go` / `*_with_*.go` build-tagged files

---

## Microkernel / Microservice Decomposition

### Can We Run Microkernels That Start Faster?

**Yes.** The architecture already logically separates Hub, Broker, and CLI but ships
as a single monolithic binary.

| Component | Current | Proposed | Startup Benefit |
|-----------|---------|----------|-----------------|
| CLI | `scion` (full binary) | `scion` (CLI-only) | ~70% faster |
| Hub Server | `scion server` | `scion-hub` | ~40% faster |
| Runtime Broker | `scion server` | `scion-broker` | ~50% faster |
| Web Server | embedded in Hub | Static + reverse proxy | Near-instant |
| sciontool | Already separate | No change | Already fast |

**Recommendation**: Do it incrementally using Go build tags, not separate repositories.
Keep single-binary mode for local development; produce separate binaries for hosted
deployments. Start by splitting CLI from server (highest ROI).

**Benefits**: Faster startup, lower memory footprint, independent scaling, faster
builds, fault isolation.

**Risks**: Increased operational complexity (multiple binaries). Mitigated by keeping
single-binary as default mode.

---

## Should SCION Be Rewritten in Rust?

### Answer: No.

| Factor | Go (Current) | Rust (Hypothetical) | Verdict |
|--------|-------------|---------------------|---------|
| Startup time | ~100-500ms | ~10-50ms | Go is fast enough |
| Memory usage | ~50-100MB | ~10-30MB | Not a bottleneck |
| Throughput | Adequate | Higher for CPU-bound | Irrelevant (I/O-bound) |
| Concurrency | Goroutines | async/tokio | Go slightly better here |
| Dev velocity | Fast compile, simple | Slow compile, steep curve | Go wins |
| Ecosystem | K8s/Docker native | Less mature | Go wins |
| Rewrite cost | N/A | 6-12 months, 253K LOC | Prohibitive |

**Why not**:
1. SCION is I/O-bound (containers, WebSockets, DB) — Rust's CPU advantages are
   irrelevant to the actual bottlenecks
2. Go dominates the container orchestration ecosystem (Docker, K8s, containerd)
3. 253K LOC rewrite would halt feature development for 6-12+ months
4. The top 10 Go improvements can be done in weeks with more impact
5. Team/ecosystem is Go-centric (Google Cloud Platform origin)

**When Rust would make sense**: Only if a specific hot-path component (e.g., PTY
relay) needed rewriting as a standalone binary — not a full rewrite.

---

## Implementation Roadmap

### Phase 1 — Quick Wins (1-2 weeks)
- #1 Web UI optimistic updates (design doc ready)
- #2 Add `-s -w` ldflags (one-line change)
- #6 sync.Pool for hot-path buffers
- #9 Initial benchmarks + pprof endpoints

### Phase 2 — Medium Effort (2-4 weeks)
- #3 SQLite read pool + indices
- #5 Parallel server initialization
- #7 Event backpressure + metrics
- #8 Configurable WebSocket buffers

### Phase 3 — Strategic (1-2 months)
- #4 Binary WebSocket frames for PTY/streams
- #10 Build tag deployment profiles
- Microkernel binary split
