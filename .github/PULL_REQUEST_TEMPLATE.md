## What and why

<!-- What changes for someone using lazyfinder, and why. Link the issue if there is one. -->

## Checks

- [ ] The test came first and failed before the change
- [ ] `go vet ./...` and `go test -race ./...` pass
- [ ] Golden files updated on purpose (`go test ./internal/ui -update`) and the diff reviewed, if the screen changed
- [ ] `CHANGELOG.md` has a line under `[Unreleased]` for a change users notice
