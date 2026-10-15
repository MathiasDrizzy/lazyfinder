# How the README media is made

| File | Made with | Source |
|---|---|---|
| `main.gif` and every `feature-*.gif` | [VHS](https://github.com/charmbracelet/vhs) | `tapes/*.tape` |
| `feature-images.png` | A real Ghostty window | the steps below |

## GIFs

Run from the repository root:

```sh
for t in assets/readme/tapes/*.tape; do vhs "$t"; done
```

Run them one after another, not in parallel: they share the demo folder. Each tape sources `demo-env.sh`, which:

- builds lazyfinder into `/tmp/lazyfinder-demo/.bin` and puts it first on `PATH`;
- uses `/tmp/lazyfinder-demo/home` as `HOME`, with a copy of `demo-home/` (invented files, nothing personal), so
  recording never touches your files or writes inside the repository. The folder is fixed, not random, because the
  path bar shows it;
- writes a `config.json` in that `HOME` with compact rows that opens `.md`, `.txt` and `.csv` files with
  [micro](https://micro-editor.github.io), so Enter and Open With run inside the recording. micro gets the Catppuccin
  Mocha colors from `micro/catppuccin-mocha.micro` (MIT, from [catppuccin/micro](https://github.com/catppuccin/micro))
  and a status bar that says how to go back;
- puts stand-ins for `open`, `qlmanage`, `osascript` and `xdg-open` first on `PATH`, so nothing the recording opens
  starts a real application;
- writes a `tmux.conf` for the split-window GIFs (`feature-sidebar` and `feature-git`): two panes with a title each,
  prefix `ctrl+a` (so `ctrl+b` reaches lazyfinder), the mouse on, and 24-bit color (`Tc`). `TMUX_TMPDIR` keeps the
  tmux socket in the demo folder, so no other tmux is touched and the `tmux info` that lazyfinder runs to check for
  24-bit color finds this one;
- makes `Projects/website` a git repository with one commit, then a staged change (`README.md`), a modified file
  (`style.css`) and a new one (`src/menu.js`);
- fixes the dates of every file (created first, then modified), so the Date Modified and Date Created columns are the
  same whichever day you record. Files the GIFs create (in `feature-files.gif`) carry the day of the recording;
- turns off lazygit's first-run popups in that `HOME`, so `L` shows lazygit itself.

All tapes share `tapes/lib/common.tape`: Catppuccin Mocha, JetBrainsMono Nerd Font (install it first: the plain
"JetBrains Mono" shows boxes instead of icons), font size 16, a window bar with rounded corners, the same typing
speed and frame rate. Every tape is 1200x640. The tapes that change the theme (`main`, `feature-settings`) set the
padding to 0: VHS paints the padding in the tape's theme, which would frame the window in Mocha after the change.

The sidebar is recorded inside tmux, next to a shell, the way it is meant to be used. `micro`, `tmux` and `lazygit`
must be installed before recording.

VHS has no mouse command, and the Git tab of the sidebar opens with a click. `feature-git.tape` types the click the way
a terminal reports it (`ESC [<0;14;3M` and `ESC [<0;14;3m`, an SGR mouse report, one row lower than without tmux
because of the pane title), with the escape character written as is inside the string.

## Image screenshot

VHS draws with xterm.js, which does not draw Kitty graphics, so the image preview is a screenshot of a real terminal
window: Ghostty, 120x35, Catppuccin Mocha, opaque background. To redo it:

1. Build lazyfinder: `go build -o /tmp/lazyfinder-img ./cmd/lazyfinder`.
2. Open Ghostty with a temporary `HOME` that holds a copy of `demo-home/` and `LANG=en_US.UTF-8`, and run
   `assets/readme/images-start.sh /tmp/lazyfinder-img` in it. The script copies the pictures to
   `/tmp/lazyfinder-demo/home/Pictures`, the folder the GIFs use, fixes their dates and starts lazyfinder there.
3. Press `↓` and `Space`: the popup shows `Sunset.png`.
4. Take a screenshot of the window and save it as `feature-images.png`.
