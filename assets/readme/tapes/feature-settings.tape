# Settings and themes: "," opens the settings; themes and languages change at once.
Output assets/readme/feature-settings.gif
Set Width 1200
Set Height 640
Source assets/readme/tapes/lib/common.tape
# no padding: VHS draws it in the tape theme, which would frame the window in Mocha after the theme changes
Set Padding 0
Type "lazyfinder" Enter
Wait+Screen@20s /Favorites/
Show
Sleep 1s

Type ","
Sleep 1.2s
Right
Sleep 1.2s
Right
Sleep 1.2s
Right
Sleep 1.2s
Down
Sleep 800ms
Right
Sleep 500ms
Right
Sleep 2.2s
Left
Sleep 500ms
Left
Sleep 1.2s
Escape
Sleep 1.5s

Type "q"
