## Why

gh-dash-ng users working in repositories with fullsend integration lack visibility into whether an AI agent is actively processing their issues and pull requests. When fullsend launches GitHub workflow jobs in response to issues or PRs, there's no indication in the dashboard that automated work is in progress, leading to uncertainty about whether to wait for the agent or proceed manually.

## What Changes

- Add a background process that monitors GitHub workflow runs for fullsend agent activity
- Detect fullsend dispatch jobs and follow their annotations to identify agent type and completion status
- Add a new column to PR and issue list views displaying a spinning icon when a fullsend agent is actively processing that item
- Add the same status indicator to the preview/details views for PRs and issues
- Implement local caching to reduce GitHub API calls by remembering completed agent runs
- Continue checking for new dispatch jobs even when previous runs have completed

## Capabilities

### New Capabilities

- `fullsend-integration/workflow-monitor`: Background monitoring of GitHub workflow runs to detect fullsend agent activity, with caching to minimize API usage
- `fullsend-integration/status-indicator`: Visual status indicator column and UI component showing active fullsend agent processing

### Modified Capabilities

<!-- No existing capabilities are being modified -->

## Impact

- **UI Changes**: New column in PR and issue list views; new indicator in preview/details views
- **Background Processing**: New async process polling GitHub API for workflow status
- **GitHub API Usage**: Additional API calls for workflow run queries, mitigated by intelligent caching
- **Data Model**: New data structures to track workflow runs, agent status, and cache state
- **Dependencies**: Potential new dependencies for animated UI elements (spinning icon) and async task management
