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

## 命令结果与回复格式

| 命令 | 成功结果必须包含 | 回复规则 |
|---|---|---|
| `getnote search <query> -o json` | `success=true`、`data.results[].title/score`，笔记结果还应有 `note_id/note_url`，可选 `content/note_type` | 按相关性返回编号列表：标题 + 必要摘要 + 真实链接。空 `results[]` 是“未找到”，不是失败；非笔记结果没有 ID/链接时如实说明。 |

失败时说明 `error.message/reason`、是否可重试和 `request_id`。群聊中不主动展开私密全文。
