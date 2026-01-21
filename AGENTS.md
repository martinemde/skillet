# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Skillet is a CLI tool written in Go that executes Claude Agent Skills (from agentskills.io) and commands as shell scripts with formatted output. It parses SKILL.md and command.md files, extracts frontmatter configuration (allowed-tools, model, etc.), interpolates variables like `{baseDir}`, and invokes the Claude CLI with appropriate permissions in headless mode. Output is formatted using Charm libraries (glamour, lipgloss) for clean terminal rendering.

## Always test, format and lint code before finishing

```bash
go test ./...
go fmt ./...
golangci-lint run
```

## Architecture

### Core Components

**main.go** (`cmd/skillet/main.go`): Entry point that orchestrates the execution flow:

- Parses CLI flags using custom flag separation logic to support flags anywhere in args
- Resolves skill/command name → file path via `resolver` package
- Builds executor config from merged CLI flags + frontmatter
- Runs executor and formatter concurrently using pipes
- Handles signals and context cancellation

**parser** (`internal/parser/`): Parses SKILL.md files with YAML frontmatter

- Extracts frontmatter (name, description, allowed-tools, model, etc.)
- Interpolates `{baseDir}` variables in content
- Validates skill structure and constraints (name format, field lengths)

**command** (`internal/command/`): Parses command .md files (similar to skills but simpler)

- Derives command name from filename
- Supports optional frontmatter (description can be auto-extracted from content)
- Handles variable interpolation like skills

**executor** (`internal/executor/`): Builds and runs `claude` CLI commands

- Constructs args from resolved config (model, allowed-tools, system-prompt, etc.)
- Executes in headless mode with `-p --verbose --output-format stream-json`
- Provides dry-run mode to show command without executing

**formatter** (`internal/formatter/`): Processes claude's stream-json output into readable markdown

- Parses JSON stream events
- Renders markdown using glamour
- Handles color control (auto/always/never)
- Shows usage statistics when requested
- Supports passthrough mode for raw output

**discovery** (`internal/discovery/`): Finds skills across search paths

- Scans `.claude/skills/*/SKILL.md` in multiple source directories
- Handles precedence (earlier paths shadow later ones)
- Marks overshadowed skills for visibility

**skillpath** (`internal/skillpath/`): Manages skill search paths with priority ordering

- Default sources: current directory, home directory `.claude/skills`
- Skills in earlier sources take precedence

**commandpath** (`internal/commandpath/`): Manages command search paths (parallel to skillpath)

- Default sources: current directory `.claude/commands`, home `.claude/commands`

**resolver** (`internal/resolver/`): Resolves skill/command names → file paths

- Handles exact paths, directory paths, skill names, command names, URLs
- Downloads remote skills/commands to temp files
- Determines resource type (skill vs command) based on filename/structure

### Key Patterns

**Dual Resource System**: Skills and commands are parallel but distinct

- Skills: Must have `name` and `description` in frontmatter, stored as `SKILL.md`
- Commands: Derive name from filename, description is optional, stored as `<name>.md`
- Both support frontmatter (allowed-tools, model, etc.) and variable interpolation

**Variable Interpolation**:

- `{baseDir}` in skills gets replaced with the absolute path to the directory containing the file.
- `$ARGUMENTS` in commands gets replaced with the additional CLI arguments passed after the command name.

**Precedence and Shadowing**: When multiple sources contain the same skill/command name, earlier sources take precedence. Shadowed items are shown in `--list` but marked as overshadowed.

**Concurrent Execution**: Main goroutine spawns two goroutines:

1. Executor: Runs `claude` CLI and writes to pipe
2. Formatter: Reads from pipe and renders output
   Error handling collects errors from both channels.

## Issue Tracking

This project uses **bd (beads)** for issue tracking.
Run `bd prime` for workflow context, or install hooks (`bd hooks install`) for auto-injection.

**Quick reference:**

- `bd ready` - Find unblocked work
- `bd create "Title" --type task --priority 2` - Create issue
- `bd close <id>` - Complete work
- `bd sync` - Sync with git (run at session end)

For full workflow details: `bd prime`
