# 上游 v0.2.24 同步记录

- 二开起点为 `7d65b57f303d6babb0169d5fb1bd8791ca08671a`，对应 `v0.2.23-custom.3`。
- 旧纯上游基线为 `5af9afb588c4b014b3df9db7d4073af1f8ae01b6`。完整合入至官方标签 `v0.2.24` 的 110 个提交，目标为 `5c2982a81b46e2142d0143d2f83c1ce9c4da1d24`。标签后上游主线的单独版本文件更新由二开版本 `0.2.24-custom.0` 对应。
- 上游新增 Exa 网页搜索、模型 ID 复制和 Claude 窗口费用估算，并包含网关参数校验、OAuth 和流式响应修复。此次没有新的 SQL 迁移，现有迁移文件和内容保持不变。[官方发布说明](https://github.com/Wei-Shaw/sub2api/releases/tag/v0.2.24)
- Bedrock 账号测试保留定制质量探针的题目，同时为不支持 `temperature` 的模型省略该参数。后台创建用户的默认平台配额接入保留了抽奖服务依赖，服务入口由 Wire 重新生成。
- 上游标签未改动 OpenAI WebSocket 中继的每轮输出状态。保留 `v0.2.23-custom.1` 的每轮独立状态修复、Fast 请求档位计费、暖色主题、供应商管理、工单、指定用户公告、抽奖、账号质量规则与渠道监控 V2。
- 补齐管理密钥目录中抽奖读取与管理权限的中英文标签，使用已有财务分类。
- 合并检查及发布构建结果见 [GitHub Actions](https://github.com/FrankJunhaoYe/franklybuilds-sub2api/actions)。
