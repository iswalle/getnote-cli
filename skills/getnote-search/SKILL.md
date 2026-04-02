---
name: getnote-search
version: 0.2.0
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
| `--kb` | — | Limit search to a knowledge base (topic_id) |
| `--limit` | 10 | Max results (max 10) |

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
- Get `topic_id` for `--kb` from `getnote kbs -o json`.
- Results are ranked by semantic relevance, not recency.
- Max `--limit` is 10; use `getnote notes` for broader browsing.
- Exit code `0` = success; non-zero = error. Error details go to stderr.
