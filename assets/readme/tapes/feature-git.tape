# Git in the sidebar: the git letters in the tree, the Git tab (a quick look, it changes nothing),
# the diff of a file, and L for lazygit. The Git tab opens with a click: VHS has no mouse command, so
# the tape types the terminal's mouse report (ESC [<0;x;y M, then m to release).
Output assets/readme/feature-git.gif
Set Width 520
Set Height 700
Source assets/readme/tapes/lib/common.tape
Type "lazyfinder --sidebar Projects/website" Enter
Wait+Screen@20s /style/
Show
Sleep 2s

Type@0ms "[<0;14;2M[<0;14;2m"
Sleep 2s
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 800ms
Enter
Sleep 3s
Escape
Sleep 800ms
Type "L"
Sleep 3s
Type "q"
Sleep 1.2s

Type "q"
