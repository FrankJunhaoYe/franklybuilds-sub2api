# OpenAI Fast 计费定制

OpenAI/Codex 账号按策略处理后实际发给上游的请求 `service_tier` 确定计费档位，覆盖 API Key、OAuth 和 setup-token。响应中的 `service_tier` 继续原样返回并单独保留观察值，但不调整扣费档位。例如请求为 `priority`，响应为 `default` 或 `flex`，仍按请求的 Fast 档位计费；普通请求收到 `priority` 响应不会升级收费。

- 请求别名 `fast` 继续由转发层归一为 `priority`。强制 Fast 策略生效后按 Fast 档位处理；策略过滤掉该字段后按标准档位处理。WebSocket 每轮使用该轮策略处理后的请求档位。
- 分组的免费 Fast 继续生效：用量记录保留 Fast 档位和原价，实际扣费按标准价格及分组倍率计算。
- 该定制仅作用于 OpenAI 账号。Claude Fast 及其他提供商仍使用原有响应降档逻辑。模型、token 用量和价格计算规则不变。
- 没有新增数据库字段、迁移或 API 参数。

## 0.2.23 倍率复核

- Fast（`priority` / `fast`）和 Ultra Fast（`ultrafast`）分别使用渠道的 `fast_multiplier` 和 `ultrafast_multiplier`，不互相借用。分组倍率、用户专属倍率及高峰倍率继续按原有规则叠加。
- 渠道 Ultra Fast 倍率未设置时沿用模型默认值：GPT-6.2 Sol / Astra 在 2027-03-01 08:00 UTC（北京时间 16:00）前为 3 倍，之后为 6 倍；GPT-6 Astra / GPT-6.1 Sol 为 6 倍。GPT-6.2 Fast 默认 2 倍。显式渠道倍率覆盖模型默认倍率，包括促销期切换。
- 「免费 Fast」只把 Fast 实际扣费改为标准价，不免除 Ultra Fast 加价。强制档位仍按 Ultra Fast > Fast > 客户端的顺序处理，并执行模型及订阅套餐门控。
- 上游新增 `252_channel_pricing_ultrafast_multiplier.sql`，添加可空且必须大于零的渠道 Ultra Fast 倍率；旧记录保持 NULL 并使用模型默认值。上文“没有新增数据库字段”指最初的请求档位计费定制，不包含本次上游功能。
- 本轮复用现有请求/响应档位矩阵、实际扣款、免费 Fast、GPT-6.2 促销切换、渠道两条计费路径及渠道表单测试，防止合并时丢失请求档位定制。

实现入口为 `backend/internal/service/service_tier_billing.go` 的 `ResolveOpenAIServiceTierBilling`，由 `openai_gateway_usage.go` 在记录用量和扣款前调用。转发结果中的 `ServiceTier` 是出站请求档位，`UpstreamResponseServiceTier` 是响应观察值；上游同步时需保留两者的区别，以及仅对 OpenAI 账号应用此规则的限制。

回归验证覆盖 `service_tier_billing_test.go` 的档位组合、`openai_gateway_record_usage_test.go` 的实际扣款与免费 Fast，以及现有 `openai_fast_service_tier_test.go` / `openai_fast_policy_ws_test.go` 的请求策略、响应透传与多轮 WebSocket 行为。在 `backend/` 运行 `go test -tags=unit ./internal/service ./internal/handler ./internal/server/middleware`，再按 CI 执行完整后端检查。
