# shellcheck shell=bash
# Isolated demo environment for recording the README GIFs. Use it with `source`
# from the repository root (ghostty/record.py does it for you):
#   - builds lazyfinder into a temporary folder and puts it on PATH;
#   - uses a temporary HOME with a copy of assets/readme/demo-home (invented files,
#     nothing personal), so recording never touches real files or writes inside the
#     repository;
#   - makes Projects/website a git repository with a commit and a few changes;
#   - fixes the dates of every file, so the columns always show the same dates;
#   - puts a fake `open`, `qlmanage`, `osascript` and `xdg-open` first on PATH, so nothing the recording opens
#     starts a real application.
# a fixed, neutral folder (not a random one in the system temp dir): the path bar shows it
DEMO_TMP=/tmp/lazyfinder-demo
rm -rf "${DEMO_TMP:?}"
mkdir -p "$DEMO_TMP/.bin"
go build -o "$DEMO_TMP/.bin/lazyfinder" ./cmd/lazyfinder || return 1 # before changing HOME: Go uses its cache there
export HOME="$DEMO_TMP/home"
mkdir -p "$HOME"
cp -R assets/readme/demo-home/. "$HOME/"
export XDG_CONFIG_HOME="$HOME/.config" XDG_DATA_HOME="$HOME/.local/share" XDG_STATE_HOME="$HOME/.local/state"
unset LAZYFINDER_CONFIG
mkdir -p "$XDG_CONFIG_HOME/lazyfinder"
# Enter on a text file opens it in micro inside the recording; Open With lists it too. Compact rows fit more files.
cat >"$XDG_CONFIG_HOME/lazyfinder/config.json" <<'JSON'
{
  "version": 1,
  "density": "compact",
  "openers": {
    "micro": { "argv": ["micro", "{file}"], "mode": "terminal" }
  },
  "open_with": { "md": "micro", "txt": "micro", "csv": "micro" }
}
JSON
# micro with the Catppuccin Mocha colors of the recordings (assets/readme/micro, MIT, from catppuccin/micro).
mkdir -p "$XDG_CONFIG_HOME/micro/colorschemes"
cp assets/readme/micro/catppuccin-mocha.micro "$XDG_CONFIG_HOME/micro/colorschemes/"
cat >"$XDG_CONFIG_HOME/micro/settings.json" <<'JSON'
{
  "colorscheme": "catppuccin-mocha",
  "ruler": false,
  "statusformatl": "$(filename) $(modified)",
  "statusformatr": "ctrl+q: back to lazyfinder"
}
JSON
export MICRO_TRUECOLOR=1 COLORTERM=truecolor
# tmux for the sidebar GIF: two panes side by side with a title each, Catppuccin borders, no status bar. The prefix
# is ctrl+a, so ctrl+b reaches lazyfinder. Its socket lives in the demo folder: no other tmux is touched, and the
# `tmux info` that colorprofile runs to see whether tmux passes 24-bit color finds this server.
export TMUX_TMPDIR="$DEMO_TMP"
cat >"$DEMO_TMP/tmux.conf" <<'TMUX'
set -g prefix C-a
unbind C-b
set -g escape-time 0
set -g status off
# clicks reach the pane under them (the Git tab of the sidebar opens with a click)
set -g mouse on
set -g default-terminal "tmux-256color"
set -as terminal-features ",*:RGB"
set -ga terminal-overrides ",*:Tc"
set -g default-command "bash --norc --noprofile"
set -g pane-border-status top
set -g pane-border-format " #{pane_title} "
# titles are set by the tape (the default one is the host name)
set -g allow-set-title off
set -g pane-border-style "fg=#45475a"
set -g pane-active-border-style "fg=#89b4fa"
# the window title (Ghostty shows it) and no "default shell is now zsh" notice from macOS's bash
set -g set-titles on
set -g set-titles-string "website"
set-environment -g BASH_SILENCE_DEPRECATION_WARNING 1
set-environment -g PS1 '\[\e[38;2;137;180;250m\]website\[\e[0m\] $ '
TMUX
# lazygit without its first-run popups (the welcome and the notes of a new version).
lg_version="$(lazygit --version 2>/dev/null | sed -n 's/.*[ ,]version=\([^,]*\),.*/\1/p')"
for d in "$XDG_CONFIG_HOME/lazygit" "$XDG_STATE_HOME/lazygit"; do
  mkdir -p "$d" && printf 'startuppopupversion: 99\nlastversion: %s\n' "$lg_version" >"$d/state.yml"
done
unset lg_version
# Stand-ins for the programs that would start a real application.
for p in open qlmanage osascript xdg-open; do printf '#!/bin/sh\nexit 0\n' >"$DEMO_TMP/.bin/$p"; done
chmod +x "$DEMO_TMP/.bin/"*
export PATH="$DEMO_TMP/.bin:$PATH"
export LANG=en_US.UTF-8 BASH_SILENCE_DEPRECATION_WARNING=1
unset LC_ALL LC_MESSAGES

# Projects/website: one commit, then a changed file, a new one and a staged one.
export GIT_CONFIG_NOSYSTEM=1 GIT_AUTHOR_NAME=Demo GIT_AUTHOR_EMAIL=demo@example.com GIT_COMMITTER_NAME=Demo GIT_COMMITTER_EMAIL=demo@example.com
export GIT_AUTHOR_DATE=2026-09-25T10:00:00 GIT_COMMITTER_DATE=2026-09-25T10:00:00
(
  cd "$HOME/Projects/website" || exit 1
  git init -q -b main
  git add README.md index.html style.css src/app.js
  git commit -q -m "Add the landing page"
  printf '\nfooter {\n  color: #888;\n}\n' >>style.css
  printf '\nOpening hours: 7:00 to 19:00.\n' >>README.md
  git add README.md
) || return 1
unset GIT_AUTHOR_DATE GIT_COMMITTER_DATE

# Fixed dates (YYYYMMDDhhmm): created first, then modified, so Date Created and Date Modified differ.
set_time() { touch -t "$1" "$HOME/$3" && touch -t "$2" "$HOME/$3"; }
set_time 202609011000 202609281500 "Documents/Meeting notes.md"
set_time 202608150900 202609301100 "Documents/Reading list.md"
set_time 202609100800 202609271800 "Documents/Trip to Lisbon.txt"
set_time 202601051200 202609300930 "Documents/Budget 2026.csv"
set_time 202609291000 202609291000 "Documents/Reports/Q3 summary.md"
set_time 202607200700 202607200700 "Pictures/Sunset.png"
set_time 202608031800 202608031800 "Pictures/Mountains.png"
set_time 202609201000 202609201000 "Downloads/Brand kit.zip"
set_time 202609251000 202610011700 "Projects/website/style.css"
set_time 202609251000 202610011650 "Projects/website/README.md"
set_time 202609251000 202609251000 "Projects/website/index.html"
set_time 202609251000 202609251000 "Projects/website/src/app.js"
set_time 202610011720 202610011720 "Projects/website/src/menu.js"
for d in Documents/Reports Documents Pictures Downloads Desktop Projects/website/src Projects/website Projects; do
  set_time 202609010900 202610011800 "$d"
done
unset -f set_time
cd "$HOME" || return 1
clear
