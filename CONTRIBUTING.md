# Contributing to groundtruth

Thanks for your interest in contributing.

## Project values

- Developer-first
- Local-first
- No surveillance features
- No forced metrics or scoring
- Transparency over automation

## Contribution guidelines

- Keep changes focused and small
- Follow Conventional Commits
- Avoid adding metrics that cannot be disabled
- No features that enable monitoring or ranking developers

## Code style

- Prefer clarity over cleverness
- Keep layers separated (git → classify → metrics → report)

## Testing

Run the full suite before opening a pull request:

```bash
go vet ./...
go test -race ./...
```

- Tests that need a repository build a real one in a temp directory with go-git (see `internal/git/testrepo_test.go`), so no fixtures are checked in and your own repositories are never touched.
- The linked-worktree test uses the `git` CLI and is skipped when it is not installed.
- New metrics, report formats and config rules should come with tests in the same package.
