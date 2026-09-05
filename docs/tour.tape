# vhs tape for docs/tour.gif — records the real running swallow TUI.
#
#   go build -o /tmp/swallow ./go/cmd/swallow   (or: make tour)
#   vhs docs/tour.tape                           (run from the repo root)
#
# Menlo carries the box-drawing + block glyphs the wordmark needs; the custom
# theme is Tokyo Night (tokyo-night.terminal) so terminal padding blends with the
# app's own #1a1b26 canvas.

Output docs/tour.gif

Set Shell "bash"
Set FontFamily "Menlo"
Set FontSize 16
Set Width 1120
Set Height 660
Set Padding 24
Set Framerate 30
Set Theme { "name": "Tokyo Night", "background": "#1a1b26", "foreground": "#c0caf5", "selection": "#33467c", "cursor": "#c0caf5", "black": "#15161e", "red": "#f7768e", "green": "#9ece6a", "yellow": "#e0af68", "blue": "#7aa2f7", "magenta": "#bb9af7", "cyan": "#7dcfff", "white": "#a9b1d6", "brightBlack": "#414868", "brightRed": "#f7768e", "brightGreen": "#9ece6a", "brightYellow": "#e0af68", "brightBlue": "#7aa2f7", "brightMagenta": "#bb9af7", "brightCyan": "#7dcfff", "brightWhite": "#c0caf5" }

Hide
Type "/tmp/swallow"
Enter
Sleep 2.5s
Show

Sleep 2.5s
Down
Sleep 1.1s
Down
Sleep 1.1s
Down
Sleep 1.1s
Down
Sleep 1.1s
Down
Sleep 1.1s
Down
Sleep 1.6s
Type "g"
Sleep 2.2s
Type "q"
Sleep 500ms
