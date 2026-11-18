# 上游 0.2.22 同步记录

## 范围

- 二开起点：`d7482320db55619291b1c74713302fae0c316ace`。
- 旧纯上游基线：`530c5e3a4f9c73ca2b0881191d0ef8e0d82c5484`（0.2.21）。
- 固定目标：`69d2fae91318359e4e25c0171eaf70364dad3629`，版本文件 `0.2.22`。
- 包含 v0.2.22 标签 `b0fa9f3e6eeab1968a53e1289b69daa4bb40ca2b` 及其后的 GPT-6.2 支持、流式转发性能更新；完整接入 193 个上游提交，没有排除或回退。
- 从 `origin/main` 创建 `sync/upstream-20261118` 和独立 worktree，使用保留历史的 merge。主线接入并验证祖先关系后才推进纯上游基线。
- 独立更新功能 PR #7 保持单独处理。本次仅更新代码分支，不部署。

## 定制与适配

- 保留暖色主题、字体及入口、被动通道监测卡片、供应商管理和空供应商筛选修复。
- OpenAI Fast 仍按策略处理后的出站请求档位计费；响应档位仅观察。接入上游强制 Ultra Fast 的模型/套餐门控，免费 Fast 仍不包含 Ultra Fast。
- 两处文本冲突均位于账号创建/编辑弹窗测试，保留供应商归属、统一 URL、默认并发和倍率同步测试，并保留上游透传模型过滤测试。
- 上游 GPT-6.2 文案更新后，一项既有文案测试仍要求旧提示含 `service_tier=ultrafast`；改为核对 Ultra Fast、GPT-6.2、Pro 200/500 及原有受支持模型，保持新版用户文案。
- 133 个不与上游重叠的定制路径在合并索引中与原主线 Git blob 一致；25 个重叠路径保留供应商、请求档位计费与 Go 1.27.4 接入。工作约定另作有意更新。
- Ent/Wire 重新生成后与合并结果一致。
- 保留全部已有 SQL 迁移的文件名与内容，新增 `251_group_force_openai_ultrafast.sql` 和 `251_group_usage_rollup_nonblocking_build.sql`，按完整文件名分别追踪。

## 验证与限制

- Node 24.13.0 / pnpm 9.15.9：冻结锁文件安装、lint、typecheck、生产构建通过。
- 前端现有全量测试首轮为 3,171 项通过、1 项过时文案断言失败；修正后仅重跑该文件，4 项全部通过，未重复全量测试。
- Go 1.27.4：`go build -tags embed ./cmd/server` 通过，包含已构建前端。
- 后端关键回归通过：`go test -tags=unit ./internal/service -run 'Supplier|ServiceTier|OpenAIFast|RecordUsage|GroupBills|OpenAIUltrafast|GPT62' -timeout=120s`。
- 已启动的全量 unit 检查中，handler、repository、server、middleware、routes、migrations 等包通过；按缩减测试范围的要求停止耗时的 service 全量检查，不能表述为全量 unit 通过。
- 首轮 Ent schema 检查受同时进行的代码生成干扰；生成完成后单独执行 `go test -tags=unit ./ent/schema` 通过。后续不要把 Ent 生成与依赖生成文件的测试并行运行。
- Windows 的 cgroup 符号链接测试因缺少创建链接权限失败，未修改测试或系统权限。
- Docker daemon 未运行，未执行本地数据库/Redis 集成验证，未新增容器、镜像或数据卷；按本次要求未扩展浏览器页面验收，也未新增测试用例。
- PR 及主线推送会触发现有 CI；本地验证不能代替其实际结果。未连接生产数据库或真实供应商。
