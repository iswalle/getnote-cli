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

## 结果契约

- 标签列表成功返回 `success=true` 和 `data.note_id/data.tags[]`；展示每项真实 `id/name/type`。
- 添加成功必须退出码 0、业务 `success=true`，并从 `data.note_id` 和 `data.tags[]` 展示更新后的标签。
- 删除成功必须退出码 0、业务 `success=true`；需要时重新读取列表核验。
- 失败时说明 `error.message/reason`、是否可重试和 `request_id`；系统标签、无权限或标签不存在都不能说成已删除。
