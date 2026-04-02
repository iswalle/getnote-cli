---
name: getnote-search
version: 0.1.0
description: Semantic search across notes in Get笔记 via the getnote CLI
---

# getnote-search Skill

Semantic search across all notes or within a specific knowledge base in Get笔记.

## Prerequisites

- `getnote` CLI installed and authenticated (`getnote auth status` should show "Authenticated")
- API key configured via `getnote auth login --api-key <key>` or the `GETNOTE_API_KEY` environment variable

## Commands

### Search notes

```
getnote search <query> [--kb <topic_id>] [--limit <n>] [--output json|table]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--kb` | — | Limit search to a specific knowledge base (topic_id) |
| `--limit` | 10 | Max results to return (max 10) |
| `--output` | table | Output format: `table` or `json` |

**Examples:**
```bash
# Search across all notes
getnote search "大模型 API"

# Search within a specific knowledge base
getnote search "RAG" --kb qnNX75j0

# Limit results and get JSON output
getnote search "机器学习" --limit 5 --output json
```

---

## Agent Usage Notes

- Use `--output json` when parsing results programmatically.
- To search within a knowledge base, first get the `topic_id` via `getnote kb list --output json`.
- Results are ranked by semantic relevance, not recency.
- Max `--limit` is 10; for broader discovery use `getnote notes list` instead.
- Exit code `0` = success; non-zero = error. Error details go to stderr.
