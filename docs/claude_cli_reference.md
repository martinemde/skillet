# CLI reference

> Complete reference for Claude Code command-line interface, including commands and flags.

## CLI commands

| Command                         | Description                                            | Example                                           |
| :------------------------------ | :----------------------------------------------------- | :------------------------------------------------ |
| `claude`                        | Start interactive REPL                                 | `claude`                                          |
| `claude "query"`                | Start REPL with initial prompt                         | `claude "explain this project"`                   |
| `claude -p "query"`             | Query via SDK, then exit                               | `claude -p "explain this function"`               |
| `cat file \| claude -p "query"` | Process piped content                                  | `cat logs.txt \| claude -p "explain"`             |
| `claude -c`                     | Continue most recent conversation in current directory | `claude -c`                                       |
| `claude -c -p "query"`          | Continue via SDK                                       | `claude -c -p "Check for type errors"`            |
| `claude -r "<session>" "query"` | Resume session by ID or name                           | `claude -r "auth-refactor" "Finish this PR"`      |
| `claude update`                 | Update to latest version                               | `claude update`                                   |
| `claude mcp`                    | Configure Model Context Protocol (MCP) servers         | See the [Claude Code MCP documentation](/en/mcp). |

## CLI flags

Customize Claude Code's behavior with these command-line flags:

| Flag                             | Description                                                                                                                                                                                             | Example                                                                                            |
| :------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ | :------------------------------------------------------------------------------------------------- |
| `--add-dir`                      | Add additional working directories for Claude to access (validates each path exists as a directory)                                                                                                     | `claude --add-dir ../apps ../lib`                                                                  |
| `--agent`                        | Specify an agent for the current session (overrides the `agent` setting)                                                                                                                                | `claude --agent my-custom-agent`                                                                   |
| `--agents`                       | Define custom [subagents](/en/sub-agents) dynamically via JSON (see below for format)                                                                                                                   | `claude --agents '{"reviewer":{"description":"Reviews code","prompt":"You are a code reviewer"}}'` |
| `--allowedTools`                 | Tools that execute without prompting for permission. To restrict which tools are available, use `--tools` instead                                                                                       | `"Bash(git log:*)" "Bash(git diff:*)" "Read"`                                                      |
| `--append-system-prompt`         | Append custom text to the end of the default system prompt (works in both interactive and print modes)                                                                                                  | `claude --append-system-prompt "Always use TypeScript"`                                            |
| `--betas`                        | Beta headers to include in API requests (API key users only)                                                                                                                                            | `claude --betas interleaved-thinking`                                                              |
| `--chrome`                       | Enable [Chrome browser integration](/en/chrome) for web automation and testing                                                                                                                          | `claude --chrome`                                                                                  |
| `--continue`, `-c`               | Load the most recent conversation in the current directory                                                                                                                                              | `claude --continue`                                                                                |
| `--dangerously-skip-permissions` | Skip permission prompts (use with caution)                                                                                                                                                              | `claude --dangerously-skip-permissions`                                                            |
| `--debug`                        | Enable debug mode with optional category filtering (for example, `"api,hooks"` or `"!statsig,!file"`)                                                                                                   | `claude --debug "api,mcp"`                                                                         |
| `--disallowedTools`              | Tools that are removed from the model's context and cannot be used                                                                                                                                      | `"Bash(git log:*)" "Bash(git diff:*)" "Edit"`                                                      |
| `--fallback-model`               | Enable automatic fallback to specified model when default model is overloaded (print mode only)                                                                                                         | `claude -p --fallback-model sonnet "query"`                                                        |
| `--fork-session`                 | When resuming, create a new session ID instead of reusing the original (use with `--resume` or `--continue`)                                                                                            | `claude --resume abc123 --fork-session`                                                            |
| `--ide`                          | Automatically connect to IDE on startup if exactly one valid IDE is available                                                                                                                           | `claude --ide`                                                                                     |
| `--include-partial-messages`     | Include partial streaming events in output (requires `--print` and `--output-format=stream-json`)                                                                                                       | `claude -p --output-format stream-json --include-partial-messages "query"`                         |
| `--input-format`                 | Specify input format for print mode (options: `text`, `stream-json`)                                                                                                                                    | `claude -p --output-format json --input-format stream-json`                                        |
| `--json-schema`                  | Get validated JSON output matching a JSON Schema after agent completes its workflow (print mode only, see [Agent SDK Structured Outputs](https://docs.claude.com/en/docs/agent-sdk/structured-outputs)) | `claude -p --json-schema '{"type":"object","properties":{...}}' "query"`                           |
| `--max-turns`                    | Limit the number of agentic turns (print mode only). Exits with an error when the limit is reached. No limit by default                                                                                 | `claude -p --max-turns 3 "query"`                                                                  |
| `--mcp-config`                   | Load MCP servers from JSON files or strings (space-separated)                                                                                                                                           | `claude --mcp-config ./mcp.json`                                                                   |
| `--model`                        | Sets the model for the current session with an alias for the latest model (`sonnet` or `opus`) or a model's full name                                                                                   | `claude --model claude-sonnet-4-5-20250929`                                                        |
| `--no-chrome`                    | Disable [Chrome browser integration](/en/chrome) for this session                                                                                                                                       | `claude --no-chrome`                                                                               |
| `--output-format`                | Specify output format for print mode (options: `text`, `json`, `stream-json`)                                                                                                                           | `claude -p "query" --output-format json`                                                           |
| `--permission-mode`              | Begin in a specified [permission mode](/en/iam#permission-modes)                                                                                                                                        | `claude --permission-mode plan`                                                                    |
| `--permission-prompt-tool`       | Specify an MCP tool to handle permission prompts in non-interactive mode                                                                                                                                | `claude -p --permission-prompt-tool mcp_auth_tool "query"`                                         |
| `--plugin-dir`                   | Load plugins from directories for this session only (repeatable)                                                                                                                                        | `claude --plugin-dir ./my-plugins`                                                                 |
| `--print`, `-p`                  | Print response without interactive mode (see [SDK documentation](https://docs.claude.com/en/docs/agent-sdk) for programmatic usage details)                                                             | `claude -p "query"`                                                                                |
| `--resume`, `-r`                 | Resume a specific session by ID or name, or show an interactive picker to choose a session                                                                                                              | `claude --resume auth-refactor`                                                                    |
| `--session-id`                   | Use a specific session ID for the conversation (must be a valid UUID)                                                                                                                                   | `claude --session-id "550e8400-e29b-41d4-a716-446655440000"`                                       |
| `--setting-sources`              | Comma-separated list of setting sources to load (`user`, `project`, `local`)                                                                                                                            | `claude --setting-sources user,project`                                                            |
| `--settings`                     | Path to a settings JSON file or a JSON string to load additional settings from                                                                                                                          | `claude --settings ./settings.json`                                                                |
| `--strict-mcp-config`            | Only use MCP servers from `--mcp-config`, ignoring all other MCP configurations                                                                                                                         | `claude --strict-mcp-config --mcp-config ./mcp.json`                                               |
| `--system-prompt`                | Replace the entire system prompt with custom text (works in both interactive and print modes)                                                                                                           | `claude --system-prompt "You are a Python expert"`                                                 |
| `--system-prompt-file`           | Load system prompt from a file, replacing the default prompt (print mode only)                                                                                                                          | `claude -p --system-prompt-file ./custom-prompt.txt "query"`                                       |
| `--tools`                        | Restrict which built-in tools Claude can use (works in both interactive and print modes). Use `""` to disable all, `"default"` for all, or tool names like `"Bash,Edit,Read"`                           | `claude --tools "Bash,Edit,Read"`                                                                  |
| `--verbose`                      | Enable verbose logging, shows full turn-by-turn output (helpful for debugging in both print and interactive modes)                                                                                      | `claude --verbose`                                                                                 |
| `--version`, `-v`                | Output the version number                                                                                                                                                                               | `claude -v`                                                                                        |

<Tip>
  The `--output-format json` flag is particularly useful for scripting and
  automation, allowing you to parse Claude's responses programmatically.
</Tip>

### Agents flag format

The `--agents` flag accepts a JSON object that defines one or more custom subagents. Each subagent requires a unique name (as the key) and a definition object with the following fields:

| Field         | Required | Description                                                                                                            |
| :------------ | :------- | :--------------------------------------------------------------------------------------------------------------------- |
| `description` | Yes      | Natural language description of when the subagent should be invoked                                                    |
| `prompt`      | Yes      | The system prompt that guides the subagent's behavior                                                                  |
| `tools`       | No       | Array of specific tools the subagent can use (for example, `["Read", "Edit", "Bash"]`). If omitted, inherits all tools |
| `model`       | No       | Model alias to use: `sonnet`, `opus`, or `haiku`. If omitted, uses the default subagent model                          |

Example:

```bash  theme={null}
claude --agents '{
  "code-reviewer": {
    "description": "Expert code reviewer. Use proactively after code changes.",
    "prompt": "You are a senior code reviewer. Focus on code quality, security, and best practices.",
    "tools": ["Read", "Grep", "Glob", "Bash"],
    "model": "sonnet"
  },
  "debugger": {
    "description": "Debugging specialist for errors and test failures.",
    "prompt": "You are an expert debugger. Analyze errors, identify root causes, and provide fixes."
  }
}'
```

For more details on creating and using subagents, see the [subagents documentation](/en/sub-agents).

### System prompt flags

Claude Code provides three flags for customizing the system prompt, each serving a different purpose:

| Flag                     | Behavior                           | Modes               | Use Case                                                             |
| :----------------------- | :--------------------------------- | :------------------ | :------------------------------------------------------------------- |
| `--system-prompt`        | **Replaces** entire default prompt | Interactive + Print | Complete control over Claude's behavior and instructions             |
| `--system-prompt-file`   | **Replaces** with file contents    | Print only          | Load prompts from files for reproducibility and version control      |
| `--append-system-prompt` | **Appends** to default prompt      | Interactive + Print | Add specific instructions while keeping default Claude Code behavior |

**When to use each:**

* **`--system-prompt`**: Use when you need complete control over Claude's system prompt. This removes all default Claude Code instructions, giving you a blank slate.
  ```bash  theme={null}
  claude --system-prompt "You are a Python expert who only writes type-annotated code"
  ```

* **`--system-prompt-file`**: Use when you want to load a custom prompt from a file, useful for team consistency or version-controlled prompt templates.
  ```bash  theme={null}
  claude -p --system-prompt-file ./prompts/code-review.txt "Review this PR"
  ```

* **`--append-system-prompt`**: Use when you want to add specific instructions while keeping Claude Code's default capabilities intact. This is the safest option for most use cases.
  ```bash  theme={null}
  claude --append-system-prompt "Always use TypeScript and include JSDoc comments"
  ```

<Note>
  `--system-prompt` and `--system-prompt-file` are mutually exclusive. You cannot use both flags simultaneously.
</Note>

<Tip>
  For most use cases, `--append-system-prompt` is recommended as it preserves Claude Code's built-in capabilities while adding your custom requirements. Use `--system-prompt` or `--system-prompt-file` only when you need complete control over the system prompt.
</Tip>

For detailed information about print mode (`-p`) including output formats,
streaming, verbose logging, and programmatic usage, see the
[SDK documentation](https://docs.claude.com/en/docs/agent-sdk).


### Permission system

Permission rules use the format: `Tool` or `Tool(optional-specifier)`

A rule that is just the tool name matches any use of that tool. For example, adding `Bash` to the list of allow rules would allow Claude Code to use the Bash tool without requiring user approval.

#### Permission modes

Claude Code supports several permission modes that can be set as the `defaultMode` in [settings files](/en/settings#settings-files):

| Mode                | Description                                                                                                               |
| :------------------ | :------------------------------------------------------------------------------------------------------------------------ |
| `default`           | Standard behavior - prompts for permission on first use of each tool                                                      |
| `acceptEdits`       | Automatically accepts file edit permissions for the session                                                               |
| `plan`              | Plan Mode - Claude can analyze but not modify files or execute commands                                                   |
| `dontAsk`           | Auto-denies tools unless pre-approved via `/permissions` or [`permissions.allow`](/en/settings#permission-settings) rules |
| `bypassPermissions` | Skips all permission prompts (requires safe environment - see warning below)                                              |

#### Working directories

By default, Claude has access to files in the directory where it was launched. You can extend this access:

* **During startup**: Use `--add-dir <path>` CLI argument
* **During session**: Use `/add-dir` slash command
* **Persistent configuration**: Add to `additionalDirectories` in [settings files](/en/settings#settings-files)

Files in additional directories follow the same permission rules as the original working directory - they become readable without prompts, and file editing permissions follow the current permission mode.

#### Tool-specific permission rules

Some tools support more fine-grained permission controls:

**Bash**

Bash permission rules support both prefix matching with `:*` and wildcard matching with `*`:

* `Bash(npm run build)` Matches the exact Bash command `npm run build`
* `Bash(npm run test:*)` Matches Bash commands starting with `npm run test`
* `Bash(npm *)` Matches any command starting with `npm ` (e.g., `npm install`, `npm run build`)
* `Bash(* install)` Matches any command ending with ` install` (e.g., `npm install`, `yarn install`)
* `Bash(git * main)` Matches commands like `git checkout main`, `git merge main`

<Tip>
  Claude Code is aware of shell operators (like `&&`) so a prefix match rule like `Bash(safe-cmd:*)` won't give it permission to run the command `safe-cmd && other-cmd`
</Tip>

<Warning>
  Important limitations of Bash permission patterns:

  1. The `:*` wildcard only works at the end of a pattern for prefix matching
  2. The `*` wildcard can appear at any position and matches any sequence of characters
  3. Patterns like `Bash(curl http://github.com/:*)` can be bypassed in many ways:
     * Options before URL: `curl -X GET http://github.com/...` won't match
     * Different protocol: `curl https://github.com/...` won't match
     * Redirects: `curl -L http://bit.ly/xyz` (redirects to github)
     * Variables: `URL=http://github.com && curl $URL` won't match
     * Extra spaces: `curl  http://github.com` won't match

  For more reliable URL filtering, consider:

  * Using the WebFetch tool with `WebFetch(domain:github.com)` permission
  * Instructing Claude Code about your allowed curl patterns via CLAUDE.md
  * Using hooks for custom permission validation
</Warning>

**Read & Edit**

`Edit` rules apply to all built-in tools that edit files. Claude will make a best-effort attempt to apply `Read` rules to all built-in tools that read files like Grep and Glob.

Read & Edit rules both follow the [gitignore](https://git-scm.com/docs/gitignore) specification with four distinct pattern types:

| Pattern            | Meaning                                | Example                          | Matches                            |
| ------------------ | -------------------------------------- | -------------------------------- | ---------------------------------- |
| `//path`           | **Absolute** path from filesystem root | `Read(//Users/alice/secrets/**)` | `/Users/alice/secrets/**`          |
| `~/path`           | Path from **home** directory           | `Read(~/Documents/*.pdf)`        | `/Users/alice/Documents/*.pdf`     |
| `/path`            | Path **relative to settings file**     | `Edit(/src/**/*.ts)`             | `<settings file path>/src/**/*.ts` |
| `path` or `./path` | Path **relative to current directory** | `Read(*.env)`                    | `<cwd>/*.env`                      |

<Warning>
  A pattern like `/Users/alice/file` is NOT an absolute path - it's relative to your settings file! Use `//Users/alice/file` for absolute paths.
</Warning>

* `Edit(/docs/**)` - Edits in `<project>/docs/` (NOT `/docs/`!)
* `Read(~/.zshrc)` - Reads your home directory's `.zshrc`
* `Edit(//tmp/scratch.txt)` - Edits the absolute path `/tmp/scratch.txt`
* `Read(src/**)` - Reads from `<current-directory>/src/`

**WebFetch**

* `WebFetch(domain:example.com)` Matches fetch requests to example.com

**MCP**

* `mcp__puppeteer` Matches any tool provided by the `puppeteer` server (name configured in Claude Code)
* `mcp__puppeteer__*` Wildcard syntax that also matches all tools from the `puppeteer` server
* `mcp__puppeteer__puppeteer_navigate` Matches the `puppeteer_navigate` tool provided by the `puppeteer` server

**Task (Subagents)**

Use `Task(AgentName)` rules to control which [subagents](/en/sub-agents) Claude can use:

* `Task(Explore)` Matches the Explore subagent
* `Task(Plan)` Matches the Plan subagent
* `Task(Verify)` Matches the Verify subagent

Add these rules to the `deny` array in your [settings](/en/settings#permission-settings) or use the `--disallowedTools` CLI flag to disable specific agents. For example, to disable the Explore agent:

```json  theme={null}
{
  "permissions": {
    "deny": ["Task(Explore)"]
  }
}
```


---

> To find navigation and other pages in this documentation, fetch the llms.txt file at: https://code.claude.com/docs/llms.txt
