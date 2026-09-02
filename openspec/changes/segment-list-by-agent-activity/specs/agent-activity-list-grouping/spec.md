## Purpose

Makes PR and issue lists easier to triage by separating unattended items from items currently being processed by Fullsend agents without splitting the interaction model.

## ADDED Requirements

### Requirement: Lists group items by current agent activity
When Fullsend integration is enabled, the system SHALL display PR and issue list items with no active Fullsend agent before items with one or more active Fullsend agents. An active agent is an agent whose current status is queued or in progress. The system SHALL preserve the existing relative order of items within each group.

#### Scenario: Mixed activity list
- **WHEN** a list contains both items without an active Fullsend agent and items with an active Fullsend agent
- **THEN** all items without an active agent appear first in their prior relative order
- **AND** all items with an active agent appear afterward in their prior relative order

#### Scenario: Status has not reported an active agent
- **WHEN** an item has no known queued or in-progress Fullsend agent
- **THEN** the item appears in the upper, no-active-agent group

#### Scenario: Fullsend integration is disabled
- **WHEN** Fullsend integration is disabled
- **THEN** the list retains its existing unsegmented order and presentation

### Requirement: Groups have distinct presentation-only headers
The system SHALL render a concise, visually distinct header immediately above each non-empty agent-activity group. Each header SHALL communicate whether its group contains items with no active agent or items with an active agent, using text in addition to any icon or color.

#### Scenario: Both groups contain items
- **WHEN** both agent-activity groups are non-empty
- **THEN** the list displays a no-active-agent header above the upper group and an active-agent header above the lower group

#### Scenario: One group is empty
- **WHEN** only one agent-activity group contains items
- **THEN** the list displays the header for the non-empty group and does not display an empty group

#### Scenario: Header accessibility
- **WHEN** a group header uses an icon or color to distinguish activity
- **THEN** visible text also identifies the group's agent-activity state

### Requirement: Grouped rows form one navigation sequence
The system SHALL expose grouped items as one continuous selectable sequence. Group headers SHALL not be selectable, and crossing a group boundary SHALL require the same navigation command as moving between adjacent items elsewhere in the list.

#### Scenario: Move from upper group to lower group
- **WHEN** selection is on the final item in the upper group and the user moves to the next item
- **THEN** selection moves directly to the first item in the lower group

#### Scenario: Move from lower group to upper group
- **WHEN** selection is on the first item in the lower group and the user moves to the previous item
- **THEN** selection moves directly to the final item in the upper group

#### Scenario: Navigate across a single populated group
- **WHEN** only one group contains items
- **THEN** first, last, next, previous, and paging navigation behave as they do in an unsegmented list

### Requirement: Existing item actions work in either group
The system SHALL apply every list-row action to the selected item regardless of its agent-activity group.

#### Scenario: Open details for an active-agent item
- **WHEN** the user selects an item in the active-agent group and presses Enter
- **THEN** the system opens the details view for that item

#### Scenario: Invoke another row action
- **WHEN** the user invokes an existing row action on an item in either group
- **THEN** the action targets that selected item with unchanged behavior

### Requirement: Grouping responds to status changes without losing identity
The system SHALL update grouping when current Fullsend agent status changes. If the selected item remains in the list, the system SHALL keep that item selected after regrouping even when its position changes.

#### Scenario: Selected item becomes active
- **WHEN** the selected item gains a queued or in-progress Fullsend agent
- **THEN** it moves to the active-agent group and remains selected

#### Scenario: Selected item becomes idle
- **WHEN** the selected item no longer has any queued or in-progress Fullsend agent
- **THEN** it moves to the no-active-agent group and remains selected

#### Scenario: Unselected item changes group
- **WHEN** an unselected item changes agent-activity group
- **THEN** the selected item remains selected if it is still present

### Requirement: Agent activity is the first list column
When Fullsend integration is enabled, the system SHALL display the Fullsend agent activity indicator as the leftmost PR and issue list column.

#### Scenario: Render a Fullsend-enabled list
- **WHEN** a PR or issue list is rendered with Fullsend integration enabled
- **THEN** the activity indicator appears before every other item column

