---
name: getnote-search
description: 在得到大脑的全部笔记或指定知识库中按自然语言进行语义搜索，返回真实标题、摘要、字符串 ID 和可打开的笔记链接。
---

# 得到大脑搜索

## 执行流程

1. “搜索、找找、关于某主题”使用 `getnote search`；“最近有哪些笔记”应使用笔记 Skill 的 `getnote notes`，不要混用。
2. 不限定知识库时直接搜索。限定知识库时先执行 `getnote kbs -o json`，根据名称和 `scope` 取得真实 `topic_id`；同名时让用户选择。
3. 用 `getnote search --help` 获取当前参数，再执行 `getnote search <query> -o json`；不要复制或猜测旧参数。
4. 用户只要求搜索时不自动修改、移动、分享或创建笔记。
5. 用户选中结果后，再交给笔记 Skill 读取详情、原文或转写。

## 结果契约

- 命令退出码必须为 0，业务结果必须 `success=true`。
- 从 `data.results[]` 读取 `title`、`content`、`note_type`、字符串 `note_id`、`note_url` 和 `score`。
- 非笔记类型结果可能没有 `note_id` 或 `note_url`，如实说明类型，不编造 ID/链接。
- 没有结果时明确说“未找到匹配笔记”，可以建议用户换关键词或取消知识库限制，不声称索引故障。
- 有结果时按相关性顺序返回精简编号列表：标题 + 必要摘要 + 真实链接；控制在用户要求数量内。
- 失败时说明 `error.message/reason`、是否可重试和 `request_id`。群聊中不主动展开私密全文。
