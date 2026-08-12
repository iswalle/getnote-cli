---
name: getnote-note
description: 使用 getnote CLI 保存文字、链接、图片和长笔记，查看、更新、删除或分享得到大脑笔记。
---

# 得到大脑笔记

把笔记意图路由到 CLI；参数和输出以对应命令的 `--help` 与 `-o json` 结果为准。

## 路由

| 意图 | 命令入口 |
|---|---|
| 保存文字、链接或本地图片 | `getnote save` |
| 查询异步保存任务 | `getnote task` |
| 查看最近笔记 | `getnote notes` |
| 查看笔记详情或指定字段 | `getnote note` |
| 读取链接/文字原文 | `getnote note original` |
| 读取录音、会议或课堂转写 | `getnote note transcript` |
| 列出图片、音频和文件附件 | `getnote note attachments` |
| 读取录音或会议时间线 | `getnote note timeline` |
| 读取录音快捷笔记 | `getnote note quick-note` |
| 读取会议总结中的派生待办 | `getnote note todos` |
| 修改笔记 | `getnote note update` |
| 删除笔记 | `getnote note delete` |
| 创建公开分享 | `getnote note share` |

## 规则

- 机器调用添加 `-o json`。
- 长文本使用 `getnote save --content-file` 或 `getnote save --stdin`。
- 笔记、任务、父笔记和游标 ID 始终按字符串原样传递。
- 链接和图片保存等待 CLI 返回最终状态；只返回 CLI 给出的 `note_url`。
- 同一次保存重试复用原 `--idempotency-key`；状态不确定时先核实，禁止重复创建。
- 不擅自补充知识库、父笔记或标签。
- 删除、覆盖正文、替换全部标签和公开分享前必须确认；确认后按 `--help` 使用 CLI 的 `--yes` 放行。
- 用户要求链接、文字原文时使用 `getnote note original`；要求录音、会议或课堂转写时使用 `getnote note transcript`，不要把 AI 摘要冒充原文。
- 会议待办使用 `getnote note todos`，并保留其 `source`；这是从总结明确章节规则解析的派生结果，不要描述成上游原生待办，也不要自由补写。
- 在群聊或共享会话中不主动展开私密笔记全文；先确认请求者和展示范围。
