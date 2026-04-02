---
name: getnote-tag
version: 0.1.0
description: Manage note tags in Get笔记 via the getnote CLI
---

# getnote-tag Skill

Add, list, and remove tags on notes in Get笔记.

## Prerequisites

- `getnote` CLI installed and authenticated (`getnote auth status` should show "Authenticated")
- API key configured via `getnote auth login --api-key <key>` or the `GETNOTE_API_KEY` environment variable

## Commands

### List tags on a note

```
getnote tag list <note_id> [--output json|table]
```

**Examples:**
```bash
getnote tag list 1896830231705320746
getnote tag list 1896830231705320746 --output json
```

Returns tag list including tag IDs (needed for `tag remove`).

---

### Add a tag to a note

```
getnote tag add <note_id> <tag>
```

**Examples:**
```bash
getnote tag add 1896830231705320746 工作
getnote tag add 1896830231705320746 阅读
```

---

### Remove a tag from a note

```
getnote tag remove <note_id> <tag_id>
```

> ⚠️ Requires **tag ID** (not tag name). Use `getnote tag list <note_id>` to find the tag ID first.
> System tags cannot be removed.

**Examples:**
```bash
# First, find the tag ID
getnote tag list 1896830231705320746 --output json

# Then remove by tag ID
getnote tag remove 1896830231705320746 123
```

---

## Agent Usage Notes

- Use `--output json` on `tag list` to get tag IDs for use with `tag remove`.
- `tag remove` takes a **tag ID** (integer), not a tag name — always call `tag list` first if you only have the name.
- System tags cannot be deleted; the CLI will return an error if attempted.
- Exit code `0` = success; non-zero = error. Error details go to stderr.
