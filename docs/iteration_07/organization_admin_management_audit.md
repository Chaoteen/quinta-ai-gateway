# Iteration 7.5 Organization Admin Management Audit

审计日期：2026-06-01

审计范围：`router/api-router.go` 及其关联 controller/model 中仍由 `AdminAuth`、`RootAuth`、`RoleAuth` 保护的管理路由。本文只做权限与迁移适配性审计，不包含代码改动建议的实现。

## 1. 当前状态

当前 RBAC 已经从 legacy `role` 逐步迁移到 `role_key`：

- `RootAuth()` 仍代表平台 root 能力，适用于跨租户、全局配置、平台目录写入、系统维护、部署管理等操作。
- `AdminAuth()` 仍是 legacy 管理边界，语义上等价于 legacy admin 及 root；在多租户语义下，多数应继续拆分为 `tenant_admin` 或保留 root。
- `RoleAuth(...)` 已用于一批只读路由，按 `tenant_admin`、`organization_admin`、`finance`、`ops`、`auditor` 等角色开放。
- `organization_admin` 已完成用户只读和运营只读的第一批迁移，且无组织归属时 fail closed。
- `AccessScope` 已支持 `tenant_id`、`organization_id`、`department_id` 字段，但 `organization_admin` 当前只自动带入组织维度，尚未启用部门维度管理入口。
- `Organization`、`Department` model 已存在，当前 `api-router.go` 中没有独立的组织管理/部门管理 CRUD 路由。

已完成迁移的关键控制面：

- 用户只读：`GET /api/user/`、`GET /api/user/search`、`GET /api/user/:id`
- 组织运营只读：`GET /api/log/`、`GET /api/user/topup`、`GET /api/redemption/`、`GET /api/mj/`、`GET /api/task/`

仍需重点保持谨慎的控制面：

- 用户写操作、Passkey/2FA 管理、OAuth 绑定解除
- 渠道写操作、渠道密钥读取、upstream 同步、test/balance 操作
- 计费计划、订阅绑定、兑换码写操作、充值订单完成
- 模型/供应商目录写操作
- 系统配置、性能维护、ratio sync、日志清理、quota data、部署管理

## 2. 路由分类表

### A. 未来适合 organization_admin 的路由

这些路由未来可以考虑开放给 `organization_admin`，但前提是 controller/model 已按 `organization_id` 和必要的 `department_id` 做完整作用域校验，且写操作有明确的组织内权限模型。

| 范围 | 路由 | 当前权限 | 当前判断 | 迁移前置条件 |
| --- | --- | --- | --- | --- |
| 用户管理 | `POST /api/user/` | `AdminAuth` | 可作为组织内创建成员候选 | 必须强制写入当前组织，禁止指定其他租户/组织，禁止创建管理员级角色，限制 quota/group/subscription 字段 |
| 用户管理 | `PUT /api/user/` | `AdminAuth` | 可作为组织内成员资料维护候选 | 必须阻断 role/role_key、tenant_id、organization_id、quota、group 等敏感字段越权更新 |
| 用户管理 | `POST /api/user/manage` | `AdminAuth` | 可拆分为组织内启停用户候选 | 只允许组织内普通用户，禁止操作 root、tenant_admin、organization_admin，需审计日志 |
| 用户管理 | `DELETE /api/user/:id` | `AdminAuth` | 高风险，暂不建议早期开放 | 需要组织级软删除策略、资源归属处理、token/API key 清理策略 |
| 用户安全 | `DELETE /api/user/:id/reset_passkey` | `AdminAuth` | 可作为组织内安全协助候选 | 需要二次确认、审计日志、仅限组织内普通用户 |
| 用户安全 | `DELETE /api/user/:id/2fa` | `AdminAuth` | 可作为组织内安全协助候选 | 需要二次确认、审计日志、仅限组织内普通用户 |
| OAuth | `GET /api/user/:id/oauth/bindings` | `AdminAuth` | 可作为组织内只读候选 | 需确认响应不泄漏 provider secret 或跨租户绑定 |
| OAuth | `DELETE /api/user/:id/oauth/bindings/:provider_id` | `AdminAuth` | 可作为组织内账号救援候选 | 仅限组织内普通用户，需审计日志 |
| OAuth | `DELETE /api/user/:id/bindings/:binding_type` | `AdminAuth` | 可作为组织内账号救援候选 | 需限制 binding 类型，避免清理平台级绑定或登录唯一凭证 |
| 计费/订阅 | `GET /api/subscription/admin/users/:id/subscriptions` | `RoleAuth(tenant_admin, finance, auditor)` | 未来可开放组织内订阅只读 | 需要 organization scope；当前未包含 organization_admin |
| 计费/订阅 | `POST /api/subscription/admin/users/:id/subscriptions` | `AdminAuth` | 可作为组织内订阅分配候选 | 依赖 Billing Foundation，需预算、计划归属、配额结算规则 |
| 计费/订阅 | `POST /api/subscription/admin/user_subscriptions/:id/invalidate` | `AdminAuth` | 可作为组织内订阅回收候选 | 依赖 Billing Foundation，需组织内归属和余额/退款规则 |
| 计费/订阅 | `DELETE /api/subscription/admin/user_subscriptions/:id` | `AdminAuth` | 高风险，建议晚于 invalidate | 依赖 Billing Foundation，需删除语义、账单可追溯性 |
| 统计报表 | `GET /api/log/stat` | `RoleAuth(tenant_admin, finance, auditor)` | 未来可开放组织统计 | 需要统计查询按 organization scope 聚合并测试覆盖 |
| 数据报表 | `GET /api/data/users` | `RootAuth` | 未来可新增组织级等价只读能力 | 需要重写为 organization scoped，当前 root 全局接口不宜直接开放 |
| 数据报表 | `GET /api/data/` | `RootAuth` | 未来可新增组织级等价只读能力 | 需要明确 quota date 是平台全局还是组织聚合 |
| 部门管理 | 暂无独立路由 | 无 | 未来适合 organization_admin 管理本组织部门 | 需要 Department Scope 完成后再新增 CRUD/成员归属调整 |

### B. 未来适合 tenant_admin 的路由

这些路由更适合租户管理员，而不是组织管理员。多数 controller 已经有 `TenantScopeFromContext` 或 `require...TenantAccess`，但仍需逐项确认写路径是否完整阻断跨租户字段。

| 范围 | 路由 | 当前权限 | 当前判断 | 迁移前置条件 |
| --- | --- | --- | --- | --- |
| 渠道读 | `GET /api/channel/models` | `AdminAuth` | 适合 tenant_admin/ops 读取 | 与 `models_enabled` 语义对齐，避免泄漏全局未授权模型 |
| 渠道测试 | `GET /api/channel/test`、`GET /api/channel/test/:id` | `AdminAuth` | 适合 tenant_admin/ops | 测试行为会消耗上游额度，需速率限制和租户审计 |
| 渠道余额 | `GET /api/channel/update_balance`、`GET /api/channel/update_balance/:id` | `AdminAuth` | 适合 tenant_admin/finance/ops | 确认只更新本租户渠道，避免多密钥渠道泄漏 |
| 渠道写 | `POST /api/channel/`、`PUT /api/channel/`、`DELETE /api/channel/:id` | `AdminAuth` | 适合 tenant_admin，不适合 organization_admin | 必须强制 tenant scope，禁止变更到其他租户，密钥字段需安全处理 |
| 渠道批量 | `DELETE /api/channel/disabled`、`POST /api/channel/batch` | `AdminAuth` | 适合 tenant_admin | 批量操作需全量校验每个 channel 的租户归属 |
| 渠道标签 | `POST /api/channel/tag/disabled`、`POST /api/channel/tag/enabled`、`PUT /api/channel/tag`、`POST /api/channel/batch/tag` | `AdminAuth` | 适合 tenant_admin/ops | 当前 model 有 tenant scope 参数，需补齐路由级角色和测试 |
| 渠道复制 | `POST /api/channel/copy/:id` | `AdminAuth` | 适合 tenant_admin | 复制后必须落在当前租户，密钥复制策略需确认 |
| 多 key | `POST /api/channel/multi_key/manage` | `AdminAuth` | 适合 tenant_admin 但高风险 | 需要密钥脱敏、审计、SecureVerification |
| Codex OAuth | `POST /api/channel/codex/oauth/start`、`POST /api/channel/codex/oauth/complete`、`POST /api/channel/:id/codex/oauth/start`、`POST /api/channel/:id/codex/oauth/complete`、`POST /api/channel/:id/codex/refresh`、`GET /api/channel/:id/codex/usage` | `AdminAuth` | 适合 tenant_admin/ops | OAuth credential 是渠道密钥级资产，需租户校验、二次验证和审计 |
| Ollama | `POST /api/channel/ollama/pull`、`POST /api/channel/ollama/pull/stream`、`DELETE /api/channel/ollama/delete`、`GET /api/channel/ollama/version/:id` | `AdminAuth` | 适合 tenant_admin/ops | 涉及外部 runtime 状态，需租户/渠道校验和资源消耗限制 |
| Upstream 更新 | `POST /api/channel/upstream_updates/apply`、`POST /api/channel/upstream_updates/apply_all`、`POST /api/channel/upstream_updates/detect`、`POST /api/channel/upstream_updates/detect_all` | `AdminAuth` | 适合 tenant_admin/ops | controller 已有租户作用域迹象，仍需批量路径测试覆盖 |
| 模型读 | `GET /api/models/`、`GET /api/models/search`、`GET /api/models/:id` | `AdminAuth` | 可迁移到 catalog read 角色 | 建议与 `GET /api/models/missing` 一致，给 tenant_admin/ops/auditor 只读 |
| 模型同步预览 | `GET /api/models/sync_upstream/preview` | `AdminAuth` | 适合 tenant_admin/ops 只读预览 | 需确认预览不会写库、不会泄漏其他租户渠道信息 |
| 兑换码写 | `POST /api/redemption/`、`PUT /api/redemption/`、`DELETE /api/redemption/invalid`、`DELETE /api/redemption/:id` | `AdminAuth` | 适合 tenant_admin/finance | 需要 Billing Foundation 明确额度归属和核销审计 |
| 用户订阅绑定 | `POST /api/subscription/admin/bind` | `AdminAuth` | 适合 tenant_admin/finance | 依赖 Billing Foundation，需目标用户租户校验 |
| 用户订阅写 | `POST /api/subscription/admin/users/:id/subscriptions`、`POST /api/subscription/admin/user_subscriptions/:id/invalidate`、`DELETE /api/subscription/admin/user_subscriptions/:id` | `AdminAuth` | 适合 tenant_admin/finance | 需要账务可追溯，不建议先开放删除 |
| 日志搜索 | `GET /api/log/search` | `AdminAuth` | 适合 tenant_admin/auditor/finance | 需确认 SearchAllLogs 使用 tenant/organization scope，不直接全局搜索 |
| 分组读 | `GET /api/group/` | `AdminAuth` | 适合 tenant_admin/ops | 需确认 group 是全局配置还是租户内配置 |
| 预填分组读 | `GET /api/prefill_group/` | `AdminAuth` | 适合 tenant_admin/ops 只读 | 写操作仍应 root，除非后续改为租户级配置 |

### C. 必须保持 root/admin 的路由

这些路由涉及平台级全局状态、系统安全边界、跨租户资源、供应商/模型目录写入、密钥读取或外部基础设施，应继续保持 root，或至少在当前阶段保持 legacy admin/root。

| 范围 | 路由 | 当前权限 | 保持原因 |
| --- | --- | --- | --- |
| 平台状态 | `GET /api/status/test` | `RootAuth` | 平台诊断，非租户资源 |
| 系统配置 | `GET /api/option/`、`PUT /api/option/` | `RootAuth` | 全局配置，影响所有租户 |
| 系统缓存 | `GET /api/option/channel_affinity_cache`、`DELETE /api/option/channel_affinity_cache`、`POST /api/option/rest_model_ratio`、`POST /api/option/migrate_console_setting` | `RootAuth` | 全局缓存/迁移/比例重置 |
| Custom OAuth Provider | `POST /api/custom-oauth-provider/discovery`、`GET /api/custom-oauth-provider/`、`GET /api/custom-oauth-provider/:id`、`POST /api/custom-oauth-provider/`、`PUT /api/custom-oauth-provider/:id`、`DELETE /api/custom-oauth-provider/:id` | `RootAuth` | 登录入口/客户端密钥/回调配置是平台身份边界 |
| 性能维护 | `GET /api/performance/stats`、`DELETE /api/performance/disk_cache`、`POST /api/performance/reset_stats`、`POST /api/performance/gc`、`GET /api/performance/logs`、`DELETE /api/performance/logs` | `RootAuth` | 主机级运行时维护 |
| Ratio sync | `GET /api/ratio_sync/channels`、`POST /api/ratio_sync/fetch` | `RootAuth` | 上游价格/倍率同步是全局计费基础 |
| 渠道密钥 | `POST /api/channel/:id/key` | `RootAuth` + secure verification | 明文/可用密钥读取，必须保持最小权限 |
| 渠道修复 | `POST /api/channel/fix` | `RootAuth` | 全局能力修复，可能跨租户修改 abilities |
| 全局模型拉取 | `POST /api/channel/fetch_models` | `RootAuth` | 当前语义为全局 fetch，不是单渠道租户内预览 |
| 供应商写 | `POST /api/vendors/`、`PUT /api/vendors/`、`DELETE /api/vendors/:id` | `RootAuth` | 供应商目录为平台全局目录 |
| 模型写 | `POST /api/models/`、`PUT /api/models/`、`DELETE /api/models/:id` | `RootAuth` | 模型目录和官方同步状态为平台全局目录 |
| 模型同步 | `POST /api/models/sync_upstream` | `RootAuth` | 写入模型目录，可能影响全部租户 |
| 订阅计划写 | `POST /api/subscription/admin/plans`、`PUT /api/subscription/admin/plans/:id`、`PATCH /api/subscription/admin/plans/:id` | `RootAuth` | 计划目录和状态是全局计费产品 |
| 2FA 统计 | `GET /api/user/2fa/stats` | `RootAuth` | 平台安全统计，跨租户聚合 |
| 日志清理 | `DELETE /api/log/` | `RootAuth` | 删除历史日志影响审计链 |
| Channel affinity usage cache | `GET /api/log/channel_affinity_usage_cache` | `RootAuth` | 全局运行时缓存观测 |
| Quota data | `GET /api/data/`、`GET /api/data/users` | `RootAuth` | 当前为全局数据报表接口 |
| 预填分组写 | `POST /api/prefill_group/`、`PUT /api/prefill_group/`、`DELETE /api/prefill_group/:id` | `RootAuth` | 当前更接近全局配置 |
| 部署管理 | `/api/deployments/**` | `RootAuth` | 外部计算资源、成本和基础设施生命周期管理 |
| Admin 完成充值 | `POST /api/user/topup/complete` | `AdminAuth` | 涉及资金/额度最终确认，Billing Foundation 前不应下放 |

### D. 已经完成 organization_admin 只读迁移的路由

这些路由已经通过 `RoleAuth` 或 controller/model 的作用域能力支持 `organization_admin` 只读访问。

| 范围 | 路由 | 当前权限 | 已完成能力 |
| --- | --- | --- | --- |
| 用户管理 | `GET /api/user/` | `RoleAuth(tenant_admin, organization_admin)` | organization_admin 只能读取本组织用户；无组织归属 fail closed |
| 用户管理 | `GET /api/user/search` | `RoleAuth(tenant_admin, organization_admin)` | 搜索结果按组织作用域过滤 |
| 用户管理 | `GET /api/user/:id` | `RoleAuth(tenant_admin, organization_admin)` | 详情读取按 `tenant_id`/`organization_id` 校验 |
| 充值记录 | `GET /api/user/topup` | `RoleAuth(tenant_admin, organization_admin, finance, auditor)` | organization_admin 可看本组织 topup |
| 兑换码记录 | `GET /api/redemption/` | `RoleAuth(tenant_admin, organization_admin, finance, auditor)` | organization_admin 可看本组织 redemption |
| 消费日志 | `GET /api/log/` | `RoleAuth(tenant_admin, organization_admin, finance, auditor)` | organization_admin 可看本组织日志 |
| Midjourney | `GET /api/mj/` | `RoleAuth(tenant_admin, organization_admin, ops, auditor)` | organization_admin 可看本组织 MJ 任务 |
| Task | `GET /api/task/` | `RoleAuth(tenant_admin, organization_admin, ops, auditor)` | organization_admin 可看本组织同步任务 |

## 3. 风险分析

主要风险如下：

- legacy `AdminAuth` 仍以 numeric role 为主，无法表达 `tenant_admin` 与 `organization_admin` 的细粒度差异；直接把 `AdminAuth` 替换为 `RoleAuth` 会改变安全边界。
- `organization_admin` 当前只读路径已经使用 `AccessScope`，但大量写路径仍只做 tenant 级校验，不能直接开放给 organization_admin。
- 渠道、模型、供应商和 ratio 相关路由混合了租户资源、平台目录和外部 upstream 操作；需要先区分“租户拥有的 channel”和“平台全局 catalog”。
- 账务相关写操作涉及 quota、subscription、redemption、topup，缺少统一 Billing Foundation 时容易造成额度归属不一致、退款/撤销不可追溯。
- OAuth、Passkey、2FA 都是账号恢复或登录能力，一旦组织管理员可操作，需要额外的二次验证、审计日志和目标用户角色限制。
- 数据导出/统计报表如果沿用全局接口，可能泄漏跨组织或跨租户数据；应优先新增 scoped query，而不是复用 root endpoint。
- 部门模型已存在，但当前 `AccessScopeFromContext` 对 organization_admin 不自动带入 department scope；部门级能力开放前不能宣称 department isolation 完成。
- 外部 upstream 的 test、sync、balance、pull/delete 操作可能产生实际上游成本或修改远端状态，即使是“读起来像 GET”的接口也应按写/执行能力评估。

## 4. 推荐迁移顺序

1. 完成只读补齐：优先把 `GET /api/models/`、`GET /api/models/search`、`GET /api/models/:id`、`GET /api/channel/models`、`GET /api/subscription/admin/users/:id/subscriptions`、`GET /api/log/stat` 迁移到合适的 `RoleAuth`，并补齐 tenant/organization scoped 测试。
2. 拆分 tenant_admin 运维能力：迁移渠道测试、余额刷新、upstream detect/preview 等执行类操作给 `tenant_admin`/`ops`，但保留密钥读取和全局 fetch/sync 为 root。
3. 迁移 tenant_admin 渠道写：在确认所有 channel 写路径都强制 tenant scope 后，再开放增删改、批量、标签、多 key、Codex OAuth、Ollama 操作。
4. 等 Billing Foundation 后迁移 finance 写：订阅绑定、用户订阅创建/失效、兑换码创建/更新/删除、充值订单完成需要统一账务审计后再拆权限。
5. 等 Department Scope 后迁移 organization_admin 写：组织内用户创建、用户资料维护、安全协助、OAuth 绑定解除可作为第一批组织管理写操作。
6. 最后处理数据导出与组织/部门管理 CRUD：新增 scoped API，而不是复用当前 root 全局报表或未来平台级组织管理接口。

## 5. Iteration 7.6 建议

建议 Iteration 7.6 只做低风险只读补齐，不开放写操作：

- 将模型元数据只读路由从 `AdminAuth` 收敛到 catalog read 角色：`tenant_admin`、`ops`、`auditor`。
- 将 `GET /api/channel/models` 从 `AdminAuth` 收敛到 channel/catalog read 角色，确认与 `GET /api/channel/models_enabled` 的数据边界。
- 将 `GET /api/subscription/admin/users/:id/subscriptions` 增加 organization scoped 只读能力，允许 organization_admin 查看本组织用户订阅。
- 将 `GET /api/log/stat` 增加 organization scoped 聚合测试后，再考虑给 organization_admin。
- 保留所有写操作不变，继续使用当前 `AdminAuth`/`RootAuth`，避免在缺少 Billing Foundation 和 Department Scope 时扩大权限。

## 6. 暂时不应开放的写操作

以下写操作不应在当前阶段开放给 `organization_admin`：

- 用户删除：`DELETE /api/user/:id`
- 用户 role/role_key、tenant_id、organization_id、quota、group、subscription 等字段写入
- 充值完成：`POST /api/user/topup/complete`
- 兑换码写操作：`POST /api/redemption/`、`PUT /api/redemption/`、`DELETE /api/redemption/invalid`、`DELETE /api/redemption/:id`
- 订阅绑定、创建、失效、删除：`POST /api/subscription/admin/bind`、`POST /api/subscription/admin/users/:id/subscriptions`、`POST /api/subscription/admin/user_subscriptions/:id/invalidate`、`DELETE /api/subscription/admin/user_subscriptions/:id`
- 渠道增删改、批量删除、标签批量修改、多 key 管理、Codex OAuth credential 写入
- 渠道密钥读取：`POST /api/channel/:id/key`
- 模型目录写、供应商目录写、订阅计划写
- 系统配置、性能维护、日志清理、quota data 全局导出、deployments 全部写操作

对 `tenant_admin` 也应暂缓的写操作：

- 明文密钥读取
- 全局模型同步和全局 ratio sync
- topup complete 和删除账务记录
- 删除历史日志和清理平台级缓存
- 部署管理生命周期操作，除非未来拆为租户资源池并建立成本归属

## 7. 需要等待 Billing Foundation 的能力

以下能力依赖统一账务基础后再迁移：

- 组织/租户维度订阅绑定、订阅创建、订阅失效、订阅删除
- 兑换码创建、修改、删除、失效清理
- topup complete、退款/撤销、人工加减额度
- 组织/部门维度配额分配、消费归集、余额展示
- 数据导出中的账务统计、quota date 聚合、log stat 财务报表
- 模型倍率、ratio sync、价格同步对历史账单和预扣费的影响评估

Billing Foundation 至少需要明确：

- quota 属于平台、租户、组织还是用户的权威来源
- 订阅计划是全局产品还是租户可配置产品
- redemption/topup/subscription 的审计日志和幂等语义
- 预扣费、结算、退款、手动调整的版本化记录
- finance、tenant_admin、organization_admin 的可见范围和可执行动作边界

## 8. 需要等待 Department Scope 的能力

以下能力应等 Department Scope 后再开放：

- 组织管理员创建/编辑用户时指定部门
- 部门管理员或组织管理员按部门查看用户、日志、topup、redemption、MJ/task
- 部门维度 quota、预算、订阅、模型可用范围
- 部门维度数据导出和统计报表
- 部门管理 CRUD、成员移动、部门归属变更
- 组织内渠道或分发渠道按部门授权

Department Scope 至少需要补齐：

- `AccessScopeFromContext` 对部门级角色的来源定义
- controller 写路径对 `department_id` 的字段级校验
- model query 对 `department_id` 的统一过滤
- 部门归属变更对历史日志、账单和 token 的处理策略
- 无部门用户是否可被组织管理员访问的策略

## 9. 外部 upstream / sync / test / balance 结论

外部 upstream 相关能力应按“会不会消耗额度、写入本地状态、写入远端状态、暴露 credential”四类判断：

- 可优先迁移给 tenant_admin/ops 的候选：单租户渠道 test、单租户渠道 balance update、单租户 upstream detect、单租户 upstream apply。
- 继续 root 的能力：全局 ratio sync、全局 model sync、全局 channel fix、全局 fetch models、deployment 管理。
- 必须附加安全措施的能力：Codex OAuth、multi-key 管理、Ollama pull/delete、渠道密钥读取。
- 不建议给 organization_admin：所有渠道 upstream 执行类操作。organization_admin 当前更适合用户与组织内运营数据管理，不应拥有 provider credential 或 channel runtime 控制权。

## 10. 验收记录

本次审计要求不修改业务代码、不修改测试、不调整 router、不新增功能。预期验收命令：

```bash
go test ./common ./model ./controller ./service ./router ./middleware
```

如默认 Go build cache 只读失败，使用：

```bash
GOCACHE=/tmp/quinta-go-build-cache go test ./common ./model ./controller ./service ./router ./middleware
```
