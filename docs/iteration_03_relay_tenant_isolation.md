# Iteration 3: Relay Tenant Isolation

## Relay Isolation 修复点

本轮仅修复 relay runtime 中的 tenant 隔离与相关缓存隔离，不调整计费规则、权限模型、迁移框架或前端行为。

### 1. 严格 Relay Tenant Context

- `model/tenant_scope.go` 新增 `RelayTenantScopeFromContext`。普通 relay 请求必须已有非零 `tenant_id` Context，否则返回错误；不再在选渠时将缺失 tenant 静默解释为 tenant 1。旧用户数据的兼容仍由 `model/user_cache.go` 的 `UserBase.WriteContext` 在认证成功后完成。
- `middleware/distributor.go` 在任何渠道选择前取得严格 relay scope；缺失 scope 时直接停止请求。因此首选渠道、specific channel 与 affinity 入口都不会在未知 tenant 下运行。
- `middleware/auth.go` 中，后台会话认证读取 User cache 失败时改为中止请求，不再写入 tenant 1 后继续执行。`TokenOrUserAuth` 的 session 分支也会写入用户 ownership Context，供视频代理验证渠道归属。

### 2. Ability 查询 Tenant Isolation

- `model/ability.go` 的 `GetChannel`、`getChannelQuery` 与 `getPriority` 新增 `TenantScope` 参数应用点。数据库选渠时，普通 tenant 的最大优先级、retry 优先级及 weighted 候选均限定在 `abilities.tenant_id` 内。
- `model/ability.go` 的 `GetGroupEnabledModels` 与 `GetEnabledModels` 支持 tenant scope；`controller/model.go` 的 relay 模型列表使用严格 relay scope，避免 token 用户看到只属于其他 tenant 的 Ability 模型。
- `model/channel_satisfy.go` 的 affinity 可用性检查同时接收 tenant scope；无论内存缓存或数据库路径，preferred channel 只能通过本 tenant 的 group/model 可用性判断。

### 3. Channel 查询与渠道选择 Tenant Isolation

- `model/channel.go` 新增 `GetChannelByIdScoped`。specific channel、历史 Task 锁定渠道、Midjourney 后续操作等直接按 ID 取渠道的 relay 路径，均可在读取阶段验证 tenant。
- `middleware/distributor.go` 使用 `GetChannelByIdScoped` 与 `CacheGetChannelScoped` 校验 specific channel 和 affinity preferred channel；普通 admin 即使可指定 channel id，也不能指定其他 tenant 的渠道。
- `service/channel_select.go` 的首选与 retry 都将同一个 `TenantScope` 传至 `model.GetRandomSatisfiedChannel`。`auto` 分组下如果 scoped 查询返回错误，立即返回失败，而不是忽略错误并继续 fallback 到下一组。
- `controller/relay.go` 将请求 scope 注入同步 relay 与 Task relay 的 retry 参数，使重试仍限定在首次请求的 tenant 范围内。

### 4. 历史任务复用路径

- `relay/relay_task.go` 的 remix / continuation 使用 `GetChannelByIdScoped` 读取原任务渠道；视频实时 fetch 也在读取上游渠道前验证请求 tenant scope。
- `relay/mjproxy_handler.go` 中已认证的 image-seed、变换/重绘原任务渠道复用以及自动禁用读取路径，均改为 scoped channel 查询；查询失败时明确返回错误，不继续使用空渠道或其他渠道。
- `controller/video_proxy.go` 的视频内容代理按当前请求 tenant 读取 Task 对应 Channel，防止损坏或遗留任务引用其他 tenant 的渠道凭证。

## Cache Key 风险与修复

### Channel Selection Memory Cache

- 风险来源：原 `model/channel_cache.go` 的内存索引仅按 `group -> model` 保存 channel id；两个 tenant 具有同名 group/model 时，普通 tenant 可能随机命中另一 tenant Channel。
- 修复方式：索引改为 `tenant_id -> group -> model -> channel ids`。`GetRandomSatisfiedChannel` 在普通 scope 下仅读取自身 tenant 分桶；仅当 `TenantScope.IsRoot` 为 true 时才合并所有 tenant 候选。
- 补充防线：`CacheGetChannelScoped` 对按 ID 命中的缓存 Channel 再校验 tenant，覆盖 affinity 和视频代理等不经过随机索引的路径。

### Channel Affinity Cache

- 风险来源：`service/channel_affinity.go` 原 affinity key 不包含 tenant；不同 tenant 使用相同 affinity 值时会共享或覆盖 preferred channel id。
- 修复方式：relay runtime 生成 affinity key 时追加 `tenant=<tenant_id>`；root 使用 `tenant=root`。此外 `middleware/distributor.go` 对命中的 preferred channel 再执行 scoped cache 读取与 scoped group/model 校验。

## Root Bypass 规则

- `model/tenant_scope.go`：只有 `TenantScope.IsRoot` 为 true 时，`Apply` 不添加 tenant 过滤，`AllowsTenant` 允许跨 tenant。
- `middleware/auth.go`：session root 原有 `role` Context 继续生效；API token owner 为 root 时，`SetupContextForToken` 显式写入 `role=root`，因此 root token 在 relay 选渠中可使用平台级跨 tenant 能力。
- 普通 admin token 不获得 root 标记。其 specific channel、affinity、自动选渠与 retry 全部受自身 tenant scope 约束。
- `model/channel_cache.go`：只有 root scope 会合并不同 tenant 的缓存候选；这是缓存层唯一的跨 tenant 选渠 bypass。

## 已 Tenant 化的查询路径

| 场景 | 文件路径 | 隔离方式 |
| --- | --- | --- |
| relay 请求进入选渠 | `middleware/distributor.go`, `model/tenant_scope.go` | 强制取得严格 tenant scope，缺失即报错 |
| Ability 最大优先级和 retry 优先级 | `model/ability.go` | `abilities.tenant_id` scope 条件 |
| DB weighted channel lookup | `model/ability.go`, `model/channel.go` | Ability 与最终 Channel 均应用同一 scope |
| 内存随机选渠 | `model/channel_cache.go` | tenant 分桶索引 |
| affinity preferred channel | `service/channel_affinity.go`, `middleware/distributor.go`, `model/channel_satisfy.go` | tenant-safe key + scoped channel/ability 校验 |
| specific channel | `middleware/distributor.go`, `model/channel.go` | 按 ID scoped 读取 |
| 同步 relay retry | `controller/relay.go`, `service/channel_select.go` | retry 参数携带同一 scope |
| Task submit retry / 原任务渠道复用 | `controller/relay.go`, `relay/relay_task.go` | retry scope + scoped locked channel |
| Midjourney 已认证后续操作 | `relay/mjproxy_handler.go` | 原任务 Channel scoped 读取 |
| 视频内容代理 | `controller/video_proxy.go`, `middleware/auth.go` | 请求 ownership Context + scoped cache 读取 |
| relay 模型可见列表 | `controller/model.go`, `model/ability.go` | Ability 模型查询使用 relay scope |

## 日志、任务与 Quota Ownership 一致性

- `relay/common/relay_info.go` 在认证 Context 中记录请求 ownership，`model/task.go` 已在上一轮由该 `RelayInfo` 写入 Task ownership；本轮选中的 Channel 现在也受同一请求 scope 限制。
- `model/log.go` 的 `RecordConsumeLog` 与 `RecordErrorLog` 使用当前 Gin Context 写入 ownership；同步 relay 消耗路径由 `service/text_quota.go`、`service/quota.go` 和 `service/task_billing.go` 使用同一请求的 `RelayInfo.ChannelId` 与用户信息记录额度及日志。
- 因此对本轮已覆盖的同步请求与 Task 提交路径，日志/Task 的 tenant ownership 与可使用的 Channel tenant 不再允许跨 tenant 混用。

## 仍需人工复核的路径

- `relay/mjproxy_handler.go` 的 `RelayMidjourneyImage` 是无认证公开图片转发入口，且会根据任务 Channel 读取代理设置；其公开访问模型及是否需要 tenant 授权需要人工复核，本轮未改变公开 URL 契约。
- `controller/midjourney.go` 与 `service/task_polling.go` 的后台轮询基于历史 Task/Midjourney 数据读取 Channel，没有请求 Context。新数据已由 ownership closure 与本轮选渠隔离保证来源一致，但已有脏数据如何阻断或修复需要专项策略。
- `relay/mjproxy_handler.go` 的后续动作会复用同 tenant 的原始渠道；实际执行渠道与部分计量展示字段是否需要统一为原始渠道，需要人工复核。本轮只保证不跨 tenant。
- `model/pricing.go` 调用全局 Ability 数据生成价格相关信息，不属于 relay channel selection；是否需要按 tenant 建立定价隔离属于计费设计问题，本轮按禁止修改 billing 的范围未处理。
- `model/log.go` 的 `LogQuotaData` 与 `pkg/perf_metrics` 仍按既有设计运行；它们不参与渠道选择，且本轮范围明确不修改聚合指标存储。

## 修改文件

| 文件路径 | 目的 |
| --- | --- |
| `model/tenant_scope.go` | 严格 relay tenant scope |
| `model/ability.go` | Ability 模型、优先级、weighted 查询应用 scope |
| `model/channel.go` | scoped channel ID 读取 |
| `model/channel_cache.go` | tenant-safe 渠道选择缓存与 scoped cache lookup |
| `model/channel_satisfy.go` | affinity 可用性查询应用 scope |
| `model/user.go` | 判定 root token owner 以保留 root bypass |
| `middleware/auth.go` | 禁止认证失败落入 tenant 1，传播 session/root token Context |
| `middleware/distributor.go` | 首选、specific 与 affinity 选渠应用严格 scope |
| `service/channel_select.go` | retry/fallback 传递 scope，并禁止忽略隔离错误 |
| `service/channel_affinity.go` | affinity cache key 纳入 tenant |
| `controller/relay.go` | retry 参数传播 tenant scope |
| `controller/model.go` | relay 模型列表限定 tenant Ability |
| `controller/video_proxy.go` | Task 视频代理渠道 scoped 读取 |
| `relay/relay_task.go` | Task 原渠道与实时 fetch scoped 读取 |
| `relay/mjproxy_handler.go` | 已认证 Midjourney 原渠道 scoped 读取 |
| `model/relay_tenant_isolation_test.go` | 验证 DB 与内存缓存选渠隔离及 root bypass |
| `controller/model_list_test.go` | 模拟已认证 tenant Context 的既有模型列表测试 |

## 下一轮建议

- 针对历史 `Task`/`Midjourney` 与 `Channel` ownership 不一致的数据建立检测和修复方案，再决定后台轮询遇到不一致时的隔离失败处理。
- 审计公开资源代理与 callback 的授权边界，特别是无认证 Midjourney 图片转发入口。
- 在不改变 billing 语义的独立迭代中，评估异步退款/补扣及统计聚合是否需要携带并校验 tenant ownership。

## Strict Relay Scope Review Fix

- Review 发现 `controller/relay.go` 的同步 `Relay()` 与异步 `RelayTask()` 在构造 `service.RetryParam` 时仍调用 `model.TenantScopeFromContext(c)`。该 helper 会把缺失的普通 tenant context 规范化为 tenant 1，因此在未来出现绕过 `middleware/distributor.go` 的 retry / fallback 调用路径时，可能静默选取 tenant 1 的渠道。
- 本补丁在 `controller/relay.go` 增加统一的 `newRelayRetryParam`，同步 relay 与 Task relay 在进入 retry / fallback / channel selection 前均通过 `model.RelayTenantScopeFromContext(c)` 获取一次严格 scope 并复用到全部重试轮次。普通请求缺少 `tenant_id` 时立即失败，不再产生 tenant 1 fallback；root 仍由 `RelayTenantScopeFromContext` 保持平台级 bypass。
- 回归覆盖包括：`controller/relay_tenant_scope_test.go` 验证 retry 参数构造拒绝缺失 tenant context 且保留 root；`model/relay_tenant_isolation_test.go` 验证 specific channel 的 scoped 读取拒绝 tenant A 指向 tenant B 渠道；`service/channel_affinity_template_test.go` 验证 tenant A/B 使用相同 affinity value 时不会互相命中缓存渠道。

## 验证

- 新增 `model/relay_tenant_isolation_test.go`，覆盖缺失 tenant context 会被拒绝、普通 tenant 无法从 DB Ability 查询或内存 Channel cache 选择另一 tenant Channel，以及 root 可跨 tenant 选择。
- 已执行 `go test ./model ./controller ./service`，通过。
