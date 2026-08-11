---
name: getnote-search
description: 使用 getnote CLI 在全部笔记或指定知识库中进行语义搜索，并返回真实标题、字符串 ID 和笔记链接。
---

# 得到大脑搜索

使用 `getnote search` 执行语义搜索。运行前通过 `getnote search --help` 获取当前参数，不复制或猜测旧参数。

## 规则

- 机器调用添加 `-o json`，从 CLI 当前结构化结果读取搜索结果。
- 限定知识库时，先用 `getnote kbs -o json` 获取真实 `topic_id`。
- 保留 CLI 返回的标题、摘要、字符串 `note_id`、`note_type` 和 `note_url`。
- 非笔记类型结果可能没有 `note_id`；不要编造 ID。
- 用户只要求搜索时不要自动修改、移动或分享笔记。
- 需要普通笔记详情时，再调用 `getnote note`。
- 在群聊或共享会话中先展示必要的标题和链接，不主动展开私密笔记全文。
