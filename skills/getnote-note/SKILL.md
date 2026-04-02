---
name: getnote-note
version: 0.2.0
description: Manage notes in Get笔记 via the getnote CLI
---

# getnote-note Skill

Manage individual notes in Get笔记 — save, list, get, update, delete, and track save tasks.

## Prerequisites

- `getnote` CLI installed and authenticated (`getnote auth status` should show "Authenticated")

## Commands

### Save a note

```
getnote save <url|text|image_path> [--title <title>] [--tag <tag>]...
```

| Flag | Description |
|------|-------------|
| `--title` | Optional title |
| `--tag` | Tag to apply; may be repeated |

- URL (`http://` or `https://`) → saved as link note
- Local image path → saved as image note
- Otherwise → saved as text note

**Examples:**
```bash
getnote save https://example.com --title "Great article"
getnote save "Remember to review the docs" --tag work --tag important
getnote save ./screenshot.png --title "Design mockup"
```

URL saves are async; the CLI auto-polls until done, then shows the result.
In `-o json` mode, silently polls and returns the final note JSON.

---

### Track save task

```
getnote task <task_id>
```

Check progress of an async save task (returned by `save` for URLs).

```bash
getnote task task_xyz789
getnote task task_xyz789 -o json
```

---

### List recent notes

```
getnote notes [--limit <n>] [--since-id <id>] [--all]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--limit` | 20 | Notes per page |
| `--since-id` | — | Pagination cursor (last note ID seen) |
| `--all` | — | Fetch all notes (auto-paginate, streams output) |

```bash
getnote notes
getnote notes --limit 5
getnote notes --all
getnote notes --since-id 1234567890
getnote notes -o json
```

---

### Get note details

```
getnote note <id> [--field <field>]
```

| Flag | Description |
|------|-------------|
| `--field` | Output a single field: `id` / `title` / `content` / `type` / `created_at` / `updated_at` / `url` / `excerpt` |

```bash
getnote note 1234567890
getnote note 1234567890 --field content
getnote note 1234567890 --field url
getnote note 1234567890 -o json
```

---

### Update a note

```
getnote note update <id> [--title <title>] [--content <content>] [--tag <tags>]
```

| Flag | Description |
|------|-------------|
| `--title` | New title |
| `--content` | New content (plain_text notes only) |
| `--tag` | Comma-separated tags (replaces all existing tags) |

```bash
getnote note update 1234567890 --title "Updated title"
getnote note update 1234567890 --tag "work,important"
```

---

### Delete a note

```
getnote note delete <id> [-y]
```

Moves note to trash.

| Flag | Description |
|------|-------------|
| `-y` | Skip confirmation prompt |

```bash
getnote note delete 1234567890
getnote note delete 1234567890 -y
```

---

## Agent Usage Notes

- Use `-o json` when parsing responses programmatically.
- `save` for URLs is async; in `-o json` mode the CLI auto-polls and returns the final note — no manual `task` polling needed.
- `note update --tag` replaces **all** existing tags; use `getnote tag add/remove` for partial updates.
- Exit code `0` = success; non-zero = error. Error details go to stderr.
