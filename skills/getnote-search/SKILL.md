---
name: getnote-search
version: 0.4.0
description: Semantic search across notes in Get笔记 via the getnote CLI
---

# getnote-search Skill

Semantic search across all notes or within a specific knowledge base.

## Prerequisites

- `getnote` CLI installed and authenticated (`getnote auth status` should show "Authenticated")

## Commands

### Search notes

```
getnote search <query> [--kb <topic_id>] [--limit <n>]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--kb` | — | Limit search to a knowledge base (`topic_id`) |
| `--limit` | 10 | Max results (max 10) |

Results are ranked by semantic relevance (high → low). Each result includes: `note_id`, `title`, `content` (excerpt), `score`, `created_at`, `note_type`.

> Note: `note_type` is one of `NOTE`, `FILE`, `BLOGGER`, `LIVE`, `URL`, `DEDAO`. `note_id` is only populated for `NOTE` type results; other types return an empty `note_id`.

```bash
# Search across all notes
getnote search "大模型 API"

# Search within a knowledge base
getnote search "RAG" --kb qnNX75j0

# Limit results + JSON output
getnote search "机器学习" --limit 5 -o json
```

---

## Agent Usage Notes

- Use `-o json` when parsing results programmatically.
- JSON response: `{"success":true,"results":[{"note_id":"...","title":"...","content":"...","score":0.95,"created_at":"...","note_type":"..."}]}`
- Note: `results` is at the top level, not nested under `data`.
- Get `topic_id` for `--kb` from `getnote kbs -o json` → `data.topics[].topic_id`.
- For `NOTE` type results, use `getnote note <note_id>` to get the full content.
- Max `--limit` is 10; use `getnote notes` for browsing without a query.
- Exit code `0` = success; non-zero = error. Error details go to stderr.
