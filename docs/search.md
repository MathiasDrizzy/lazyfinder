# Search

`/` (or `ctrl+f`, or a click on the lens in the toolbar) opens the search bar on the bottom row. Results appear
while you type. Design: [ADR 0009](adr/0009-finder-and-explorer-parity.md).

| Key | Does |
| --- | --- |
| type | Edit the query; the list shows the matches |
| `Tab` | Switch between this folder and its subfolders |
| `↑` `↓` `PgUp` `PgDn` | Move through the results while typing |
| `Enter` | Keep the results and give the keyboard back to the list (`/` edits again) |
| `Esc` | Clear the search and bring the folder back |
| `Ctrl+U` | Clear the query |

Clicking a chip adds its filter to the query, clicking it again removes it; clicking `[This folder]` or
`[Subfolders]` switches the scope.

## Query

Words are separated by spaces and must all match the name, ignoring case. `*` and `?` make a word a glob
(`*.go`); quotes keep a phrase together (`"my notes"`).

| Filter | Examples |
| --- | --- |
| `kind:` | `kind:image`, `text`, `audio`, `video`, `archive`, `document`, `folder`, `file`, or any extension (`kind:go`) |
| `ext:` | `ext:go`, `ext:.md` |
| `size:` | `size:>10MB`, `size:<=1.5KB`, `size:5` (units B, KB, MB, GB, TB; 1 KB = 1024 bytes; folders never match) |
| `modified:` and `created:` | `today`, `yesterday`, `week`, `month`, `year`, `2026-10-13`, `>2026-10-01`, `<2026-01-01` |
| `name:` | `name:report` (same as a plain word) |

`week`, `month` and `year` count back from today. Example: `report kind:document modified:month size:<5MB`.

A subfolder search runs in the background, skips hidden files unless they are shown, never follows symbolic links
into folders, and stops at `search.max_results` (default 5000), saying so in the bar. Each result shows where it
lives after its name.
