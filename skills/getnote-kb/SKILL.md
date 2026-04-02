---
name: getnote-kb
version: 0.3.0
description: Manage knowledge bases in Get笔记 via the getnote CLI
---

# getnote-kb Skill

Manage knowledge bases — list, create, browse notes, add/remove notes.

## Prerequisites

- `getnote` CLI installed and authenticated (`getnote auth status` should show "Authenticated")

## Commands

### List all knowledge bases

```
getnote kbs
```

Returns all knowledge bases. Each item includes: `topic_id`, `name`, `description`, `note_count`, `created_at`.

```bash
getnote kbs
getnote kbs -o json
```

---

### List notes in a knowledge base

```
getnote kb <topic_id> [--limit <n>] [--all]
```

Returns 20 notes per page by default.

| Flag | Default | Description |
|------|---------|-------------|
| `--limit` | 20 | Notes per page |
| `--all` | — | Fetch all notes (auto-paginate) |

```bash
getnote kb vnrOAaGY
getnote kb vnrOAaGY --all
getnote kb vnrOAaGY -o json
```

---

### Create a knowledge base

```
getnote kb create <name> [--desc <description>]
```

```bash
getnote kb create "Research Papers"
getnote kb create "Project Docs" --desc "Documentation links"
```

> Max 50 knowledge bases per day (resets at 00:00 Beijing time).

---

### Add notes to a knowledge base

```
getnote kb add <topic_id> <note_id> [note_id...]
```

Supports multiple note IDs. Max 20 per call.

```bash
getnote kb add vnrOAaGY 1234567890
getnote kb add vnrOAaGY 1234567890 9876543210
```

> Already-existing notes are silently skipped.

---

### Remove notes from a knowledge base

```
getnote kb remove <topic_id> <note_id> [note_id...]
```

```bash
getnote kb remove vnrOAaGY 1234567890
```

---

## Agent Usage Notes

- Use `-o json` when parsing results programmatically.
- Get `topic_id` from `getnote kbs -o json` (the `topic_id` field, not `id`).
- `kb add` / `kb remove` accept multiple note IDs — prefer batching over multiple calls.
- Exit code `0` = success; non-zero = error. Error details go to stderr.
