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
sh scripts/herdr-build.sh || return 1 # bin/lazyfinder, which the herdr plugin runs
LAZYFINDER_REPO="$PWD"
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
# herdr for the GIFs of the sidebar and the Git tab, with the lazyfinder plugin of this checkout (herdr-plugin.toml).
# Its config, socket and plugin list live in the demo HOME: the herdr the recording runs never meets another one. The
# workspace sidebar of herdr starts hidden and the tab row only shows with two tabs, so the GIF is lazyfinder and the
# terminal. The prefix is ctrl+a; prefix+e is the plugin's toggle action.
mkdir -p "$XDG_CONFIG_HOME/herdr"
cat >"$XDG_CONFIG_HOME/herdr/config.toml" <<'TOML'
onboarding = false

[terminal]
default_shell = "/bin/bash"
shell_mode = "non_login"

[keys]
prefix = "ctrl+a"

[[keys.command]]
key = "prefix+e"
type = "plugin_action"
command = "lazyfinder.toggle"
description = "toggle lazyfinder"

[theme]
name = "catppuccin"

[ui]
sidebar_start_collapsed = true
sidebar_collapsed_mode = "hidden"
hide_tab_bar_when_single_tab = true
prompt_new_tab_name = false
window_title = "website"

[update]
version_check = false
manifest_check = false
TOML
# the shell of the herdr panes: the prompt of the recordings, no "default shell is now zsh" notice
cat >"$HOME/.bashrc" <<'BASH'
export BASH_SILENCE_DEPRECATION_WARNING=1
PS1='\[\e[38;2;137;180;250m\]\W\[\e[0m\] $ '
BASH
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

# an unknown key would show a warning in the recording and fall back to the user's shell
herdr config check >/dev/null 2>&1 || { herdr config check >&2; return 1; }
# The plugin is linked with a server of the demo HOME that stops right after: the herdr of the recording starts its own
# and finds the plugin in its list.
(
  unset HERDR_SOCKET_PATH HERDR_ENV HERDR_PANE_ID HERDR_SESSION HERDR_WORKSPACE_ID HERDR_TAB_ID HERDR_PLUGIN_ID
  herdr server >/dev/null 2>&1 &
  for _ in $(seq 50); do [ -S "$XDG_CONFIG_HOME/herdr/herdr.sock" ] && break; sleep 0.1; done
  linked=0
  herdr plugin link "$LAZYFINDER_REPO" >/dev/null && linked=1
  herdr server stop >/dev/null
  [ "$linked" = 1 ]
) || return 1

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
unset LAZYFINDER_REPO
cd "$HOME" || return 1
clear
