# How the README media is made

| File | Made with | Source |
|---|---|---|
| `main.gif`, every `feature-*.gif` and `feature-images.png` | Real [Ghostty](https://ghostty.org) windows, recorded by `ghostty/record.py` | `ghostty/scenes/*.json` |

## GIFs and the image screenshot

Run from the repository root, on macOS, with Ghostty, ffmpeg, uv, micro, [herdr](https://herdr.dev) and lazygit installed, and the
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
   nothing else on the desktop shows), cuts 14 points off each side (the rounded corners) and turns the frames into a
   GIF with ffmpeg. The window has no title bar and a wider padding, so the GIF is the terminal from edge to edge.

So the GIFs show lazyfinder exactly as Ghostty draws it, and images are drawn with the Kitty graphics protocol.

`main.gif` is a tour of both modes in one herdr window: the full window (preview, an editor, columns, search with an
image, tabs, go to a folder, themes, and the shell left in that folder on quit), then the sidebar of the lazyfinder
herdr plugin, opened with its key, next to the shell.
Its scene sets `"speed": 1.25`, so the GIF plays faster than it was recorded.

`feature-sidebar` and `feature-git` run herdr inside the window, and the lazyfinder plugin docks the sidebar next to
the shell by itself: on the left in `feature-sidebar`, on the right in `feature-git` (each scene sets `herdr.dock` in
its `"config"`). `ctrl+a` and `h` or `l` move between the panes; the scenes send `h` and `l` in the Kitty keyboard
form (`ESC [104u`, `ESC [108u`), because herdr turns that protocol on in Ghostty. The Git tab opens with a click,
which the scene sends the way a terminal reports it (`ESC [<0;91;3M` and `ESC [<0;91;3m`): the Ghostty window ends
up 124x34 cells, so the right sidebar starts at column 78 and the Git icon is its 13th cell.

`feature-images.png` is the last frame of its scene: herdr, lazyfinder in `Pictures`, `↓` and `Space` on
`Sunset.png`. herdr passes the Kitty graphics on to Ghostty, so the picture is drawn as in a plain terminal.

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
- links the lazyfinder herdr plugin of the repository (`scripts/herdr-build.sh` builds its binary) and writes a herdr
  `config.toml` in that `HOME`: bash without a login, prefix `ctrl+a` (so `ctrl+b` reaches lazyfinder), `prefix+e`
  for the plugin's toggle, Catppuccin, no workspace sidebar, no tab bar with one tab and no update checks. It runs
  `herdr config check`, because herdr shows a warning for an unknown key. herdr keeps its socket in that `HOME`, so
  no other herdr is touched, and `record.py` stops the demo's server after each scene;
- makes `Projects/website` a git repository with one commit, then a staged change (`README.md`), a modified file
  (`style.css`) and a new one (`src/menu.js`);
- fixes the dates of every file (created first, then modified), so the Date Modified and Date Created columns are the
  same whichever day you record. Files the GIFs create (in `feature-files.gif`) carry the day of the recording;
- turns off lazygit's first-run popups in that `HOME`, so `L` shows lazygit itself.
