# Git next to a terminal: the Git tab of the sidebar (a quick look, it changes nothing) on the left of a tmux window,
# git commands on the right. The tab follows what the commands do; Enter shows the diff of a file.
# The Git tab opens with a click: VHS has no mouse command, so the tape types the terminal's mouse report
# (ESC [<0;x;y M, then m to release), here at the tab's icon below the pane title.
Output assets/readme/feature-git.gif
Set Width 1200
Set Height 640
Source assets/readme/tapes/lib/common.tape
Type "cd Projects/website && tmux -f $DEMO_TMP/tmux.conf new-session -s demo 'lazyfinder --sidebar .' \; select-pane -T 'lazyfinder --sidebar' \; split-window -h -l 62% \; select-pane -T terminal" Enter
Wait+Screen@20s /terminal/
Sleep 1s
Type@0ms "[<0;14;3M[<0;14;3m"
Wait+Screen@10s /Staged/
Ctrl+A
Right
Sleep 1s
Show
Sleep 2s

Type "git add style.css"
Sleep 400ms
Enter
Sleep 2.2s
Ctrl+A
Left
Sleep 1s
Down
Sleep 500ms
Down
Sleep 800ms
Enter
Sleep 3.5s
Escape
Sleep 1s
Ctrl+A
Right
Sleep 600ms
Type 'git commit -m "Add the footer color"'
Sleep 400ms
Enter
Sleep 3s
Ctrl+A
Left
Type "q"
Sleep 300ms
Hide
Type "exit" Enter
