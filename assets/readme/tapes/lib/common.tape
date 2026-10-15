# Shared preamble of every README GIF: same theme, font, speed and the isolated demo
# environment (assets/readme/demo-env.sh). Each tape sets its own Width and Height
# before it, then starts lazyfinder the way its feature needs.
# Use it with: Source assets/readme/tapes/lib/common.tape
Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 16
Set Padding 14
Set Theme "Catppuccin Mocha"
# a window with its title bar, so each GIF reads as a terminal on a desktop
Set WindowBar Colorful
Set BorderRadius 10
Set Margin 18
Set MarginFill "#45475a"
Set TypingSpeed 60ms
Set Framerate 25

Hide
Type "source assets/readme/demo-env.sh" Enter
Wait@60s
