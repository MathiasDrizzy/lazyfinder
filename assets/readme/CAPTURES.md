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
- writes a `config.json` in that `HOME` that opens `.md`, `.txt` and `.csv` files with `less`, so Enter and Open With
  run inside the recording;
- puts stand-ins for `open` and `qlmanage` first on `PATH`, so nothing the recording opens starts a real
  application;
- makes `Projects/website` a git repository with one commit, then a staged change (`README.md`), a modified file
  (`style.css`) and a new one (`src/menu.js`);
- fixes the dates of every file (created first, then modified), so the Date Modified and Date Created columns are the
  same whichever day you record. Files the GIFs create (in `feature-files.gif`) carry the day of the recording;
- turns off lazygit's first-run popups in that `HOME`, so `L` in `feature-git.gif` shows lazygit itself.

All tapes share `tapes/lib/common.tape`: Catppuccin Mocha, JetBrainsMono Nerd Font (install it first: the plain
"JetBrains Mono" shows boxes instead of icons), font size 16, the same typing speed and frame rate. Each tape sets its
own size: 1200x700 for the window, 520x700 for the sidebar mode.

VHS has no mouse command, and the Git tab of the sidebar opens with a click. `feature-git.tape` types the click the way
a terminal reports it (`ESC [<0;14;2M` and `ESC [<0;14;2m`, an SGR mouse report), with the escape character written
as is inside the string.

`feature-git.gif` runs the `lazygit` on your `PATH`; install it before recording.

## Image screenshot

VHS draws with xterm.js, which does not draw Kitty graphics, so the image preview is a screenshot of a real terminal
window: Ghostty, 120x35, Catppuccin Mocha, opaque background. To redo it:

1. Build lazyfinder: `go build -o /tmp/lazyfinder-img ./cmd/lazyfinder`.
2. Open Ghostty with a temporary `HOME` that holds a copy of `demo-home/` and `LANG=en_US.UTF-8`, and run
   `assets/readme/images-start.sh /tmp/lazyfinder-img` in it. The script copies the pictures to
   `/tmp/lazyfinder-demo/home/Pictures`, the folder the GIFs use, fixes their dates and starts lazyfinder there.
3. Press `↓` and `Space`: the popup shows `Sunset.png`.
4. Take a screenshot of the window and save it as `feature-images.png`.
