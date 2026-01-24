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

# See all the files, commands, and output.
skillet skill-name --verbose

# Run a remote skill (e.g. the test skill from this repo)
skillet https://raw.githubusercontent.com/martinemde/skillet/refs/heads/main/.claude/skills/test-skill/SKILL.md
```

## So What?

Skillet makes claude scripting simple and beautiful.

I've made many throw-away headless `claude` scripts. None of them ever work on the first try. They always suck. There's an ugly prompt buried in the middle. It uses the wrong CLI flags, wrong permissions, or just skips permissions entirely. When you get it to run, your feedback is either _nothing_ or an unreadable flood of json.

Skills solve many of these problems by setting allowed tools, model, and more in the frontmatter. Skillet reads and parses skills directly, just like claude. Allowed tools, model, and other frontmatter gets fed to `claude` with the correct permissions for the skill.

Instead of the unpleasant or absent output, Skillet uses [Charm](https://charm.land/) to glam up the markdown hidden in claude's json streams and history logs. Skillet formats claude output beautifully, showing code, commands, and even errors in a sleek minimal interface with controllable verbosity.

## Installation

```bash
brew install martinemde/tap/skillet
```

Or download a pre-built [release](https://github.com/martinemde/skillet/releases).

## Quick start

1. Have claude make a skill: `Create a claude skill to summarize meeting transcripts`
2. Run it in claude: `/summarize-transcript "filename"`
3. Run it with skillet: `skillet summarize-transcript "filename"`

Then script it:

```bash
for file in transcripts/*.txt; do
  skillet summarize-transcript "summarize $file and output to meetings/${file%.txt}.md"
done
```

## Cooking with Skillet

```bash
# See which skills and commands you can run
skillet --list

# By skill name (looks in all .claude/skills/<skill-name>/SKILL.md paths)
skillet skill-or-command-name

# Run a remote URL
skillet https://raw.githubusercontent.com/user/repo/main/skill.md
```

> [!NOTICE]
> **Skills are a security risk.** Skills can execute commands, exfiltrate data, and modify files. Only use skills from sources you trust.

## Convert a Command to a Skill

[Commands are deprecated](https://martinemde.com/blog/claude-code-commands-deprecated).
Easily convert your existing commands to skills with Skillet:

```bash
# Non-destructive (you'll need to clean up the old commands yourself)
skillet command --convert-to-skill
```

## Parse or Review History

- Review past Claude sessions
- Browse conversation history logs in `~/.claude/projects/`
- Pipe live Claude output through Skillet's formatter

```bash
# Format a saved JSONL file
skillet --parse conversation.jsonl

# Pipe directly from Claude Code
claude -p "explain this code" --verbose --output-format stream-json | skillet --parse

# Read from stdin
cat session.jsonl | skillet --parse
```

### Browsing Claude History

With `fzf` we can make a simple history browser.
Claude stores conversation logs in `~/.claude/projects/`.

```bash
# Browse history for current project with live preview
find ~/.claude/projects/$(pwd | tr '/' '-') -name '*.jsonl' | \
  fzf --preview 'go run ./cmd/skillet --parse {} --verbose --color=always'
```

You can `brew install fzf` if you don't already have it.

### Using Task Lists

Start a skill with a pre-existing task list. This is useful when one agent creates tasks for another to complete.

```bash
# Run a skill with a specific task list
skillet --task-list=f766aec0-6085-46cb-9810-320278fd08a2 my-skill

# The task list ID is passed via CLAUDE_CODE_TASK_LIST_ID environment variable
# Claude will pick up and work on tasks from that list
```

This enables workflows where a planning agent creates a task list, then skillet runs a specialized skill to complete those tasks autonomously.

## Developing Skillet

Nerd shit (I love you nerds. Let's fry up some eggs or break some shells)

Make contributions:

1. All tests pass: `go test ./...`
2. Code follows Go conventions: `go fmt ./...`
3. New features include tests
4. Documentation is updated

- Make concise changes
- Make a second and third pass to ensure your change is focused and clean
- Test it manually to make sure it looks good (with human eyes)

### Requirements

- Go 1.21 or later
- Claude CLI installed and configured

## License

See LICENSE file for details.

## References

- [Agent Skills Specification](https://agentskills.io)
- [Claude CLI Documentation](https://code.claude.com/docs)
