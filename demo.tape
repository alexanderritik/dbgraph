# demo.tape
Output demo.gif

# 1. Aesthetics
Set FontSize 16
Set Width 1200
Set Height 600
Set Padding 20
Set FontFamily "Menlo"      # Realistic Mac font
Set Theme "Catppuccin Macchiato" # Clean, modern dark theme
Set LineHeight 1.2
Set TypingSpeed 60ms

# 2. Setup (Set env var to keep commands clean)
Type "export DBGRAPH_DB_URL='postgres://postgres:postgres@localhost:5432/dbgraph_test'"
Enter
# Clear screen to start fresh demo
Ctrl+L 
Sleep 1s

# 3. Intro & Impact Analysis
Type "# 1. Visualize Dangerous Cascades"
Sleep 500ms
Enter
Type "dbgraph impact users"
Sleep 500ms
Enter
Sleep 3s

# 4. Simulation (Zoom In for detail)
Ctrl+L
Sleep 500ms
Set FontSize 22
Type "# 2. Simulate Breaking Changes (Dry Run)"
Sleep 500ms
Enter
Type "dbgraph simulate --drop-column users.email"
Sleep 500ms
Enter
Sleep 4s

# 5. Trace Performance
Ctrl+L
Type "# 3. Deep Query Tracing"
Sleep 500ms
Enter
# Trace a clean, readable query
Type "dbgraph trace --query 'SELECT * FROM users JOIN orders ON users.id = orders.user_id'"
Sleep 500ms
Enter
Sleep 4s

# 6. Summary & Outro (Zoom Out)
Ctrl+L
Set FontSize 16
Type "# 4. Architectural Overview"
Sleep 500ms
Enter
Type "dbgraph summary"
Sleep 500ms
Enter
Sleep 3s
