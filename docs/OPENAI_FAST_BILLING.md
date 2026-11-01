# OpenAI Fast 计费定制

OpenAI/Codex 账号按策略处理后实际发给上游的请求 `service_tier` 确定计费档位，覆盖 API Key、OAuth 和 setup-token。响应中的 `service_tier` 继续原样返回并单独保留观察值，但不调整扣费档位。例如请求为 `priority`，响应为 `default` 或 `flex`，仍按请求的 Fast 档位计费；普通请求收到 `priority` 响应不会升级收费。

- 请求别名 `fast` 继续由转发层归一为 `priority`。强制 Fast 策略生效后按 Fast 档位处理；策略过滤掉该字段后按标准档位处理。WebSocket 每轮使用该轮策略处理后的请求档位。
- 分组的免费 Fast 继续生效：用量记录保留 Fast 档位和原价，实际扣费按标准价格及分组倍率计算。
- 该定制仅作用于 OpenAI 账号。Claude Fast 及其他提供商仍使用原有响应降档逻辑。模型、token 用量和价格计算规则不变。
- 没有新增数据库字段、迁移或 API 参数。

实现入口为 `backend/internal/service/service_tier_billing.go` 的 `ResolveOpenAIServiceTierBilling`，由 `openai_gateway_usage.go` 在记录用量和扣款前调用。转发结果中的 `ServiceTier` 是出站请求档位，`UpstreamResponseServiceTier` 是响应观察值；上游同步时需保留两者的区别，以及仅对 OpenAI 账号应用此规则的限制。

回归验证覆盖 `service_tier_billing_test.go` 的档位组合、`openai_gateway_record_usage_test.go` 的实际扣款与免费 Fast，以及现有 `openai_fast_service_tier_test.go` / `openai_fast_policy_ws_test.go` 的请求策略、响应透传与多轮 WebSocket 行为。在 `backend/` 运行 `go test -tags=unit ./internal/service ./internal/handler ./internal/server/middleware`，再按 CI 执行完整后端检查。
