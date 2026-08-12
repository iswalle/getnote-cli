---
name: getnote-kb
description: 查看和管理得到大脑的默认知识库、书籍、客户档案、团队知识库、文件夹、抖音博主订阅与直播，并把笔记准确归档到指定知识库和目录。
---

# 得到大脑知识库

通过官方 CLI 读取真实知识库、权限和目录后再操作；不能用名称猜 ID。

## 意图路由

| 意图 | 命令入口 |
|---|---|
| 自有/可管理知识库 | `getnote kbs` |
| 订阅知识库 | `getnote kbs-sub` |
| 知识库笔记 | `getnote kb <topic_id>` |
| 新建个人知识库 | `getnote kb create` |
| 加入笔记 | `getnote kb add` |
| 移出笔记 | `getnote kb remove` |
| 浏览文件夹 | `gnote kb dir` |
| 创建文件夹 | `gnote kb mkdir` |
| 重命名/移动文件夹 | `gnote kb mvdir` |
| 删除空文件夹 | `gnote kb rmdir` |
| 博主列表 | `getnote kb bloggers` |
| 博主内容列表 | `getnote kb blogger-contents` |
| 博主内容详情 | `getnote kb blogger-content` |
| 订阅抖音博主 | `getnote kb blogger-follow` |
| 直播列表 | `getnote kb lives` |
| 直播详情 | `getnote kb live` |
| 订阅直播 | `getnote kb live-follow` |

`gnote` 和短命令是稳定别名；旧环境没有别名时回退到 `getnote kb directories/directory-create/directory-update/directory-delete`。参数一律以目标命令 `--help` 为准。

## 选择知识库和权限

1. 先执行 `getnote kbs -o json`，保留全部真实 Scope：`DEFAULT`、`BOOKSPACE`、`CUSTOMER`、`TEAMSPACE`，不能只返回默认知识库。
2. 按名称和 `scope` 匹配；同名或用户意图不明确时让用户选择，不猜 `topic_id`。
3. 订阅知识库通常只读。团队知识库成员可以读取；只有接口返回具备维护权限的拥有者或管理员才能创建目录、加入笔记或订阅内容。
4. 普通成员写入失败时明确说明权限不足，不尝试绕过。当前不代替用户新建团队知识库。

## 文件夹和归档流程

1. 用户要求放入文件夹时，用 `gnote kb dir <topic_id> -o json` 读取根目录或指定目录。
2. 已有文件夹使用返回的真实 `directory_id`；缺失时先询问是否创建，再用 `mkdir`。
3. `getnote kb add` 同时传真实 `topic_id`、字符串 `note_id` 和 CLI 帮助中规定的目录参数。
4. 每批最多 20 条。移出笔记和删除目录必须先确认；删除目录还必须由 CLI/服务校验为空。
5. 移动或重命名时只改变用户指定项，未指定的名称或父目录保持不变。

## 博主和直播

1. 用户给出抖音主页并要求持续关注时，先确认目标知识库和写权限，再使用 `blogger-follow`；只是找某条内容时先查询，不创建订阅。
2. 列表先返回博主/直播名称、真实字符串 ID 和必要状态，选中后再读取完整内容。
3. 博主内容详情中的 `post_media_text` 才是完整原文，不用摘要冒充。

## 结果契约

- 列表命令：退出码 0 且 `success=true`；从 `data.topics[]`、`data.notes[]`、`data.bloggers[]`、`data.contents[]` 或 `data.lives[]` 读取实际项目和分页状态。
- 目录浏览：读取 `data.current_directory`、`data.directories[]`、`data.resources[]`、`data.total`；ID 全程按字符串。
- 创建/移动/删除目录、加入/移出笔记和订阅：只有退出码 0 且业务 `success=true` 才回复完成；回复实际知识库、目录或订阅对象。
- 权限不足、目录非空、批量超限等失败必须原样解释，不伪造降级成功；保留 `request_id` 和 `retryable`。
