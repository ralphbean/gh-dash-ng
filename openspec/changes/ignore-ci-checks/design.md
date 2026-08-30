## Context

The PR detail query already fetches up to 100 status contexts and can filter them locally. The high-frequency PR list query fetches only GitHub's aggregate status, so it cannot remove a named check without requesting identities. The default list limit is 20 and the default refresh interval is 30 minutes.

GitHub estimates GraphQL primary cost by adding the requests needed for unique connections, dividing by 100, and rounding, with a one-point minimum. Adding one nested contexts page for each of 20 PRs will commonly add zero or one reported point to a list query, but exact cost depends on the rest of the query and GitHub's current calculation. The larger concern is response size: a `contexts(last: 100)` connection can add as many as 2,000 nodes to a default 20-PR section.

## Goals / Non-Goals

**Goals:**

- Keep the existing query and cost exactly unchanged unless filtering is configured.
- Provide identical matching and rollup semantics in list and detail views.
- Make actual GraphQL cost observable in tests or a documented manual measurement.

**Non-Goals:**

- Alter GitHub's branch-protection or mergeability decisions.
- Suppress checks on GitHub or in `gh pr checks`.
- Support regular expressions in the first version.
- Guarantee complete matching beyond GitHub's 100-context-per-commit GraphQL limit.

## Decisions

### Use slash-delimited glob identities

Patterns use shell-style glob semantics, with `*` allowed to span slash-delimited components, and match the same identity rendered to users: `creator/workflow/check` for check runs and `creator/context` for legacy statuses. Pending suites use their available workflow or app identity. This makes configuration readable and allows both `fullsend-ai-*/fullsend/dispatch*` and `*fullsend/dispatch*` to behave directly. Regex was rejected as harder to read and validate for this use case.

### Compile and centralize matching at configuration load

Configuration validation rejects malformed patterns. A shared helper constructs identities and answers whether a check is ignored, preventing the list and detail views from drifting.

### Use two PR-list query shapes

The current lightweight query remains the only query when the ignore list is empty. When patterns exist, an enriched list query requests individual status contexts for the last commit and computes the displayed rollup locally. GitHub GraphQL does not provide a server-side name exclusion filter, so an accurate local result requires identities and states.

The opt-in query requests at most 100 contexts, matching the existing detail-view ceiling. Fetching checks in separate per-PR requests was rejected because it creates an N+1 request pattern and greater secondary-limit risk. Always enriching the existing list query was rejected because it would charge every user for an unused feature.

### Derive rollup from non-ignored checks

Local precedence is failure/error, then pending/expected, then success, then unknown when no included checks remain. This mirrors the states consumed by the existing row renderer. Detail categories, summary counts, pending suites, and missing required-check entries all apply the same ignore predicate.

### Measure rather than rely only on the estimate

Implementation adds a repeatable manual measurement using GraphQL's `rateLimit.cost`, comparing the existing and opt-in query shapes at representative limits. The result is documented before considering the feature ready. A default page of 20 PRs refreshed twice hourly would consume roughly 0–2 additional primary points per section per hour if the estimated delta is 0–1 point per query, versus a typical 5,000-point user limit. Multiple configured sections multiply that amount.

On 2026-08-30, representative authenticated queries over 20 authored PRs reported a cost of 1 point both with the aggregate status alone and with up to 100 status contexts per PR. The status-only response grew from roughly 2 KB to 43 KB. This supports the primary-quota estimate while confirming that payload size and latency are the practical trade-offs.

## Risks / Trade-offs

- [Nested contexts increase payload size and latency] → Opt in only, cap at 100 contexts, test representative response behavior, and retain an easy rollback branch.
- [A PR with more than 100 contexts can hide a matching result outside the fetched page] → Document the ceiling and avoid claiming a filtered state is authoritative beyond it.
- [GitHub can change its cost formula] → Record observed `rateLimit.cost` rather than treating the estimate as a guarantee.
- [Displayed identities can omit unavailable components] → Construct identities only from non-empty components and test each check type.

## Migration Plan

The configuration is additive and empty by default. Rollback consists of removing the configuration entry or abandoning/reverting the feature branch; existing configuration files remain valid.
