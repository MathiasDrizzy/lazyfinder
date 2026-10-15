# Search: "/" finds files while you type; Tab looks in the subfolders too.
Output assets/readme/feature-search.gif
Set Width 1200
Set Height 640
Source assets/readme/tapes/lib/common.tape
Type "lazyfinder" Enter
Wait+Screen@20s /Favorites/
Show
Sleep 1s

Type "/"
Sleep 800ms
Type "md"
Sleep 1.5s
Tab
Sleep 2s
Ctrl+U
Sleep 600ms
Type "kind:image"
Sleep 2s
Enter
Sleep 600ms
Space
Sleep 1.5s
Space
Sleep 400ms
Escape
Sleep 1s

Type "q"
