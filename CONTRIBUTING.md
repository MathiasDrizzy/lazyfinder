# Contributing

Thanks for looking. lazyfinder is small on purpose.

## Setup

```sh
git clone https://github.com/MathiasDrizzy/lazyfinder
cd lazyfinder
go build -o bin/lazyfinder ./cmd/lazyfinder
```

## Before you open a pull request

- Write the test first and watch it fail. Bug fixes ship with the test that failed before.
- `go vet ./...` and `go test -race ./...` must pass. `scripts/tuiqa.sh` runs the end-to-end scenarios in
  `testdata/tuiqa` against the real binary in a pty (it needs the tuiqa runner, see the script).
- Screens are checked against golden files in `internal/ui/testdata/golden`. After a visual change run
  `go test ./internal/ui -update` and review the diff.
- Tests never touch the real home folder: they use temporary directories.
- Commits follow [Conventional Commits](https://www.conventionalcommits.org/): `feat(ui): add a column`.
- Design decisions live in [docs/adr](docs/adr); add one when you change a decision.
