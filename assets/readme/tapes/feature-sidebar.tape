# Sidebar mode: a narrow tree next to your editor that follows the project by itself: a file
# another program writes appears at once.
Output assets/readme/feature-sidebar.gif
Set Width 520
Set Height 700
Source assets/readme/tapes/lib/common.tape
Type "(sleep 6; echo 'Ideas for the menu' > Projects/website/notes.md) &" Enter
Type "clear" Enter
Type "lazyfinder --sidebar Projects/website" Enter
Wait+Screen@20s /style/
Show
Sleep 1.2s

Enter
Sleep 1.2s
Down
Sleep 400ms
Down
Sleep 400ms
Down
Sleep 400ms
Space
Sleep 2s
Space
Sleep 2.5s
Up
Sleep 400ms
Up
Sleep 400ms
Up
Sleep 400ms
Enter
Sleep 1.5s

Type "q"
