# SKILL.md Deep Dive

_From <https://leehanchung.github.io/blogs/2025/10/26/claude-skills-deep-dive/>._

SKILL.md is the core of an skill’s prompt. It is a markdown file that follows a two-part structure - frontmatter and content. The frontmatter configures HOW the skill runs (permissions, model, metadata), while the markdown content tells Claude WHAT to do. Frontmatter is the header of the markdown file written in YAML.

┌─────────────────────────────────────┐
│ 1. YAML Frontmatter (Metadata)      │ ← Configuration
│    ---                              │
│    name: skill-name                 │
│    description: Brief overview      │
│    allowed-tools: "Bash, Read"      │
│    version: 1.0.0                   │
│    ---                              │
├─────────────────────────────────────┤
│ 2. Markdown Content (Instructions)  │ ← Prompt for Claude
│                                     │
│    Purpose explanation              │
│    Detailed instructions            │
│    Examples and guidelines          │
│    Step-by-step procedures          │
└─────────────────────────────────────┘
Frontmatter

The frontmatter contains metadata that controls how Claude discovers and uses the skill. As an example, here’s the frontmatter from skill-creator:

```frontmatter yaml
---
name: skill-creator
description: Guide for creating effective skills. This skill should be used when users want to create a new skill (or update an existing skill) that extends Claude's capabilities with specialized knowledge, workflows, or tool integrations.
license: Complete terms in LICENSE.txt
---
```

Lets walk through the fields for the frontmatter one by one.

## Claude Skills Frontmatter

### name (Required)

The name of a skill is used as a command in Skill Tool.

### description (Required)

The description field provides a brief summary of what the skill does. This is the primary signal Claude uses to determine when to invoke a skill. In the example above, the description explicitly states “This skill should be used when users want to create a new skill” — this type of clear, action-oriented language helps Claude match user intent to skill capabilities.

### license (Optional)

Self explanatory.

### allowed-tools (Optional)

The allowed-tools field defines which tools the skill can use without user approval, similar to Claude’s allowed-tools.

This is a comma-separated string that gets parsed into an array of allowed tool names. You can use wildcards to scope permissions, e.g., Bash(git:*) allows only git subcommands, while Bash(npm:*) permits all npm operations. The skill-creator skill uses "Read,Write,Bash,Glob,Grep,Edit" to give it broad file and search capabilities. A common mistake is listing every available tool, which creates a security risk and defeats the security model.

Only include what your skill actually needs—if you’re just reading and writing files, "Read,Write" is sufficient.
# ✅ skill-creator allows multiple tools
allowed-tools: "Read,Write,Bash,Glob,Grep,Edit"

# ✅ Specific git commands only
allowed-tools: "Bash(git status:*),Bash(git diff:*),Bash(git log:*),Read,Grep"

# ✅ File operations only
allowed-tools: "Read,Write,Edit,Glob,Grep"

# ❌ Unnecessary surface area
allowed-tools: "Bash,Read,Write,Edit,Glob,Grep,WebSearch,Task,Agent"

# ❌ Unnecessary surface area with all npm commands
allowed-tools: "Bash(npm:*),Read,Write"

> [!NOTE]
> These allowed tools will be passed directly to the command line --allowed-tools.

### model (Optional)

The model field defines which model the skill can use. It defaults to inheriting the current model in the user session. For complex tasks like code review, skills can request more capable models such as Claude Opus or other OSS Chinese models. IYKYK.

model: "claude-opus-4-20250514"  # Use specific model
model: "inherit"                 # Use session's current model (default)

### version, disable-model-invocation, and mode (Optional)

Skills support three optional frontmatter fields for versioning and invocation control. The version field (e.g., version: “1.0.0”) is a metadata field for tracking skill versions, parsed from the frontmatter but primarily used for documentation and skill management purposes.

The disable-model-invocation field (boolean) prevents Claude from automatically invoking the skill via the Skill tool. When set to true, the skill is excluded from the list shown to Claude and can only be invoked manually by users via `/skill-name`, making it ideal for dangerous operations, configuration commands, or interactive workflows that require explicit user control.

The mode field (boolean) categorizes a skill as a “mode command” that modifies Claude’s behavior or context. When set to true, the skill appears in a special “Mode Commands” section at the top of the skills list (separate from regular utility skills), making it prominent for skills like debug-mode, expert-mode, or review-mode that establish specific operational contexts or workflows.

## SKILL.md Prompt Content

After the frontmatter comes the markdown content - the actual prompt that Claude receives when the skill is invoked. This is where you define the skill’s behavior, instructions, and workflows. The key to writing effective skill prompts is keeping them focused and using progressive disclosure: provide core instructions in SKILL.md, and reference external files for detailed content.

Here’s a recommended content structure

```markdown
---
# Frontmatter here
---

# [Brief Purpose Statement - 1-2 sentences]

## Overview
[What this skill does, when to use it, what it provides]

## Prerequisites
[Required tools, files, or context]

## Instructions

### Step 1: [First Action]
[Imperative instructions]
[Examples if needed]

### Step 2: [Next Action]
[Imperative instructions]

### Step 3: [Final Action]
[Imperative instructions]

## Output Format
[How to structure results]

## Error Handling
[What to do when things fail]

## Examples
[Concrete usage examples]

## Resources
[Reference scripts/, references/, assets/ if bundled]
```

As an example, skill-creator skill contains the following instructions that specifies each steps of the workflow required to create skills.

```markdown
## Skill Creation Process

### Step 1: Understanding the Skill with Concrete Examples
### Step 2: Planning the Reusable Skill Contents
### Step 3: Initializing the Skill
### Step 4: Edit the Skill
### Step 5: Packaging a Skill
```

When Claude invokes this skill, it receives the entire prompt as new instructions with the base directory path prepended. The {baseDir} variable resolves to the skill’s installation directory, allowing Claude to load reference files using the Read tool: `Read({baseDir}/scripts/init_skill.py)`. This pattern keeps the main prompt concise while making detailed documentation available on demand.

Best practices for prompt content:

Keep under 5,000 words (~800 lines) to avoid overwhelming context
Use imperative language (“Analyze code for…”) not second person (“You should analyze…”)
Reference external files for detailed content rather than embedding everything
Use {baseDir} for paths, never hardcode absolute paths like /home/user/project/
❌ Read /home/user/project/config.json
✅ Read {baseDir}/config.json
When the skill is invoked, Claude receives access only to the tools specified in allowed-tools, and the model may be overridden if specified in the frontmatter. The skill’s base directory path is automatically provided, making bundled resources accessible.
