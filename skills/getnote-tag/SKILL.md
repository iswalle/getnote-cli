---
name: getnote-tag
description: 使用 getnote CLI 查看、添加和按标签 ID 删除得到大脑笔记标签。
---

# 得到大脑标签

使用 `getnote tag --help` 获取当前子命令和参数。

## 路由

| 意图 | 命令入口 |
|---|---|
| 查看标签 | `getnote tag list` |
| 添加标签 | `getnote tag add` |
| 删除标签 | `getnote tag remove` |

## 规则

- 机器调用添加 `-o json`。
- 笔记 ID 和标签 ID 始终按字符串传递。
- 删除前先查看标签，并把真实标签 ID 传给 `tag remove`，不要传标签名。
- 系统标签不可删除；替换全部标签属于高风险操作，执行前向用户确认。
