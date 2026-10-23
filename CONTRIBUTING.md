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

CI runs `gofmt`, `go vet`, `go test -race` and `goreleaser check` on every push to `main` and `dev` and on every pull request.

## Releasing

Releases are built by GoReleaser from a tag:

```bash
git tag v0.2.0
git push origin v0.2.0
```

- The release workflow builds Linux, macOS and Windows binaries (amd64 and arm64), sets the version from the tag and publishes them with `checksums.txt`.
- A tag with a suffix, such as `v0.3.0-rc1`, is published as a pre-release.
