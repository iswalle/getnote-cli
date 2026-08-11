---
name: getnote-kb
description: 使用 getnote CLI 查看和管理自有或订阅知识库、书籍、客户档案、博主内容与直播。
---

# 得到大脑知识库

参数和子命令统一通过 `getnote kbs --help`、`getnote kbs-sub --help` 或 `getnote kb --help` 获取。

## 路由

| 意图 | 命令入口 |
|---|---|
| 查看自有知识库 | `getnote kbs` |
| 查看订阅知识库 | `getnote kbs-sub` |
| 查看知识库内笔记 | `getnote kb` |
| 新建知识库 | `getnote kb create` |
| 加入或移出笔记 | `getnote kb add` / `getnote kb remove` |
| 查看订阅博主及内容 | `getnote kb bloggers` / `getnote kb blogger-contents` / `getnote kb blogger-content` |
| 查看或订阅直播 | `getnote kb lives` / `getnote kb live` / `getnote kb live-follow` |

## 规则

- 机器调用添加 `-o json`。
- 自有列表必须保留默认、书籍和客户档案等全部 Scope，不能只返回默认知识库。
- 保存或搜索到指定知识库时使用 CLI 返回的真实 `topic_id`，不用名称代替。
- ID 始终按字符串传递。
- 订阅知识库通常只读；执行写操作前确认权限。
- 移出笔记属于高风险操作，执行前向用户确认。
