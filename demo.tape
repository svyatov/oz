# VHS demo tape for oz
# Run: vhs demo.tape
# Output: docs/demo.gif

Output docs/demo.gif

Set Shell zsh
Set FontSize 16
Set Width 900
Set Height 500
Set Padding 20
Set TypingSpeed 60ms

# Show available wizards
Type "oz list"
Enter
Sleep 1.5s

# Run the rails-new wizard in dry-run mode
Type "oz run rails-new -n"
Enter
Sleep 2s

# Select Ruby version (first option)
Enter
Sleep 1s

# Type the app name
Type "myapp"
Sleep 500ms
Enter
Sleep 1s

# Select database (navigate to PostgreSQL)
Down
Sleep 300ms
Enter
Sleep 1s

# Skip through a few more options with defaults
Enter
Sleep 800ms
Enter
Sleep 800ms
Enter
Sleep 800ms
Enter
Sleep 800ms
Enter
Sleep 800ms
Enter
Sleep 800ms
Enter
Sleep 800ms

# Show the generated command
Sleep 3s
