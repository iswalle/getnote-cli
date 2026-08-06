---
name: getnote-note
version: 0.5.0
description: Manage notes in Get笔记 via the getnote CLI
---

# getnote-note Skill

Save, list, view, update, and delete notes in Get笔记.

## Prerequisites

- `getnote` CLI installed and authenticated (`getnote auth status` should show "Authenticated")

## Commands

### Save a note

```
getnote save <url|text|image_path> [--title <title>] [--tag <tag>]... [--topic-id <topic_id>] [--parent-id <note_id>] [--idempotency-key <key>]
```

| Flag | Description |
|------|-------------|
| `--title` | Optional title |
| `--tag` | Tag to apply; may be repeated |
| `--topic-id` | Save directly into an owned DEFAULT / BOOKSPACE / CUSTOMER knowledge base |
| `--parent-id` | Create a child note; pass the snowflake ID as a decimal string |
| `--idempotency-key` | Stable 1-128 character request key; reuse it for retries of the same create request |

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

In `-o json` mode, silently polls and returns the final note JSON (including `title`, `content`/summary, `note_type`, `tags`, `created_at`, and the environment-correct `note_url`). Return that `note_url` to the user after a successful save; do not construct one yourself.

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
getnote notes [--cursor <cursor>] [--limit <n>] [--all]
```

The API fetches 20 notes per page. `--limit` returns the first N and rewrites `cursor` to the last visible note, so the next page does not skip data.

| Flag | Description |
|------|-------------|
| `--cursor` | Pagination cursor (the `cursor` value from the previous page) |
| `--limit` | Number of notes to return from the fetched page |
| `--all` | Fetch all notes (auto-paginate, streams output) |

```bash
getnote notes
getnote notes --all
getnote notes --cursor 1914025811879486080
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
- JSON responses preserve the API envelope `{"success":true,"data":{...}}`.
- Link/image saves read the async ID from `data.tasks[0].task_id`; task polling only treats `error_msg` as meaningful when `status=failed`.
- `notes` fetches 20 from the API and may apply a smaller local `--limit`; paginate with the returned string `cursor`.
- Note IDs are int64 — always handle as strings to avoid precision loss in JavaScript.
- HTTP 200 with `success:false` is still an error. The CLI exits non-zero and preserves `code`, `reason`, `retryable`, `field`, `constraint`, `expected_type`, and `request_id`.
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
