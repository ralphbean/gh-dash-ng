## 1. Shared Grouped Table Presentation

- [x] 1.1 Add opt-in group-header metadata to the shared table model, keyed to logical selectable row indices.
- [x] 1.2 Render themed group headers above their first logical rows without adding cursor positions or changing row action indices.
- [x] 1.3 Update viewport layout and scrolling calculations to account for group-header lines in compact, non-compact, and separator modes.
- [x] 1.4 Add shared table and viewport tests proving headers render at boundaries and navigation skips directly between logical rows.

## 2. Agent-Activity Projections

- [x] 2.1 Add a stable agent-activity partition for visible PRs that uses the Fullsend status store only when integration is enabled.
- [x] 2.2 Make PR row building, current-row lookup, row count, and group-header metadata consume the same partitioned projection.
- [x] 2.3 Add the equivalent stable partition and canonical projection usage for visible issues.
- [x] 2.4 Add PR and issue section tests for mixed groups, single non-empty groups, preserved within-group order, unknown status, snoozed items, and disabled integration.

## 3. Selection and Actions

- [x] 3.1 Add stable PR and issue row identity helpers using item kind, repository, and number.
- [x] 3.2 Preserve the selected identity when Fullsend status updates rebuild and repartition section rows, falling back to existing cursor clamping when the item disappears.
- [x] 3.3 Add tests for selected and unselected items moving in both directions, including multiple status changes in one refresh.
- [x] 3.4 Add interaction tests showing next/previous and boundary navigation form one sequence and Enter opens the correct active-group item.
- [x] 3.5 Verify representative existing row actions target the correct selected item in both groups.

## 4. Activity Column and Visual Treatment

- [x] 4.1 Move the Fullsend activity column to the first position in PR column definitions and align PR row cell construction with it.
- [x] 4.2 Move the Fullsend activity column to the first position in issue column definitions and align issue row cell construction with it.
- [x] 4.3 Style the “No active agent” and “Active agent” headers with theme-derived distinctions and optional icons while retaining explicit text labels.
- [x] 4.4 Add rendering tests for header labels, feature-flag visibility, narrow layouts, and leftmost activity column order.

## 5. Verification

- [x] 5.1 Run formatting on changed Go files and run focused table, viewport, PR-section, and issue-section tests.
- [x] 5.2 Run `go test ./...`.
- [x] 5.3 Run the configured golangci-lint command and address findings introduced by this change.
