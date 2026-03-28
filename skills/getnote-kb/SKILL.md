---
name: getnote-kb
version: 0.1.0
description: Manage knowledge bases in Get笔记 via the getnote CLI
---

# getnote-kb Skill

Manage knowledge bases (话题/topic) in Get笔记 — list, create, and add or remove notes.

## Prerequisites

- `getnote` CLI installed and authenticated (`getnote auth status` should show "Authenticated")
- API key configured via `getnote auth login --api-key <key>` or the `GETNOTE_API_KEY` environment variable

## Commands

### List knowledge bases

```
getnote kb list [--output json|table]
```

Returns all knowledge bases accessible to the authenticated user.

**Examples:**
```bash
# Human-friendly table
getnote kb list

# Machine-readable JSON
getnote kb list --output json
```

---

### Create a knowledge base

```
getnote kb create <name> [--desc <description>]
```

| Flag | Description |
|------|-------------|
| `--desc` | Optional description for the knowledge base |

**Examples:**
```bash
getnote kb create "Research Papers"
getnote kb create "Project Docs" --desc "Documentation links for the main project"
```

---

### List notes in a knowledge base

```
getnote kb notes <topic_id> [--limit <n>] [--output json|table]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--limit` | 20 | Maximum number of notes to return |

**Examples:**
```bash
# List notes in a knowledge base
getnote kb notes kb_abc123

# Get more results
getnote kb notes kb_abc123 --limit 50

# Machine-readable output
getnote kb notes kb_abc123 --output json
```

---

### Add notes to a knowledge base

```
getnote kb add <topic_id> <note_id> [note_id...]
```

Supports adding multiple notes in a single call.

**Examples:**
```bash
# Add a single note
getnote kb add kb_abc123 note_xyz789

# Add multiple notes at once
getnote kb add kb_abc123 note_xyz789 note_def456 note_ghi012
```

---

### Remove notes from a knowledge base

```
getnote kb remove <topic_id> <note_id> [note_id...]
```

Supports removing multiple notes in a single call.

**Examples:**
```bash
# Remove a single note
getnote kb remove kb_abc123 note_xyz789

# Remove multiple notes at once
getnote kb remove kb_abc123 note_xyz789 note_def456
```

---

## Agent Usage Notes

- Use `--output json` for all commands when parsing responses programmatically.
- To get a `topic_id`, first call `getnote kb list --output json` and extract the `id` field.
- `kb add` and `kb remove` accept multiple note IDs in a single call — prefer batching over multiple calls.
- Exit code `0` = success; non-zero = error. Error details go to stderr.
- Combine with `getnote note list --output json` to get note IDs before adding them to a knowledge base.
