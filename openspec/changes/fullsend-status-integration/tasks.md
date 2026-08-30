## 1. Data Model Extensions

- [x] 1.1 Add `FullsendStatus` struct to `internal/data/issueapi.go` with `ActiveAgents` and `LastChecked` fields
- [x] 1.2 Add `ActiveAgent` struct with `Type`, `WorkflowID`, `Status`, and `StartedAt` fields
- [x] 1.3 Add `FullsendStatus` field to `IssueData` struct in `internal/data/issueapi.go`
- [x] 1.4 Add `FullsendStatus` field to `PullRequestData` struct (locate in prapi.go or similar)
- [x] 1.5 Add feature flag `fullsendIntegration` to `internal/config/feature_flags.go`

## 2. Cache Implementation

- [x] 2.1 Create `internal/data/fullsendcache.go` with two-layer cache structure
- [x] 2.2 Implement workflow run cache (run ID → terminal status map) with no TTL for completed runs
- [x] 2.3 Implement issue/PR check cache (repo+number → last check timestamp + active run IDs) with 5-minute TTL
- [x] 2.4 Add cache eviction logic for the issue/PR check cache
- [x] 2.5 Add thread-safe access methods for both cache layers

## 3. GitHub API Integration

- [x] 3.1 Create `internal/data/workflowapi.go` for GitHub Actions workflow queries
- [x] 3.2 Implement function to query workflow runs for a specific repository and issue/PR number
- [x] 3.3 Add GraphQL query to fetch workflow run status (queued, in_progress, completed, failed, cancelled)
- [x] 3.4 Implement batch workflow run query using GraphQL for multiple issues/PRs
- [x] 3.5 Add rate limit header parsing (`X-RateLimit-Remaining`) and tracking
- [x] 3.6 Implement exponential backoff when approaching rate limits

## 4. Workflow Detection via Job Names

- [x] 4.1 Create `internal/data/fullsenddetector.go` for fullsend agent detection from job names
- [x] 4.2 Implement function to extract agent type from job name pattern `"dispatch / <AgentType>"`
- [x] 4.3 Add function to detect active fullsend agents from workflow run jobs
- [x] 4.4 Handle multiple concurrent agents for the same issue/PR
- [x] 4.5 Add graceful fallback when job name pattern doesn't match (return unknown agent type)

## 5. Background Monitoring with Bubbletea

- [x] 5.1 Create `internal/tui/components/fullsendmonitor/monitor.go` for monitor component
- [x] 5.2 Implement `tea.Cmd` that returns a tick message for periodic polling
- [x] 5.3 Add message type `FullsendStatusUpdatedMsg` with updated agent status
- [x] 5.4 Implement adaptive polling intervals (30s for active workflows, 5min for idle)
- [x] 5.5 Add logic to poll only visible repositories (filter by current tab)
- [x] 5.6 Implement lazy loading (skip initial fetch, populate on first tick)
- [x] 5.7 Add error handling that logs and continues on API failures
- [x] 5.8 Implement retry logic with backoff for failed workflow queries

## 6. UI Column Rendering for Issues

- [x] 6.1 Add `renderFullsendStatus()` method to `internal/tui/components/issuerow/issuerow.go`
- [x] 6.2 Implement spinner animation using Unicode spinner characters (⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏)
- [x] 6.3 Add agent type abbreviation rendering (first letter of agent name, e.g., "C" for Code)
- [x] 6.4 Implement logic for multiple concurrent agents (max 2 indicators, else show count badge)
- [x] 6.5 Update `ToTableRow()` to include fullsend status column
- [x] 6.6 Add animation tick message handling in issue row update logic
- [x] 6.7 Implement performance optimization: disable animations for lists >100 items

## 7. UI Column Rendering for PRs

- [x] 7.1 Add `renderFullsendStatus()` method to `internal/tui/components/prrow/prrow.go` (or similar)
- [x] 7.2 Implement spinner animation (reuse logic from issue row)
- [x] 7.3 Add agent type abbreviation rendering (first letter of agent name)
- [x] 7.4 Implement logic for multiple concurrent agents
- [x] 7.5 Update `ToTableRow()` to include fullsend status column
- [x] 7.6 Add animation tick message handling in PR row update logic
- [x] 7.7 Implement performance optimization: disable animations for lists >100 items

## 8. Details View Integration

- [x] 8.1 Add fullsend status display to `internal/tui/components/issueview/` (locate correct file)
- [x] 8.2 Add fullsend status display to `internal/tui/components/prview/` (locate correct file)
- [x] 8.3 Implement status indicator with agent type and workflow status text
- [x] 8.4 Add conditional rendering (only show when active agents exist)
- [ ] 8.5 Wire status updates from monitor into view components

## 9. Main TUI Integration

- [x] 9.1 Integrate fullsend monitor into main TUI model (`internal/tui/ui.go`)
- [x] 9.2 Wire `FullsendStatusUpdatedMsg` handling in main Update function
- [x] 9.3 Add monitor initialization when feature flag is enabled
- [x] 9.4 Propagate status updates to issue and PR sections
- [x] 9.5 Add animation tick command to main update loop
- [ ] 9.6 Ensure monitor cleanup on application exit

## 10. Accessibility Support

- [ ] 10.1 Add ARIA-style text descriptions for fullsend status in row rendering
- [ ] 10.2 Ensure status column is accessible via keyboard navigation
- [ ] 10.3 Test with terminal screen readers (if available) or add text-only mode
- [ ] 10.4 Document accessibility features in code comments

## 11. Testing and Validation

- [ ] 11.1 Write unit tests for workflow detection and parsing logic
- [ ] 11.2 Write unit tests for two-layer cache implementation
- [ ] 11.3 Add integration test for GitHub API workflow queries
- [ ] 11.4 Test rate limit backoff behavior
- [ ] 11.5 Test UI rendering with 0, 1, 2, and 3+ active agents
- [ ] 11.6 Verify animation performance with large lists (100+ items)
- [ ] 11.7 Test cache behavior across dashboard restarts
- [ ] 11.8 Verify graceful degradation when GitHub API is unavailable

## 12. Documentation and Configuration

- [x] 12.1 Document fullsend integration feature flag in README or docs
- [x] 12.2 Add code comments explaining workflow detection strategy
- [x] 12.3 Document cache TTL values and rationale
- [ ] 12.4 Add troubleshooting guide for common integration issues
- [ ] 12.5 Update configuration examples to show feature flag usage
