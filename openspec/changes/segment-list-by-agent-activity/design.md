## Context

PR and issue sections currently build a flat `table.Row` slice and map the table cursor index directly back to the corresponding filtered domain item. The shared table and list viewport assume each rendered row is selectable and has a uniform height. Fullsend activity is stored separately from fetched GitHub data and arrives asynchronously, so a status refresh can change an item's group without changing the fetched item slice. See `proposal.md` for motivation and `specs/agent-activity-list-grouping/spec.md` for required behavior.

## Goals / Non-Goals

**Goals:**

- Use one canonical ordered item projection for rendering, cursor lookup, row actions, row counts, and pagination decisions.
- Add presentation-only group boundaries without inserting selectable pseudo-items into domain data.
- Preserve selection by stable item identity across asynchronous regrouping.
- Keep shared table behavior unchanged for sections that do not opt into grouping.

**Non-Goals:**

- Changing Fullsend polling, workflow detection, or the definition of an active agent.
- Adding a user-configurable grouping mode or independently collapsible groups.
- Persisting grouping or selection state across application restarts.
- Changing the ordering configured by each PR or issue section within a group.

## Decisions

### Build a stable, partitioned item projection in each supported section

PR and issue sections will derive their visible items by first applying existing visibility filters and then, when Fullsend integration is enabled, stable-partitioning them into no-active-agent and active-agent slices. The slices are concatenated to form the canonical selectable order, accompanied by group-boundary metadata.

Both `BuildRows` and `GetCurrRow` will consume this same projection. This extends the existing pattern that prevents snoozed rows from breaking cursor-to-data mapping and avoids maintaining a second index translation table.

Alternative considered: sort the fetched backing slices in place. This would destroy the original within-group ordering, complicate refresh reconciliation, and mutate API-derived state for a presentation concern.

### Model group headers as decorations on logical rows

The shared table will gain optional metadata that associates a header with the first logical row of a group. The viewport's selectable item count remains the number of domain rows, and cursor movement continues to operate only on those logical rows. Rendering and viewport height calculations will account for the extra header lines without exposing them as cursor positions.

The table header decoration API will be opt-in so repository, notification, and other tables retain their existing behavior. Headers will use theme-derived styling and explicit text labels; icons can supplement but not replace those labels.

Alternative considered: insert headers as ordinary `table.Row` values and teach each section to skip them. That spreads index translation and skip logic across navigation, selection, paging, and every row action, making it too easy for Enter or another action to target the wrong item.

### Preserve selection using stable row identity during status-driven rebuilds

Before rebuilding rows in response to Fullsend status updates, a section will capture the selected item's stable identity: item kind, repository owner/name, and issue or PR number. After rebuilding the partitioned projection, it will locate that identity and set the cursor to its new logical index. If the selected item is no longer present, existing cursor clamping behavior applies.

This identity-based approach is necessary because retaining the numeric cursor can silently select a different item when an item crosses the boundary.

Alternative considered: adjust the cursor by the change in active item count. That fails when multiple statuses change in one poll and is sensitive to items on either side of the selection.

### Treat only currently active agents as active-group membership

Membership uses the existing Fullsend status store and active-agent definition: at least one agent reported as queued or in progress. Missing, not-yet-polled, completed, failed, and empty statuses remain in the no-active-agent group. This provides deterministic startup behavior and matches the status indicator's existing semantics.

### Move the existing Fullsend column definition and cell to index zero

PR and issue column definitions and row cell construction will place the existing Fullsend activity indicator first whenever the feature is enabled. The column remains hidden under the existing feature flag rather than introducing configuration or data-model changes.

## Risks / Trade-offs

- **[Risk] Extra header lines conflict with viewport scrolling assumptions** → Extend table/viewport layout tests to cover compact, non-compact, separator, paging, and boundary-scroll modes; keep logical item count separate from rendered line count.
- **[Risk] Status refresh causes selection to jump to another item** → Centralize identity capture/restoration and test simultaneous status changes around the selected row.
- **[Risk] PR and issue implementations drift** → Share partition/header metadata helpers where domain types permit, while keeping domain identity extraction explicit.
- **[Risk] Unknown status is visually described as having no active agent** → Use wording such as “No active agent,” not “verified idle,” and immediately regroup after successful status updates.
- **[Trade-off] Stable partition overrides global interleaving from the configured sort** → Preserve that sort within each group; prioritizing agent availability is the intentional primary grouping key.

## Migration Plan

No persisted state or API migration is required. Ship the behavior behind the existing Fullsend integration feature flag. Rollback consists of reverting the grouped projection, table decorations, and column reordering; underlying Fullsend status data remains compatible.
