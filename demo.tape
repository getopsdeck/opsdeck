# OpsDeck Hero GIF — brief + metrics demo

Output demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1100
Set Height 700
Set Theme "TokyoNight"
Set WindowBar Colorful
Set WindowBarSize 40
Set TypingSpeed 35ms
Set Padding 20
Set CursorBlink false
Set LoopOffset 70%
Set Framerate 15

# Setup (hidden)
Hide
Type `export PATH=/Users/jason/Documents/GitHub/OpsDeck:$PATH && clear`
Enter
Sleep 1s
Show

# Daily brief
Type "opsdeck brief --since 12h"
Enter
Sleep 5s

# Clear and show metrics
Type "clear"
Enter
Sleep 500ms
Type "opsdeck metrics"
Enter
Sleep 5s
