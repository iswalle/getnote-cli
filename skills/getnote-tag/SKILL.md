---
name: getnote-tag
description: 查看得到大脑笔记已有标签，为笔记添加标签，或按真实标签 ID 安全删除标签，并避免误删系统标签或覆盖其他标签。
---

# 得到大脑标签

## 执行流程

1. 先用 `getnote tag list <note_id> -o json` 读取当前标签；笔记 ID 和标签 ID 始终按字符串。
2. 添加使用 `getnote tag add <note_id> <tag> -o json`，只增加用户指定项，不覆盖其他标签。
3. 删除前从列表中取得真实 `tag_id`，再运行 `getnote tag remove <note_id> <tag_id> -o json`；不能把标签名当 ID。
4. 系统标签不可删除。替换全部标签属于覆盖性操作，必须先确认并按 `getnote note update --help` 使用对应参数。

## 每条命令的结果与回复格式

| 命令 | 成功结果必须包含 | 回复规则 |
|---|---|---|
| `getnote tag list <note_id> -o json` | `success=true`、`data.note_id`、`data.tags[].id/name/type` | 列出真实标签；删除时只能使用返回的 `tag_id`。 |
| `getnote tag add <note_id> <tag> -o json` | `success=true`、`data.note_id`、更新后的 `data.tags[]` | 说明已新增的标签，不暗示替换了旧标签。 |
| `getnote tag remove <note_id> <tag_id> -o json` | `success=true`、`data?` | 仅确认指定标签已删除；需要展示剩余标签时再运行 `tag list` 核验。 |

失败时说明 `error.message/reason`、是否可重试和 `request_id`；系统标签、无权限或标签不存在都不能说成已删除。
