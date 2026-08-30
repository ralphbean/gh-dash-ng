## Context

gh-dash-ng is a terminal UI (TUI) application built with bubbletea that displays GitHub issues and pull requests. The current architecture:

- Data fetching uses GraphQL queries through `internal/data/` packages
- Row rendering is handled by `issuerow` and `prrow` components with `ToTableRow()` methods
- Background tasks use bubbletea's `tea.Cmd` system for async operations
- No persistent background workers exist; updates are event-driven
- Mutations go through `gh` CLI subprocesses via `fireTask()` pattern

See proposal.md for motivation.

## Goals / Non-Goals

**Goals:**
- Add fullsend workflow monitoring without blocking the UI or degrading performance
- Display real-time agent status with minimal GitHub API overhead
- Support both PR and issue lists with consistent behavior
- Cache workflow completion to avoid redundant API calls

**Non-Goals:**
- Integration with fullsend configuration or agent control (read-only status display)
- Detailed workflow logs or failure diagnostics (link to GitHub Actions for details)
- Historical agent run tracking beyond current/recent activity
- Support for non-GitHub hosting platforms (fullsend is GitHub Actions-specific)

## Decisions

### 1. Background Monitoring Architecture

**Decision:** Use bubbletea's `tea.Tick` command pattern with per-repository polling intervals rather than a persistent goroutine worker.

**Rationale:**
- Bubbletea's command system is the established pattern in this codebase for async operations
- `tea.Tick` integrates cleanly with the message loop and handles cleanup automatically
- Per-repository polling allows intelligent rate limiting based on visible items
- Avoids complexity of coordinating a separate worker with the TUI lifecycle

**Alternatives Considered:**
- **Persistent goroutine worker**: More complex lifecycle management; bubbletea doesn't encourage this pattern
- **Event-driven only (no polling)**: Misses status changes; GitHub doesn't push workflow updates to clients

### 2. Workflow Detection Strategy

**Decision:** Detect fullsend agents by querying workflow runs named "fullsend", fetching their jobs, and extracting agent type from job names with pattern `"dispatch / <AgentType>"`.

**Rationale:**
- Fullsend uses a reusable workflow architecture where all agent execution happens as jobs within a single "fullsend" workflow run
- Job names follow a consistent pattern: `"dispatch / Review"`, `"dispatch / Code"`, `"dispatch / Triage"`, `"dispatch / Fix"`, etc.
- Agent type is directly extractable via `strings.TrimPrefix(jobName, "dispatch / ")`
- No workflow file parsing needed - all information is available from the GitHub Actions API
- Works for any agent type (code, review, triage, fix, retro, prioritize) without hardcoding

**Implementation verified via API testing:**
- Queried `ralphbean/gh-dash-ng` workflow runs
- Confirmed job naming pattern across multiple fullsend runs
- All agent types appear as distinct jobs within the same workflow run

**Alternatives Considered (and rejected):**
- **Parse workflow YAML files**: Unnecessary complexity; job names provide all needed information
- **Issue/PR label-based detection**: Not reliable; fullsend may not always apply labels
- **Comment parsing**: Fragile; comment format could change

### 3. Cache Implementation

**Decision:** In-memory cache with two layers:
1. **Workflow run cache**: Maps workflow run ID → terminal status (completed/failed/cancelled)
2. **Issue/PR check cache**: Maps repo+number → last check timestamp + active run IDs

Both caches use TTL-based eviction (completed runs: never re-check; last check timestamp: 5 minutes).

**Rationale:**
- In-memory sufficient for session lifetime; no need to persist across restarts
- Two-layer design balances API efficiency with fresh dispatch detection
- Completed runs never change; safe to cache indefinitely per session
- 5-minute TTL on issue/PR checks ensures new dispatches are detected promptly

**Alternatives Considered:**
- **Disk-based persistent cache**: Added complexity with marginal benefit (session lifetime is typically short)
- **Single-layer cache**: Either misses new dispatches (if caching too aggressively) or doesn't reduce API calls (if not caching enough)

### 4. Data Model Extension

**Decision:** Add `FullsendStatus` field to `IssueData` and `PullRequestData` structs:
```go
type FullsendStatus struct {
    ActiveAgents []ActiveAgent
    LastChecked  time.Time
}

type ActiveAgent struct {
    Type        string // "code", "review", "triage", "fix", etc.
    WorkflowID  int64
    Status      string // "queued", "in_progress"
    StartedAt   time.Time
}
```

**Rationale:**
- Extends existing data structures without breaking changes
- Supports multiple concurrent agents (e.g., review + fix running simultaneously)
- `LastChecked` enables smart polling decisions in the monitor
- Minimal memory footprint (only populated for items with active agents)

**Alternatives Considered:**
- **Separate data structure**: Requires parallel state management and synchronization complexity
- **Single agent only**: Doesn't handle concurrent workflows (e.g., auto-review during manual code run)

### 5. UI Column Rendering

**Decision:** Add a new `renderFullsendStatus()` method to `issuerow` and `prrow`, returning a single column with:
- Animated spinner (using Unicode spinner characters rotated via tick messages) when agents active
- Agent type displayed dynamically from job name (e.g., "Review", "Code", "Triage", "Fix")
- First letter abbreviation for compact display (e.g., "R", "C", "T", "F")
- Empty string when no active agents
- Multiple indicators stacked/concatenated for concurrent agents

**Rationale:**
- Follows existing `render*()` pattern in row components
- Unicode spinners are lightweight and work in all terminals
- First-letter abbreviation is computed dynamically, supporting any future agent types automatically
- Reuses existing animation tick system from other TUI components
- No hardcoded agent type mappings needed

**Alternatives Considered:**
- **Color-coded icons only**: Harder to distinguish agent types without text
- **Full agent names**: Consumes too much horizontal space in constrained terminal UI
- **Separate column per agent type**: Wastes space when most items have zero or one agent
- **Hardcoded abbreviations**: Breaks when new agent types are added; dynamic approach is more flexible

### 6. API Rate Limit Mitigation

**Decision:**
- Poll only repositories with visible issues/PRs (current tab/filter)
- Batch workflow run queries using GraphQL where possible
- Adaptive polling: 30s interval for active workflows, 5min for idle repositories
- Respect `X-RateLimit-Remaining` header and back off exponentially if nearing limit

**Rationale:**
- Most users view a small subset of repositories at any time
- GraphQL batching reduces request count significantly
- Active workflows need frequent updates; idle repos do not
- GitHub's rate limit headers provide explicit guidance for throttling

**Alternatives Considered:**
- **Poll all repositories regardless of visibility**: Wastes API quota on off-screen items
- **Fixed polling interval**: Either too aggressive (wastes quota) or too slow (stale status)
- **No rate limit awareness**: Risks exhausting quota and blocking legitimate dashboard usage

## Risks / Trade-offs

**[Risk]** GitHub API rate limits exhausted by workflow polling → **Mitigation:** Adaptive polling intervals, visibility-based filtering, exponential backoff on rate limit warnings

**[Risk]** Fullsend job naming pattern changes → **Mitigation:** Simple pattern matching (`"dispatch / "` prefix); if pattern changes, graceful degradation (no agent type detected but workflow still tracked)

**[Risk]** Concurrent agent workflows create confusing UI (too many indicators) → **Mitigation:** UX rule: max 2 agent indicators per row; if >2 active, show count badge ("3 agents")

**[Risk]** Spinner animation causes high CPU usage in large lists → **Mitigation:** Throttle animation tick rate; disable animations in lists >100 items

**[Risk]** Workflow run queries slow down dashboard startup → **Mitigation:** Lazy-load fullsend status (initial render without status; populate on first tick)

**[Trade-off]** In-memory cache loses state on restart → Acceptable: users restart dashboard infrequently; re-checking on startup is fast enough

**[Trade-off]** 30-second polling delay means status updates aren't instant → Acceptable: workflow state changes are not time-critical for dashboard users

## Migration Plan

1. **Phase 1 - Data Layer (no UI changes)**
   - Add `FullsendStatus` to data structs
   - Implement workflow monitor with caching
   - Add feature flag to enable/disable monitoring

2. **Phase 2 - UI Integration**
   - Add column rendering functions
   - Wire monitor into row components
   - Enable spinner animations

3. **Phase 3 - Optimization**
   - Add adaptive polling based on visibility
   - Implement rate limit backoff
   - Performance testing with large lists

**Rollback:** Feature flag allows disabling without code changes; no database migrations or persistent state to roll back.

## Open Questions

None - all design decisions needed for implementation are resolved.
