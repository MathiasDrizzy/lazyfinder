# Open-with execution model

* Status: accepted (2026-10-13)
* Date: 2026-10-13

## Context and problem statement

Opening a file means starting a process with a user-controlled file name. A shell, a placeholder expanded inside a
string, or running an executable would turn file names into code. The order says: "nunca ejecutar nada sin que el
usuario lo haya configurado".

## Decision outcome

* Processes start with `exec.Command(argv[0], args...)`, never through `sh -c`.
* `{file}` is substituted as one complete argv element, so spaces, quotes, `;` and `$(...)` stay data. A file name
  starting with `-` is passed as `./name` (or after `--` when the opener allows) to avoid option injection.
* Only three sources may start a process: an opener defined in the user's `config.json`, the built-in default
  `open -- <file>` (macOS), and Quick Look (`qlmanage -p -- <file>`, a fixed argv started only by the explicit `Q` key). Executable bits are ignored: lazyfinder never runs the file it is showing; `.app`
  bundles and scripts open with `open`, which asks LaunchServices, not the shell.
* `mode: terminal` suspends the TUI (Bubble Tea `ExecProcess`), restores the terminal after.
* A missing opener binary is an error in the status line, with no fallback to another program.
* Tests inject `opener.Runner`; the integration test records argv in a stub script inside a temp dir.

### Consequences

* Good: the injection class is closed by construction and tested with hostile file names.
* Bad: `open -a "App"` openers depend on app names; documented in `docs/configuration.md`.
