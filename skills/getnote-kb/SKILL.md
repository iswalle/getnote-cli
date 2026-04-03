---
name: getnote-kb
version: 0.5.0
description: Manage knowledge bases in Get笔记 via the getnote CLI
---

# getnote-kb Skill

Manage knowledge bases — list, create, browse notes, add/remove notes. Also supports subscribed knowledge bases, blogger content, live sessions, and quota.

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

### List subscribed knowledge bases

```
getnote kbs-sub [--page <n>]
```

Returns knowledge bases the user has subscribed to. Supports pagination.

| Flag | Default | Description |
|------|---------|-------------|
| `--page` | 1 | Page number |

```bash
getnote kbs-sub
getnote kbs-sub --page 2
getnote kbs-sub -o json
```

> Use `getnote kb <topic_id>` to browse notes inside a subscribed knowledge base.

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

### List bloggers in a knowledge base

```
getnote kb bloggers <topic_id> [--page <n>]
```

Returns subscribed bloggers. Each item includes: `follow_id` (required for content queries), `account_name`, `platform`, `follow_time`.

```bash
getnote kb bloggers vnrOAaGY
getnote kb bloggers vnrOAaGY --page 2 -o json
```

---

### List blogger contents

```
getnote kb blogger-contents <topic_id> <follow_id> [--page <n>]
```

Returns content list (no full text). Use `post_id_alias` to fetch detail.

```bash
getnote kb blogger-contents vnrOAaGY follow123
getnote kb blogger-contents vnrOAaGY follow123 --page 2
```

---

### Show blogger content detail

```
getnote kb blogger-content <topic_id> <post_id>
```

Returns full content including original text (`post_media_text`).

```bash
getnote kb blogger-content vnrOAaGY post_abc123
getnote kb blogger-content vnrOAaGY post_abc123 -o json
```

---

### List completed lives in a knowledge base

```
getnote kb lives <topic_id> [--page <n>]
```

Returns only completed live sessions that have been AI-processed.

```bash
getnote kb lives vnrOAaGY
getnote kb lives vnrOAaGY --page 2
```

---

### Show live detail

```
getnote kb live <topic_id> <live_id>
```

Returns AI summary (`post_summary`) and full transcript (`post_media_text`).

```bash
getnote kb live vnrOAaGY live_abc123
getnote kb live vnrOAaGY live_abc123 -o json
```

---

### Show API quota usage

```
getnote quota
```

```bash
getnote quota
getnote quota -o json
```

---

## Agent Usage Notes

- Use `-o json` when parsing results programmatically.
- `kbs -o json` returns `{"success":true,"data":{"topics":[...],"total":N}}`
- `kbs-sub -o json` returns the same shape as `kbs -o json`.
- `kb <topic_id> -o json` returns `{"success":true,"data":{"notes":[...],"has_more":...}}`
- Get `topic_id` from `getnote kbs -o json` or `getnote kbs-sub -o json` → `data.topics[].topic_id` field (not `id`).
- `kb add` / `kb remove` accept multiple note IDs — prefer batching over multiple calls.
- `kb bloggers` → get `follow_id` → `kb blogger-contents` → get `post_id_alias` → `kb blogger-content` for full text.
- `kb lives` → get `live_id` → `kb live` for AI summary + transcript.
- `quota -o json` returns `{"success":true,"data":{"quotas":[{name,used,total,reset_time}]}}`
- Exit code `0` = success; non-zero = error. Error details go to stderr.
