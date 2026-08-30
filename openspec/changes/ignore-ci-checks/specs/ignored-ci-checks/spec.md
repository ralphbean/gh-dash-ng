## Purpose

Allows users to exclude selected automation checks from gh-dash-ng's displayed pull-request readiness while retaining all other CI results.

## ADDED Requirements

### Requirement: Configure ignored checks with glob patterns
The system SHALL accept an ordered list of glob patterns in global defaults configuration and SHALL validate every configured pattern when loading configuration.

#### Scenario: Valid patterns load
- **WHEN** configuration contains one or more syntactically valid ignored-check glob patterns
- **THEN** the configuration loads and the patterns are available to all pull-request views

#### Scenario: Invalid pattern is rejected
- **WHEN** configuration contains a syntactically invalid ignored-check glob pattern
- **THEN** configuration loading fails with an error identifying the invalid pattern

### Requirement: Match the visible check identity
The system SHALL match each pattern against the slash-delimited identity displayed for that check, including available creator, workflow, and check-name components.

#### Scenario: Check run matches a pattern
- **WHEN** a check is displayed as `fullsend-ai-bot/fullsend/dispatch-pr` and the configured pattern is `fullsend-ai-*/fullsend/dispatch*`
- **THEN** the check is treated as ignored

#### Scenario: No pattern matches
- **WHEN** a check's displayed identity matches none of the configured patterns
- **THEN** the check contributes to CI status normally

### Requirement: Exclude ignored checks from CI presentation
The system SHALL omit ignored checks from the detailed checks view and SHALL exclude their state from detailed statistics and the PR-list CI indicator.

#### Scenario: Ignored failure is the only failure
- **WHEN** an ignored check fails and all non-ignored checks succeed
- **THEN** the PR-list CI indicator reports success and the ignored check does not appear in check details

#### Scenario: Non-ignored failure remains
- **WHEN** both an ignored check and a non-ignored check fail
- **THEN** the PR-list CI indicator reports failure and check details display only the non-ignored failure

#### Scenario: Every check is ignored
- **WHEN** all reported checks match ignored-check patterns
- **THEN** the system displays an unknown or empty CI state rather than success

### Requirement: Preserve API usage for unconfigured users
The system MUST NOT fetch individual list-view check identities solely for ignored-check filtering when no ignored-check patterns are configured.

#### Scenario: Ignore list is empty
- **WHEN** no ignored-check patterns are configured
- **THEN** the PR-list data request and GitHub aggregate status behavior remain unchanged
