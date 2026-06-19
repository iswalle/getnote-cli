---
name: getnote-note
version: 0.4.0
description: Manage notes in Get笔记 via the getnote CLI
---

# getnote-note Skill

Save, list, view, update, and delete notes in Get笔记.

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

- URL (`http://` or `https://`) → link note:
  - **Share link** (`biji.com/note/share_note/*` or `d.biji.com/*` short link) → **sync**, returns `note_id` directly, no polling needed
  - **Internal note link** (`biji.com/note/{note_id}`) → this is the format for linking to another note inside note content; use it in `content` field when referencing other notes. If the current note will be shared publicly, prefer using the referenced note's share link (`getnote note share <id>`) instead of the internal link format
  - **Other URLs** → async, auto-polls until done
- Local image path → image note (async, auto-polls until done)
- Otherwise → text note (sync)

```bash
getnote save https://example.com --title "Great article"
getnote save "Remember to review the docs" --tag work --tag important
getnote save ./screenshot.png --title "Design mockup"
```

In `-o json` mode, silently polls and returns the final note JSON (including `title`, `content`/summary, `note_type`, `tags`, `created_at`).

---

### Track save task

```
getnote task <task_id>
```

Manually check progress of an async save task.

```bash
getnote task task_xyz789 -o json
```

Returns `status` (`pending` / `processing` / `success` / `failed`) and `note_id` when done.

---

### List recent notes

```
getnote notes [--since-id <id>] [--all]
```

Returns 20 notes per page (fixed). No `--limit` flag.

| Flag | Description |
|------|-------------|
| `--since-id` | Pagination cursor (last note ID seen) |
| `--all` | Fetch all notes (auto-paginate, streams output) |

```bash
getnote notes
getnote notes --all
getnote notes --since-id 1234567890
getnote notes -o json
```

**Note types**: `plain_text` / `img_text` / `link` / `audio` / `meeting` / `local_audio` / `internal_record` / `class_audio` / `recorder_audio` / `recorder_flash_audio`

---

### Get note details

```
getnote note <id> [--field <field>]
```

Returns full note including content, tags, attachments. Use `--field` to extract a single value.

| `--field` values | Description |
|------|-------------|
| `id` | Note ID |
| `title` | Title |
| `content` | Content / AI summary |
| `type` | Note type |
| `created_at` | Creation time |
| `updated_at` | Last updated time |
| `url` | Source URL (link notes) |
| `excerpt` | Excerpt |
| `web_content` | Full web page content (link notes only) |
| `audio_original` | 录音笔记的转写原文（`audio` 类型笔记专用，非 AI 总结） |
| `source` | Note source (e.g. `openapi`, `manual`) |
| `tags` | Comma-separated tag names |

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
| `--tag` | Comma-separated tags — **replaces all existing tags** |

```bash
getnote note update 1234567890 --title "Updated title"
getnote note update 1234567890 --tag "work,important"
```

> ⚠️ `--tag` replaces all tags. For partial tag changes use `getnote tag add/remove`.
> ⚠️ Content update only works on `plain_text` notes.

---

### Delete a note

```
getnote note delete <id> [-y]
```

Moves note to trash.

```bash
getnote note delete 1234567890 -y
```

---

### Share a note

```
getnote note share <id> [--exclude-audio]
```

Generates a public share link for a note. Idempotent — calling multiple times returns the same URL.

```bash
getnote note share 1234567890
getnote note share 1234567890 --exclude-audio
getnote note share 1234567890 -o json
```

Returns: `share_url` (e.g. `https://biji.com/note/share_note/rBzdMlXrzgYVM`)

---

## Agent Usage Notes

- Use `-o json` when parsing responses programmatically.
- All JSON responses follow `{"success":true,"data":{...}}` structure, **except**:
  - `save` (text): returns `{"note_id":"..."}` directly
  - `save` (share link): returns `{"note_id":"...","title":"...","created_at":"...","updated_at":"..."}` directly
  - `save` (regular link/image): returns `{"data":{"tasks":[{"task_id":"..."}],...}}`
  - `task`: returns `{"success":true,"data":{"status":"...","note_id":"..."}}`
- `notes` list returns **20 per page** (no `--limit`); paginate with `--since-id`.
- Note IDs are int64 — always handle as strings to avoid precision loss in JavaScript.
- Exit code `0` = success; non-zero = error. Error details go to stderr.

### 字段语义提示（"原文" vs AI 总结）

不同笔记类型的"原文"字段不同，`content` 通常是 AI 总结而非原文。用户要求"读原文"时，先用 `getnote note <id> -o json` 查看 `note_type`，再按下表选择对应字段：

| 笔记类型 | 原文字段 | AI 总结字段 |
|---------|---------|------------|
| 普通文字笔记 | `content` | `content` |
| 链接/网页笔记 | `web_content` | `content` |
| 录音笔记 | `audio_original` | `content` |
| 知识库博主内容 | `post_media_text`（via `kb blogger-content`）| `content` |
| 知识库直播 | `post_media_text`（via `kb live`）| `post_summary` |
