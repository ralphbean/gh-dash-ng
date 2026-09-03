# Agent Conventions for gh-dash-ng

This is a Go TUI application forked from [dlvhdr/gh-dash](https://github.com/dlvhdr/gh-dash). It is a GitHub CLI extension (`gh dash`) built with [Bubbletea](https://github.com/charmbracelet/bubbletea), [Lipgloss](https://github.com/charmbracelet/lipgloss), and [Cobra](https://github.com/spf13/cobra).

## Build

```sh
go build .
```

Or via task runner:

```sh
task build
```

## Test

```sh
go test ./...
```

The repo uses [prism](https://go.dalton.dog/prism) as a test runner, but `go test ./...` works for standard test execution.

## Lint

```sh
golangci-lint run --path-mode=abs --config=".golangci.yml" --timeout=5m
```

If `golangci-lint` is not available in the environment, install it first:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.10.1
```

Or via task runner:

```sh
task lint
```

To auto-fix lint issues:

```sh
task lint:fix
```

The project uses golangci-lint v2 with config in `.golangci.yml`. Key enabled linters: staticcheck, misspell, bodyclose, whitespace, tparallel. Disabled linters: errcheck, ineffassign, unused.

## Formatting

Formatters are configured in `.golangci.yml` under the `formatters` section:

- **gofumpt** — stricter gofmt
- **goimports** — import grouping
- **golines** — line length enforcement (uses `--chain-split-dots`)

Run formatting standalone:

```sh
gofumpt -w $(git ls-files '*.go')
```

If `gofumpt` is not available in the environment, install it first:

```sh
go install mvdan.cc/gofumpt@v0.8.0
```

Or via task runner:

```sh
task fmt
```

## Commit Conventions

Use conventional commits with the issue number as scope:

```
fix(#N): short description
feat(#N): short description
docs(#N): short description
chore: short description
```

## Development Environment

The project uses [Devbox](https://github.com/jetpack-io/devbox) to manage tooling. See `devbox.json` for pinned versions. Key tools: Go, golangci-lint, gofumpt, go-task.

## Project Structure

```
cmd/           CLI entrypoint (cobra commands)
internal/
  config/      User config parsing (config.yml)
  data/        GitHub GraphQL API data fetching
  git/         Git operations
  shell/       Shell command execution
  tui/         TUI rendering (bubbletea components)
    components/  UI components (sections, views, rows, sidebar)
    context/     App context and styles
    keys/        Keybinding definitions
    markdown/    Markdown rendering
    theme/       Theme configuration
docs/          Documentation site (Astro)
testdata/      Test fixtures
```

## New Configuration Options Checklist

When adding a new column or configuration option to `PrsLayoutConfig` or `IssuesLayoutConfig` in `internal/config/parser.go`:

1. Add the field to the config struct in `internal/config/parser.go` and update the default layout
2. Add the column rendering logic in the appropriate row component (`internal/tui/components/prrow/prrow.go` or `internal/tui/components/issuerow/issuerow.go`)
3. Add the column definition in the section component (`internal/tui/components/prssection/prssection.go` or `internal/tui/components/issuessection/issuessection.go`)
4. Update the layout documentation in `docs/src/content/docs/configuration/layout/pr.mdx` and/or `issue.mdx` — add to the default columns list, YAML example, and a dedicated section
5. Update the JSON schema in `docs/src/pages/schema/layout/pr.json.ts` and/or `issue.json.ts` — add the property following existing patterns
6. Add unit tests for the new rendering function
