## Why

Pull requests and issues that already have a Fullsend agent working on them compete visually with items that may need the user's attention. Separating those states lets users focus on unattended work while still keeping every item available in the same list workflow.

## What Changes

- Partition PR and issue list rows into an upper group with no currently running Fullsend agent and a lower group with one or more running agents.
- Render a small, visually distinct header for each non-empty group so the reason for the grouping is clear.
- Preserve one continuous selection and navigation order across both groups; group headers are presentation-only and cannot receive focus.
- Keep all existing row actions, including opening details with Enter, working identically for rows in either group.
- Repartition the visible list when Fullsend status updates, while preserving the selected item when it remains present.
- Move the Fullsend activity indicator to the leftmost table column so the grouping signal is easy to scan.
- Retain the current unsegmented list behavior when Fullsend integration is disabled.

## Capabilities

### New Capabilities

- `agent-activity-list-grouping`: Groups issue and pull-request list rows by current Fullsend agent activity while preserving continuous navigation and actions.

### Modified Capabilities

None.

## Impact

- PR and issue section row construction, ordering, selection mapping, and rendering.
- Shared table rendering may need support for non-selectable group headers or row decorations.
- PR and issue table column definitions and Fullsend status refresh handling.
- Tests for grouping, navigation, selection stability, actions, feature-flag behavior, and column order.
- No GitHub API, configuration format, or persisted-data changes are expected.
