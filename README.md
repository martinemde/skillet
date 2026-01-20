<div align="center">

[![asciicast](https://asciinema.org/a/768855.svg)](https://asciinema.org/a/768855)

# 🍳 Skillet

### Run [Agent Skills](https://agentskills.io) as Shell Scripts

[![Release](https://img.shields.io/github/v/release/martinemde/skillet)](https://github.com/martinemde/skillet/releases)
[![Go Version](https://img.shields.io/github/go-mod/go-version/martinemde/skillet)](https://go.dev/)
[![Go Report Card](https://goreportcard.com/badge/github.com/martinemde/skillet)](https://goreportcard.com/report/github.com/martinemde/skillet)
[![License](https://img.shields.io/github/license/martinemde/skillet)](LICENSE)
[![Build Status](https://img.shields.io/github/actions/workflow/status/martinemde/skillet/ci.yml?branch=main)](https://github.com/martinemde/skillet/actions)

</div>

---

Claude skills as _shell_ scripts with clean, beautiful output. :chefkiss:

```bash
# Runs .claude/skills/<skill-bame> as a command
skillet skill-name

# Run a remote skill (e.g. the test skill from this repo)
skillet https://raw.githubusercontent.com/martinemde/skillet/refs/heads/main/.claude/skills/test-skill/SKILL.md
```

## So What?

Sometimes you want to run claude like a shell script: "generate a summary of this transcript."
Sometimes you want to be able to run it hundreds of times.

With Skillet:

1. Make a `summarize-transcript` skill
2. Run `skillet summarize-transcript -p "$filename"`

I've made many throw-away headless `claude` scripts. None of them ever work on the first try. They always suck. There's an ugly prompt buried in the middle. It uses the wrong CLI flags, wrong permissions, or just skips permissions entirely. When you get it to run, your feedback is either _nothing_ or an unreadable flood of json.

Skills solve many of these problems by setting allowed tools, model, and more in the frontmatter, but invoking them from the command line undermines the advantage.

Skillet reads and parses the skill directly, just like claude, parses the tools, model, and other frontmatter, then runs `claude` with the correct permissions for the skill. Instead of the unpleasant output, Skillet uses Charm terminal formatting to glam up the markdown hidden in that json, showing code, commands, and even errors in a sleek minimal interface with controllable verbosity.

Skillet makes claude scripting simple.

## Installation

### Homebrew (macOS/Linux)

```bash
brew install martinemde/tap/skillet
```

### Install with Go

```bash
go install github.com/martinemde/skillet/cmd/skillet@latest
```

### Download pre-built binaries

Download the latest release for your platform from the [releases page](https://github.com/martinemde/skillet/releases):

## Usage

```bash
# See which skills and commands you can run
skillet --list

# By skill name (looks in all .claude/skills/<skill-name>/SKILL.md paths)
skillet skill-or-command-name

# By command name (looks in all .claude/skills/<namespace>/<command>.md paths)
skillet namespace:command

# Run a remote URL
skillet https://raw.githubusercontent.com/user/repo/main/skill.md
```

> [!NOTICE]
> **Skills are a security risk.** Skills can execute commands, exfiltrate data, and modify files. Only use skills from sources you trust.

## Parse or Review History

Skillet can format Claude's history files or `stream-json` output.

- Review past Claude sessions
- Browse conversation history logs in `~/.claude/projects/`
- Pipe live Claude output through Skillet's formatter

```bash
# Format a saved JSONL file
skillet --parse conversation.jsonl

# Pipe directly from Claude
claude -p "explain this code" --verbose --output-format stream-json | skillet --parse

# Read from stdin (explicit)
cat session.jsonl | skillet --parse -
```

### Browsing Claude History

Claude stores conversation logs in `~/.claude/projects/`.
Here's a cool hack: you can browse them interactively with `fzf`:

```bash
# Browse history for current project with live preview
find ~/.claude/projects/$(pwd | tr '/' '-') -name '*.jsonl' | \
  fzf --preview '/tmp/skillet --parse {} --color=always --verbose'
```

## Testing

Run all tests:

```bash
go test -v ./...
```

## Examples

### Review Skill

```yaml
---
name: code-reviewer
description: Expert code reviewer. Use after making code changes.
allowed-tools: Read Grep Glob Bash(git:*)
model: claude-opus-4-5-20251101
license: Apache-2.0
metadata:
  author: example-org
  version: "1.0.0"
---

# Code Reviewer

Reviews code for quality, security, and best practices.

## Instructions

1. Read the code changes using Read and Grep
2. Analyze for potential issues
3. Provide detailed feedback
```

### With Variable Interpolation

```yaml
---
name: data-processor
description: Processes data files. Use for data analysis tasks.
allowed-tools: Read Write Bash
---

# Data Processor

Process data files from the skill directory.

## Setup

1. Read configuration from {baseDir}/config.json
2. Load data from {baseDir}/data/input.csv
3. Write output to {baseDir}/data/output.csv
```

## Requirements

- Go 1.21 or later
- Claude CLI installed and configured

## Contributing

Contributions are welcome! Please ensure:

1. All tests pass: `go test ./...`
2. Code follows Go conventions: `go fmt ./...`
3. New features include tests
4. Documentation is updated

## License

See LICENSE file for details.

## References

- [Agent Skills Specification](https://agentskills.io)
- [Claude CLI Documentation](https://code.claude.com/docs)
