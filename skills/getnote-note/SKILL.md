---
name: getnote-note
version: 0.1.0
description: Manage notes in Get笔记 via the getnote CLI
---

# getnote-note Skill

Manage individual notes in Get笔记 — save, list, get, update, delete, and track save tasks.

## Prerequisites

- `getnote` CLI installed and authenticated (`getnote auth status` should show "Authenticated")
- API key configured via `getnote auth login --api-key <key>` or the `GETNOTE_API_KEY` environment variable

## Commands

### Save a note

```
getnote note save <url|text> [--title <title>] [--tag <tag>]...
```

| Flag | Description |
|------|-------------|
| `--title` | Optional title for the note |
| `--tag` | Tag to apply; may be repeated |

- If the argument starts with `http://` or `https://`, it is saved as a **link note**.
- Otherwise it is saved as a **text note**.

**Examples:**
```bash
# Save a URL
getnote note save https://example.com/article --title "Interesting article" --tag reading

# Save plain text
getnote note save "Remember to review the API docs" --tag todo

# Save with multiple tags
getnote note save https://docs.example.com --tag docs --tag reference
```

The response includes a `task_id` you can use to track progress:
```bash
getnote note task <task_id>
```

---

### List notes

```
getnote note list [--limit <n>] [--since-id <id>] [--output json|table]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--limit` | 20 | Maximum number of notes to return |
| `--since-id` | — | Cursor for pagination (ID of last note seen) |
| `--output` | table | Output format: `table` or `json` |

**Examples:**
```bash
# List recent 20 notes
getnote note list

# List 50 notes
getnote note list --limit 50

# Paginate using cursor
getnote note list --since-id note_abc123

# Machine-readable JSON output
getnote note list --output json
```

---

### Get note details

```
getnote note get <note_id>
```

**Example:**
```bash
getnote note get note_abc123
getnote note get note_abc123 --output json
```

---

### Update a note

```
getnote note update <note_id> [--title <title>] [--content <content>] [--tags <tag1,tag2>]
```

| Flag | Description |
|------|-------------|
| `--title` | New title |
| `--content` | New content body |
| `--tags` | Comma-separated tags (replaces all existing tags) |

**Example:**
```bash
getnote note update note_abc123 --title "Updated title" --tags "work,important"
```

---

### Delete a note

```
getnote note delete <note_id> [--yes]
```

| Flag | Description |
|------|-------------|
| `--yes` | Skip the confirmation prompt |

**Examples:**
```bash
# Interactive confirmation
getnote note delete note_abc123

# Skip confirmation (useful in scripts)
getnote note delete note_abc123 --yes
```

---

### Query task progress

```
getnote note task <task_id>
```

Use this after `note save` to check whether a URL has been processed.

**Example:**
```bash
getnote note task task_xyz789
getnote note task task_xyz789 --output json
```

---

## Agent Usage Notes

- Use `--output json` for all commands when parsing responses programmatically.
- `note save` may be async for URLs; poll `note task <task_id>` until `status` is `done`.
- `note list --output json` returns a JSON array suitable for further processing with `jq`.
- Pass `--yes` to `note delete` to avoid interactive prompts in automated workflows.
- Exit code `0` = success; non-zero = error. Error details go to stderr.
