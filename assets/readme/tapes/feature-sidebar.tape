# Sidebar mode next to a terminal: lazyfinder --sidebar on the left of a tmux window, a shell on the right. What the
# shell changes shows in the tree at once: a new file, its git letter, a staged file, an edit.
Output assets/readme/feature-sidebar.gif
Set Width 1200
Set Height 640
Source assets/readme/tapes/lib/common.tape
Type "cd Projects/website && tmux -f $DEMO_TMP/tmux.conf new-session -s demo 'lazyfinder --sidebar .' \; select-pane -T 'lazyfinder --sidebar' \; split-window -h -l 76% \; select-pane -T terminal" Enter
Wait+Screen@20s /terminal/
Sleep 1.5s
Show
Sleep 1.5s

Type 'echo "Ideas for the menu" > notes.md'
Sleep 400ms
Enter
Sleep 2s
Type "git add notes.md"
Sleep 400ms
Enter
Sleep 2s
Type "micro index.html"
Sleep 400ms
Enter
Sleep 1.2s
Down 10
Sleep 300ms
Enter
Up
Type "  <p>Open every day.</p>"
Sleep 500ms
Ctrl+S
Sleep 400ms
Ctrl+Q
Sleep 2.2s

Ctrl+A
Left
Sleep 1s
Enter
Sleep 1.2s
Down
Sleep 400ms
Down
Sleep 800ms
Space
Sleep 3s
Space
Sleep 1.2s
Type "q"
Sleep 300ms
Hide
Type "exit" Enter
