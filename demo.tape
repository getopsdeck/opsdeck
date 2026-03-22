# OpsDeck Hero GIF — full feature demo

Output demo.gif

Set Shell "bash"
Set FontSize 16
Set Width 1100
Set Height 700
Set Theme "TokyoNight"
Set WindowBar Colorful
Set WindowBarSize 40
Set TypingSpeed 30ms
Set Padding 20
Set CursorBlink false
Set LoopOffset 60%
Set Framerate 15

# Setup (hidden)
Hide
Type `export PATH=/Users/jason/Documents/GitHub/OpsDeck:$PATH && clear`
Enter
Sleep 1s
Show

# 1. List all sessions
Type "opsdeck list"
Enter
Sleep 4s

# 2. Clear and show brief
Type "clear"
Enter
Sleep 500ms
Type "opsdeck brief --since 12h"
Enter
Sleep 5s

# 3. Clear and show metrics
Type "clear"
Enter
Sleep 500ms
Type "opsdeck metrics"
Enter
Sleep 4s
