---
name: getnote-search
description: 使用 getnote CLI 在全部笔记或指定知识库中进行语义搜索，并返回真实标题、字符串 ID 和笔记链接。
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

Results are ranked by semantic relevance (high → low). Each result includes: `note_id`, `note_url`, `title`, `content` (excerpt), `score`, `created_at`, `note_type`. Preserve the returned `note_url` when presenting results; do not construct one yourself.

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
- JSON response: `{"success":true,"data":{"results":[{"note_id":"...","note_url":"...","title":"...","content":"...","score":0.95,"created_at":"...","note_type":"..."}]}}`
- Search results are read from `data.results`.
- Get `topic_id` for `--kb` from `getnote kbs -o json` → `data.topics[].topic_id`.
- For `NOTE` type results, use `getnote note <note_id>` to get the full content.
- Max `--limit` is 10; use `getnote notes` for browsing without a query.
- Exit code `0` = success; non-zero = error. Error details go to stderr.
