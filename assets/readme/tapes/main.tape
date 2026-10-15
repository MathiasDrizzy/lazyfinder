# Main GIF, a short tour: enter a folder, look inside files with Space, open a text file in micro and come back,
# switch to the column view, and change the theme in Settings.
Output assets/readme/main.gif
Set Width 1200
Set Height 640
Source assets/readme/tapes/lib/common.tape
# no padding: VHS draws it in the tape theme, which would frame the window in Mocha after the theme changes
Set Padding 0
Type "lazyfinder" Enter
Wait+Screen@20s /Favorites/
Show
Sleep 1.2s

Down
Sleep 500ms
Enter
Sleep 1.2s
Down
Sleep 300ms
Down
Sleep 600ms
Space
Sleep 2.2s
Down
Sleep 2s
Space
Sleep 600ms

Down
Sleep 400ms
Enter
Sleep 2.5s
Ctrl+Q
Sleep 1.2s

Type "M"
Sleep 1.2s
Up
Sleep 700ms
Up
Sleep 700ms
Up
Sleep 1.5s
Type "M"
Sleep 800ms

Type ","
Sleep 1.2s
Right
Sleep 1.3s
Right
Sleep 1.3s
Right
Sleep 1.3s
Escape
Sleep 1.8s

Type "q"
