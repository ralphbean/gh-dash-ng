## Purpose

Monitors GitHub workflow runs to detect active fullsend agent activity on issues and pull requests, with intelligent caching to minimize API usage while providing real-time status updates.

## ADDED Requirements

### Requirement: Background workflow monitoring

The system SHALL periodically query the GitHub API for workflow runs associated with issues and pull requests visible in the current view.

#### Scenario: Initial workflow discovery
- **WHEN** the dashboard loads or refreshes
- **THEN** the system queries GitHub workflow runs for all visible issues and PRs
- **AND** identifies fullsend-related workflows by name ("fullsend")

#### Scenario: Periodic status updates
- **WHEN** a fullsend agent workflow is detected as in-progress
- **THEN** the system polls for status updates at regular intervals
- **AND** stops polling once the workflow completes or fails

#### Scenario: Successful poll replaces prior status
- **GIVEN** an issue or PR previously had an active fullsend agent
- **WHEN** a successful poll finds no queued or in-progress fullsend agent for that item
- **THEN** the system records an empty active-agent status for the item
- **AND** removes the active status indicator from the UI

#### Scenario: Failed poll preserves prior status
- **GIVEN** an issue or PR has a previously recorded fullsend status
- **WHEN** the workflow-run or workflow-job query for that item fails
- **THEN** the system preserves the previously recorded status
- **AND** retries according to the error-handling policy

### Requirement: Fullsend workflow detection

The system SHALL identify fullsend agents by querying jobs within workflow runs named "fullsend" and extracting agent types from job names.

#### Scenario: Workflow run identification
- **WHEN** checking workflow runs for an issue or PR
- **THEN** the system identifies workflow runs with name "fullsend"
- **AND** queries the jobs for those workflow runs

#### Scenario: Targeted item polling
- **WHEN** a delayed poll checks one specific issue or PR
- **THEN** the system matches workflow runs to that requested item
- **AND** does not attribute an active workflow for another item to the requested item
- **AND** records an empty active-agent status when no matching active workflow exists

#### Scenario: Agent type extraction from job names
- **WHEN** examining jobs from a fullsend workflow run
- **THEN** the system identifies jobs matching pattern "dispatch / <AgentType>"
- **AND** extracts the agent type by removing the "dispatch / " prefix
- **AND** records the job status (queued, in_progress, completed) and conclusion (success, failure, cancelled)

### Requirement: Completion status caching

The system SHALL cache the completion status of fullsend workflow runs to avoid redundant API queries.

#### Scenario: Cache completed workflows
- **WHEN** a workflow run transitions to a terminal state (completed, failed, cancelled)
- **THEN** the system records the workflow run ID and final status in the cache
- **AND** does not query that specific workflow run again in future checks

#### Scenario: Fresh dispatch detection
- **WHEN** checking an issue or PR with cached completed workflows
- **THEN** the system still queries for new workflow runs
- **AND** detects if a new fullsend agent has been dispatched since the last check

#### Scenario: Cache invalidation on new activity
- **WHEN** a new fullsend workflow is dispatched for an issue or PR
- **THEN** the system adds the new workflow to the monitoring set
- **AND** caches it only after it reaches a terminal state

### Requirement: API rate limiting protection

The system SHALL implement safeguards to prevent excessive GitHub API usage.

#### Scenario: Polling interval control
- **WHEN** monitoring active workflows
- **THEN** the system SHALL NOT poll more frequently than once every 30 seconds per workflow
- **AND** backs off if GitHub API rate limits are approached

#### Scenario: Batch workflow queries
- **WHEN** checking status for multiple issues and PRs
- **THEN** the system batches API requests where possible
- **AND** prioritizes visible items in the current view

### Requirement: Error handling and graceful degradation

The system SHALL handle API errors and network failures without disrupting the dashboard.

#### Scenario: API failure handling
- **WHEN** a GitHub API request fails
- **THEN** the system logs the error
- **AND** continues monitoring other workflows
- **AND** retries the failed request after a backoff period

#### Scenario: Missing workflow data
- **WHEN** workflow run data cannot be retrieved
- **THEN** the system preserves the last known status for that issue or PR
- **AND** retries the request after a backoff period
