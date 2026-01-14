---
name: test-skill
description: A simple test skill that creates a greeting file
allowed-tools: Write Read
model: claude-sonnet-4-5-20250929
---

# Test Skill

This is a simple test skill to verify skillet is working correctly.

## Task

Create a file called `greeting.txt` in the current directory with the following content:

```
Hello from Skillet!
This file was created by Claude Code through the skillet CLI.
Date: [current date]
```

After creating the file, read it back and confirm the contents were written correctly.
