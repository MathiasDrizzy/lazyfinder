# Shared preamble of every README GIF: same theme, font, speed and the isolated demo
# environment (assets/readme/demo-env.sh). Each tape sets its own Width and Height
# before it, then starts lazyfinder the way its feature needs.
# Use it with: Source assets/readme/tapes/lib/common.tape
Set Shell "bash"
Set FontFamily "JetBrainsMono Nerd Font"
Set FontSize 16
Set Padding 16
Set Theme "Catppuccin Mocha"
Set TypingSpeed 60ms
Set Framerate 25

Hide
Type "source assets/readme/demo-env.sh" Enter
Wait@60s
