# How the README media is made

| File | Made with | Source |
|---|---|---|
| `main.gif` and every `feature-*.gif` | Real [Ghostty](https://ghostty.org) windows, recorded by `ghostty/record.py` | `ghostty/scenes/*.json` |
| `feature-images.png` | A real Ghostty window | the steps below |

## GIFs

Run from the repository root, on macOS, with Ghostty, ffmpeg, uv, micro, tmux and lazygit installed, and the
JetBrainsMono Nerd Font (the plain "JetBrains Mono" shows boxes instead of icons):

```sh
uv run --with pyobjc-framework-Quartz --with pyobjc-framework-Cocoa python assets/readme/ghostty/record.py [scene ...]
```

Without a scene name it records them all, one after another (they share the demo folder). For each scene:

1. `demo-env.sh` prepares a fresh demo environment (below).
2. `record.py` opens a new Ghostty window (JetBrainsMono Nerd Font 16, Catppuccin Mocha, 118x32 cells, a fixed title)
   whose command is `ghostty/driver.py`. The window takes the focus while it records.
3. `driver.py` runs the scene's program in a pseudo-terminal of its own and types the scene's keys with fixed pauses.
   Its output goes to Ghostty untouched; from the keyboard only Ghostty's answers to the program's queries (colors,
   image support, sizes) go through, so keys typed by hand never reach the recording.
4. `record.py` captures the window 20 times a second straight from the window server (the window, not the screen, so
   nothing else on the desktop shows), puts it on a plain backdrop and turns the frames into a GIF with ffmpeg.

So the GIFs show lazyfinder exactly as Ghostty draws it: the theme colors reach the whole window, title bar
included, and images are drawn with the Kitty graphics protocol.

`feature-sidebar` and `feature-git` run tmux inside the window: `lazyfinder --sidebar` in a narrow pane on the left
and a shell on the right, the way the sidebar is meant to be used. The keys `ctrl+a` and an arrow move between the
panes. The Git tab opens with a click, which the scene sends the way a terminal reports it (`ESC [<0;14;3M` and
`ESC [<0;14;3m`, one row lower because of the pane title).

`demo-env.sh`:

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

## Image screenshot

The image preview is also a screenshot of a real terminal window: Ghostty, 120x35, Catppuccin Mocha, opaque background. To redo it:

1. Build lazyfinder: `go build -o /tmp/lazyfinder-img ./cmd/lazyfinder`.
2. Open Ghostty with a temporary `HOME` that holds a copy of `demo-home/` and `LANG=en_US.UTF-8`, and run
   `assets/readme/images-start.sh /tmp/lazyfinder-img` in it. The script copies the pictures to
   `/tmp/lazyfinder-demo/home/Pictures`, the folder the GIFs use, fixes their dates and starts lazyfinder there.
3. Press `↓` and `Space`: the popup shows `Sunset.png`.
4. Take a screenshot of the window and save it as `feature-images.png`.
