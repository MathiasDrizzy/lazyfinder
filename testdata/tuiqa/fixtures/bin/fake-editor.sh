#!/bin/sh
# Stands in for a terminal editor: shows what it was asked to open and waits for a key.
printf 'FAKE_EDITOR_OPENED:%s\n' "$1"
printf 'press any key\n'
IFS= read -r _ || true
