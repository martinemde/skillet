---
name: test-skill
description: A simple test skill that creates a greeting file
allowed-tools: Write Read
model: sonnet
---

# Test Skill

This is a simple test skill to verify skillet is working correctly.

## Task

Create a todo list item:

1. Check if the greeting.txt file exists
2. Write a new version of the greeting.txt file with the content

```
Hello from Skillet!
This file was created by Claude Code through the skillet CLI.
Date: [current date]
```

After creating the file, read it back and confirm the contents were written correctly.
