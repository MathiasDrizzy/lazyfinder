#!/bin/sh
# Starts lazyfinder for the inline-image screenshot (feature-images.png), which needs a real Ghostty window: VHS
# cannot draw Kitty graphics. Run it through a script that gives it an isolated HOME holding a copy of demo-home
# (see CAPTURES.md). It copies the pictures to the same fixed folder as the GIFs, so the path bar matches them, and
# fixes their dates. Usage: images-start.sh <lazyfinder binary>
set -eu
DEMO=/tmp/lazyfinder-demo/home
rm -rf "${DEMO:?}"
mkdir -p "$DEMO"
cp -R "$HOME/Pictures" "$DEMO/"
touch -t 202607200700 "$DEMO/Pictures/Sunset.png"
touch -t 202608031800 "$DEMO/Pictures/Mountains.png"
touch -t 202609010900 "$DEMO/Pictures"
touch -t 202610011800 "$DEMO/Pictures"
cd "$DEMO/Pictures"
exec "$1"
