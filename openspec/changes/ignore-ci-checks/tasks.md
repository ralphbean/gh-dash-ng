## 1. Configuration and Matching

- [x] 1.1 Add `defaults.ignoredChecks` parsing, merging, defaults, and validation for glob syntax
- [x] 1.2 Add shared constructors for check-run, status-context, and check-suite identities plus table-driven matcher tests

## 2. Detail View Filtering

- [x] 2.1 Filter matching check runs, legacy statuses, pending suites, and unreported required checks from detail rendering
- [x] 2.2 Recompute detail summary categories and statistics from non-ignored checks and add mixed-result tests

## 3. Quota-Conscious List Filtering

- [x] 3.1 Preserve the existing list GraphQL query when no ignored-check patterns are configured
- [x] 3.2 Add an opt-in list query shape that fetches at most 100 status contexts per last commit when patterns are configured
- [x] 3.3 Compute list CI precedence from non-ignored contexts, including the all-ignored and mixed-failure cases
- [x] 3.4 Compare `rateLimit.cost` and response characteristics of existing and opt-in query shapes at representative limits and record the observed result

## 4. Documentation and Verification

- [x] 4.1 Document `defaults.ignoredChecks`, identity format, glob examples, opt-in API impact, and the 100-context ceiling
- [x] 4.2 Run focused config, data, PR-row, and PR-view tests
- [x] 4.3 Run `go test ./...`, formatting, and applicable lint checks
