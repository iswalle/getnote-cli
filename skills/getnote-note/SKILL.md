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
- 删除、覆盖正文、替换全部标签和公开分享前必须确认。
- 用户要求链接/录音原文时，先运行 `getnote note --help` 查看当前字段参数，不猜字段名。
