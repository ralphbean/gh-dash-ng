## Why

Some informational or automation checks intentionally do not represent pull-request readiness, but GitHub includes them in its aggregate status. Users need to exclude configured checks from both the PR-list CI indicator and the detailed checks view without unexpectedly exhausting the GitHub API quota.

## What Changes

- Add global configuration for glob patterns identifying ignored checks.
- Match patterns against stable, displayed check identities such as `creator/workflow/check`.
- Exclude matching checks from the checks detail view, its categories, and its summary statistics.
- Compute the PR-list CI state without matching checks while minimizing and measuring added GraphQL cost.
- Preserve GitHub's existing aggregate rollup path when no ignore patterns are configured, so existing users incur no added API cost.

## Capabilities

### New Capabilities

- `ignored-ci-checks`: Configure glob patterns for checks that do not contribute to displayed CI state.

### Modified Capabilities

None.

## Impact

- Configuration parsing, defaults, validation, examples, and user documentation.
- Pull-request list GraphQL data and local CI rollup calculation when the feature is configured.
- Detailed checks rendering and statistics.
- GraphQL query cost for users who configure ignored checks; the design must bound and document this cost and retain a zero-cost default path.
