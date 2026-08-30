## Purpose

Provides visual indication of active fullsend agent processing in the dashboard interface through a dedicated status column and UI indicators.

## ADDED Requirements

### Requirement: Status column in list views

The system SHALL display a status column in PR and issue list views showing fullsend agent activity.

#### Scenario: Active agent indication
- **WHEN** a fullsend agent workflow is actively processing (queued or in_progress) for an issue or PR
- **THEN** the status column displays an animated spinning icon
- **AND** the icon indicates the agent type (code, review, triage, fix, etc.)

#### Scenario: No active agent
- **WHEN** no fullsend agent workflow is currently processing an issue or PR
- **THEN** the status column is blank or displays a placeholder
- **AND** no icon is shown

#### Scenario: Multiple concurrent agents
- **WHEN** multiple fullsend agent workflows are active for the same issue or PR
- **THEN** the status column displays an indicator for each active agent
- **OR** shows a combined indicator representing multiple agents

### Requirement: Status indicator in details view

The system SHALL display fullsend agent status in the preview/details view for issues and pull requests.

#### Scenario: Details view with active agent
- **WHEN** viewing details of an issue or PR with an active fullsend agent
- **THEN** the details view displays the agent status indicator
- **AND** shows the agent type and current workflow status

#### Scenario: Details view without active agent
- **WHEN** viewing details of an issue or PR with no active fullsend agent
- **THEN** the details view does not display a status indicator
- **OR** displays an inactive/idle status if the feature is enabled

### Requirement: Visual feedback and animation

The system SHALL provide clear visual feedback indicating active processing through animation.

#### Scenario: Animated spinner
- **WHEN** displaying an active fullsend agent
- **THEN** the indicator shows a continuously spinning or pulsing animation
- **AND** the animation runs smoothly without blocking the UI

#### Scenario: Agent type differentiation
- **WHEN** multiple agent types are available
- **THEN** the indicator visually distinguishes between agent types (code, review, triage, fix, etc.)
- **AND** uses consistent visual language (colors, icons, or labels)

### Requirement: Performance and responsiveness

The system SHALL update status indicators efficiently without degrading dashboard performance.

#### Scenario: Real-time status updates
- **WHEN** a fullsend agent workflow status changes (starts, completes, fails)
- **THEN** the UI updates the status indicator within a reasonable timeframe (under 60 seconds)
- **AND** the update does not cause visible UI lag or freezing

#### Scenario: List view performance
- **WHEN** displaying a list with many items (50+ issues/PRs)
- **THEN** the status column renders efficiently
- **AND** animations do not consume excessive CPU resources

### Requirement: Accessibility

The system SHALL make status indicators accessible to users with assistive technologies.

#### Scenario: Screen reader support
- **WHEN** a screen reader focuses on a status indicator
- **THEN** it announces the fullsend agent status (e.g., "Code agent processing" or "No active agent")
- **AND** provides meaningful context about the workflow state

#### Scenario: Keyboard navigation
- **WHEN** navigating the list view with keyboard only
- **THEN** the status column is accessible via tab navigation
- **AND** status information is available without requiring mouse interaction
