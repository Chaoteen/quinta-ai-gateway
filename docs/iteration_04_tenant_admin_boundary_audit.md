# Iteration 4：Tenant Admin Boundary 审计与设计

## 范围与结论

本轮仅进行后台管理面权限边界审计与下一阶段设计，不修改 Go 业务代码、测试、UI、migration 或配置。审计基于当前分支 `feat/iteration-04-tenant-admin-boundary` 的代码状态，并以 Iteration 2 已完成 ownership 写入、Iteration 3 已完成 relay tenant isolation 为前提。

核心结论如下：

- 当前角色只有 `root / admin / user / guest` 四个数值级别，`admin` 还没有被拆分为 tenant admin、organization admin、finance、ops、auditor；依据为 `common/constants.go` 的角色常量与 `middleware/auth.go` 的最小角色校验。
- 一部分租户资源后台操作已经具备 tenant scope 或对象归属校验，例如用户、渠道常规操作、充值补单、兑换码、日志列表/统计、任务和 Midjourney 列表；依据见下文“已覆盖”。
- 一部分被 `AdminAuth()` 放行的接口实际上操作平台全局资源或全量数据，例如日志历史清理、2FA 全局统计、`quota_data` 看板、订阅计划、模型/供应商元数据、渠道 Ability 全量修复和 deployment；普通 `admin` 当前可触达这些入口，是下一轮最高优先级的后台边界问题。
- `root` 的平台级 bypass 已存在：`model.TenantScopeFromContext()` 以 `role == RoleRootUser` 设置 `IsRoot`，`TenantScope.Apply()` 与 `AllowsTenant()` 对 root 放行。这一机制可保留，但不能继续用一个宽泛的 `AdminAuth()` 代表细粒度管理权限。

## 已完成

### 1. 当前角色体系与鉴权行为已识别

| 项目 | 当前实现 | 判断与原因 | 文件路径 / 函数 |
| --- | --- | --- | --- |
| 角色常量 | `RoleGuestUser=0`、`RoleCommonUser=1`、`RoleAdminUser=10`、`RoleRootUser=100` | 当前没有 `tenant_admin`、`organization_admin`、`finance`、`ops`、`auditor` 的表达能力；角色值只能表达粗粒度层级。 | `common/constants.go`：`RoleGuestUser`、`RoleCommonUser`、`RoleAdminUser`、`RoleRootUser`、`IsValidateRole` |
| `UserAuth` | 调用 `authHelper(c, RoleCommonUser)` | 面向登录用户的 self 接口；认证成功后读取 user cache 并写入 ownership context。 | `middleware/auth.go`：`authHelper`、`UserAuth` |
| `AdminAuth` | 调用 `authHelper(c, RoleAdminUser)` | `admin` 与 `root` 均可进入；本身不区分平台配置、租户运营、财务或审计能力，必须由 controller 的 scope 或更严格中间件补齐。 | `middleware/auth.go`：`AdminAuth` |
| `RootAuth` | 调用 `authHelper(c, RoleRootUser)` | 已可表达平台管理员专属入口，当前用于 option、custom OAuth provider、performance、ratio sync 与个别渠道敏感操作。 | `middleware/auth.go`：`RootAuth`；`router/api-router.go`：`optionRoute`、`customOAuthRoute`、`performanceRoute`、`ratioSyncRoute` |
| `TokenAuth` | 验证 API token 并由 token 所属 user cache 写入 Context ownership | relay/API 客户端使用；其 tenant 来源是 token 所属用户，而非客户端输入。该行为与后台角色授权不是同一维度。 | `middleware/auth.go`：`TokenAuth`、`SetupContextForToken` |
| `TokenOrUserAuth` | session 登录优先，否则回落至 `TokenAuth` | 两种入口均写入 user ownership context，适合 relay/用户侧资源访问，不应直接当作后台管理授权。 | `middleware/auth.go`：`TokenOrUserAuth` |
| `TokenAuthReadOnly` | 允许已禁用或额度耗尽 token 查询只读使用信息，但仍校验所属用户状态并写入 Context | 用于 token 自身使用量/日志查询，属于 self/token-only，而非 admin 能力。 | `middleware/auth.go`：`TokenAuthReadOnly`；`router/api-router.go`：`/api/usage/token`、`/api/log/token` |
| `TryUserAuth` | 仅从 session 写入 `id`，不写入完整 role / tenant context | 其挂载接口不具备 tenant-admin 授权语义；如果返回的是全局统计，应另行收紧。 | `middleware/auth.go`：`TryUserAuth`；`router/api-router.go`：`/api/pricing`、`/api/perf-metrics/*` |

### 2. Root 平台能力的现有基础已确认

`model/tenant_scope.go` 的 `TenantScopeFromContext()` 通过 `c.GetInt("role") == common.RoleRootUser` 设置 `IsRoot`；`TenantScope.Apply()` 对 root 不附加 `tenant_id` 条件，`AllowsTenant()` 对 root 返回 `true`。因此 root 当前可跨 tenant 查看和操作已 tenant-scoped 的资源，这是符合平台管理员定位的 bypass 点。

现有明确使用 `RootAuth()` 的平台入口包括：

| 路由 | 处理函数 | 保持 root-only 的理由 | 文件路径 |
| --- | --- | --- | --- |
| `/api/option/*` | `GetOptions`、`UpdateOption`、`ResetModelRatio` 等 | Option 与模型倍率影响全平台运行和定价，不是单租户资源。 | `router/api-router.go`；`controller/option.go`；`controller/pricing.go` |
| `/api/custom-oauth-provider/*` | provider 管理函数 | 登录身份源配置影响全平台认证面；操作审计覆盖情况需要人工复核。 | `router/api-router.go`；`controller/custom_oauth.go` |
| `/api/performance/*` | performance 与缓存/日志清理函数 | 进程和磁盘状态为平台运维能力。 | `router/api-router.go` |
| `/api/ratio_sync/*` | `GetSyncableChannels`、`FetchUpstreamRatios` | 同步倍率会改变全局价格或渠道策略。 | `router/api-router.go`；`controller/ratio_sync.go` |
| `/api/channel/:id/key` | `GetChannelKey` | 暴露渠道密钥，即使渠道具备 tenant ownership，也应继续限制为平台高敏操作，除非以后专门设计租户密钥托管授权。 | `router/api-router.go`；`controller/channel.go` |
| `/api/channel/fetch_models` | `FetchModels` | 当前路由已标为 root-only；批量拉取行为的实际影响范围需要人工复核。 | `router/api-router.go`；`controller/channel.go` |

## 已覆盖

本节中的“已覆盖”表示当前 controller/model 已有 tenant scope 或 self 约束，能够作为下一轮保留基础；不等同于未来角色授权模型已经完成。

### 1. Tenant-admin scoped 候选资源

| 资源 / 路由 | 当前保护链路 | 为什么可作为 tenant-admin scoped 基础 | 文件路径 / 函数 |
| --- | --- | --- | --- |
| 用户列表与搜索：`GET /api/user/`、`GET /api/user/search` | `AdminAuth` 后调用 `model.GetAllUsers(..., TenantScopeFromContext(c))`、`model.SearchUsers(..., scope)` | 非 root 的列表读取按 `tenant_id` 过滤；root 由 `TenantScope` bypass。 | `router/api-router.go`；`controller/user.go`：`GetAllUsers`、`SearchUsers`；`model/user.go` |
| 用户详情/更新/删除/绑定清理 | controller 加载目标用户后调用 `requireUserTenantAccess`，并用 role 比较防止普通 admin 操作同级或更高级用户 | 对象级操作不仅依赖列表过滤，可阻止按 ID 猜测访问其他 tenant 用户。 | `controller/user.go`：`GetUser`、`UpdateUser`、`DeleteUser`、`ManageUser`、`AdminClearUserBinding`；`controller/tenant_access.go` |
| 渠道列表、详情及常规变更 | 列表/标签批处理传入 `TenantScopeFromContext`；详情、更新、删除、测试、余额更新、copy、multi-key、Codex/Ollama 操作调用 `requireChannelTenantAccess` 或批量校验 | 渠道是租户运营资源，当前已有资源归属边界；后续可授权给 `tenant_admin` 或 `ops`。 | `router/api-router.go`：`/api/channel/*`；`controller/channel.go`；`controller/channel-billing.go`；`controller/channel_upstream_update.go`；`controller/tenant_access.go` |
| 渠道 upstream update 的单项与批量动作 | 单渠道 `ApplyChannelUpstreamModelUpdates` / `DetectChannelUpstreamModelUpdates` 校验目标 channel；批量动作逐项用 `scope.AllowsTenant(channel.TenantId)` 跳过其他 tenant | 当前普通 admin 不应修改其他 tenant 的 channel 模型配置；后台定时任务是否平台级执行属于 system-only，需单独管理。 | `controller/channel_upstream_update.go` |
| Topup 管理列表与补单：`GET /api/user/topup`、`POST /api/user/topup/complete` | 列表传入 `TenantScope`；补单先按 trade no 取订单，再调用 `requireTopUpTenantAccess` | 财务动作涉及额度增加，已有对象归属检查；后续应限制给 `finance` 或 tenant admin 中的财务授权。 | `controller/topup.go`：`GetAllTopUps`、`AdminCompleteTopUp`；`controller/tenant_access.go`；`model/topup.go` |
| Redemption CRUD 与 invalid 清理 | list/search/invalid delete 使用 `TenantScope`；detail/update/delete 调用 `requireRedemptionTenantAccess`；新增调用 `ApplyOwnershipFromContext` | 兑换码属于租户发行资产，目前创建与操作均绑定租户归属。 | `controller/redemption.go`；`controller/tenant_access.go`；`model/redemption.go` |
| Log 列表与统计：`GET /api/log/`、`GET /api/log/stat` | 分别调用 `model.GetAllLogs(TenantScopeFromContext(c), ...)` 和 `model.SumUsedQuota(..., TenantScopeFromContext(c))` | 已可作为 tenant auditor 的只读范围，但不覆盖日志删除和其他统计数据源。 | `controller/log.go`：`GetAllLogs`、`GetLogsStat`；`model/log.go` |
| Task 管理列表：`GET /api/task/` | `GetAllTask` 向 `TaskGetAllTasks` 与 `TaskCountAllTasks` 传入 `TenantScope` | 当前管理面只有只读列表路由，读取已 tenant-scoped；如果以后加入重试/取消/删除，需要重新执行对象级审计。 | `router/api-router.go`；`controller/task.go`；`model/task.go` |
| Midjourney 管理列表：`GET /api/mj/` | `GetAllMidjourney` 向 model list/count 传入 `TenantScope` | 当前管理面只有只读列表路由，已按 tenant 隔离；后台状态回写链路属于 system-only 路径，仍需人工复核。 | `router/api-router.go`；`controller/midjourney.go`；`model/midjourney.go` |
| 用户订阅实例管理 | `AdminBindSubscription`、`AdminListUserSubscriptions`、`AdminCreateUserSubscription` 校验目标 user tenant；invalidate/delete 校验订阅实例 tenant | 针对某个用户的订阅实例已有 tenant 边界，可归为 tenant-admin / finance scoped；套餐定义本身不在此结论内。 | `controller/subscription.go`：`ensureAdminTargetUserInTenant`、`ensureAdminSubscriptionInTenant` 及 admin instance handlers；`model/subscription.go` |
| 管理员操作用户 OAuth binding、Passkey、单用户 2FA | 对目标用户调用 `requireUserTenantAccess`；OAuth binding 和 2FA 还校验目标角色等级 | 这些是租户内用户支持/安全恢复动作，可设计为 tenant admin 或限定的 ops 能力；需要补充审计日志要求。 | `controller/custom_oauth.go`：`GetUserOAuthBindingsByAdmin`、`UnbindCustomOAuthByAdmin`；`controller/passkey.go`：`AdminResetPasskey`；`controller/twofa.go`：`AdminDisable2FA` |

### 2. Self-only 与 public / callback / system-only 候选

| 分类 | 路由或能力 | 当前依据与建议定位 | 文件路径 / 函数 |
| --- | --- | --- | --- |
| self-only | `/api/token/*` token CRUD/key/batch | 路由使用 `UserAuth()`，`controller/token.go` 的读取、更新、删除均使用 `c.GetInt("id")` 作为 userId，属于用户本人 token 管理。 | `router/api-router.go`；`controller/token.go`：`GetAllTokens`、`GetToken`、`AddToken`、`UpdateToken`、`DeleteToken`、`DeleteTokenBatch` |
| self-only | `/api/user/self`、`/api/user/setting`、`/api/user/passkey/*`、`/api/user/2fa/*`、用户 OAuth binding | 全部挂在 `UserAuth()` 下并围绕当前 session 用户执行；不应赋予 tenant admin 读取用户密钥/验证码材料的能力。 | `router/api-router.go`；`controller/user.go`；`controller/passkey.go`；`controller/twofa.go`；`controller/custom_oauth.go` |
| self-only | `/api/subscription/self`、购买请求、`/api/user/topup/self` 与支付请求 | 是当前用户账单和支付发起面；后台财务管理应通过独立 scoped 管理接口处理。 | `router/api-router.go`；`controller/subscription.go`；`controller/topup.go` |
| token/self-only | `/api/usage/token`、`/api/log/token` | 由 `TokenAuthReadOnly()` 以 token identity 查询该 token 的使用数据。 | `router/api-router.go`；`middleware/auth.go`；`controller/log.go` |
| public / callback-only | 注册、登录、OAuth 登录回调、密码重置、支付 webhook/notify/return | 这些接口不是后台角色能力，安全边界依赖验证码、OAuth state、签名校验与幂等处理，而非 tenant admin 权限。回调签名完整性需要人工复核。 | `router/api-router.go`：`/api/user/register`、`/api/user/login`、`/api/oauth/*`、`/api/*/webhook`、`/api/subscription/*/notify`；相关 controller |
| public product view，需人工复核 | `/api/pricing`、`/api/ratio_config` | `GetPricing` 返回模型定价、vendors 与 group ratios；`GetRatioConfig` 在配置启用时公开暴露 ratio data。这可能是产品展示需求，也可能暴露租户/内部价格策略；在确定多租户定价模型前不能简单归为 tenant-admin。 | `router/api-router.go`；`controller/pricing.go`：`GetPricing`；`controller/ratio_config.go`：`GetRatioConfig` |
| system-only，需人工复核 | 任务/Midjourney 后台状态同步、渠道 upstream 自动更新 task | 管理列表已 scoped，但后台 worker 不经后台角色路由；其是否仅操作正确租户和是否保留审计记录需另行检查。 | `controller/channel_upstream_update.go`：`StartChannelUpstreamModelUpdateTask`；任务与 Midjourney relay/worker 相关文件 |

## 待补齐

### 1. 后台接口目标归属矩阵

下表给出下一阶段最小可落地目标，而不是当前已实现的角色授权。`organization-admin scoped` 只有设计结论，当前代码尚未形成 organization scope 查询机制。

| 接口类别 | 代表路由 / 函数 | 当前状态 | 目标归属 | 原因 |
| --- | --- | --- | --- | --- |
| 用户 tenant 内管理 | `/api/user/`、`GetUser`、`CreateUser`、`UpdateUser`、`DeleteUser`、`ManageUser` | `AdminAuth` + tenant access 基础存在 | tenant-admin scoped；未来可将组织内普通用户下放为 organization-admin scoped | 用户资源有 tenant ownership；organization admin 还需要 organization scope 与禁止提升角色/配额等限制。路径：`router/api-router.go`、`controller/user.go`、`controller/tenant_access.go`。 |
| 用户安全恢复 | `AdminResetPasskey`、`AdminDisable2FA`、admin OAuth unbind | `AdminAuth` + 目标用户 tenant 校验 | tenant-admin scoped 或受限 ops；organization-admin 仅限所属组织且需审计 | 操作可导致账户访问方式改变，不能开放给 auditor/finance。路径：`controller/passkey.go`、`controller/twofa.go`、`controller/custom_oauth.go`。 |
| 2FA 汇总 | `/api/user/2fa/stats`、`Admin2FAStats` | `AdminAuth`，全局无 scope | root-only，直到新增 scoped 聚合后再开放 auditor/tenant-admin | `model.GetTwoFAStats()` 直接统计全表 `User` 与 `TwoFA`。路径：`router/api-router.go`、`controller/twofa.go`、`model/twofa.go`。 |
| Token 管理 | `/api/token/*` | `UserAuth` self-only | self-only | API token 是用户密钥；后台代操作应另设经过审计的恢复流程，而不是放宽当前路由。路径：`router/api-router.go`、`controller/token.go`。 |
| Channel CRUD/测试/余额/tenant 内模型更新 | `/api/channel/*` 除全局/密钥入口 | `AdminAuth` + 多数 scoped | tenant-admin scoped；操作类可拆给 ops | 渠道归属 tenant 且控制 relay 能力；当前已有 scope 基础。路径：`controller/channel.go`、`controller/channel-billing.go`、`controller/channel_upstream_update.go`。 |
| Channel key reveal | `/api/channel/:id/key` | root-only | root-only | 密钥高敏，维持当前最小授权。路径：`router/api-router.go`、`controller/channel.go`。 |
| Channel Ability 全量修复 | `/api/channel/fix`、`FixChannelsAbilities` | `AdminAuth`，调用 `model.FixAbility()` 无 scope | root-only；或未来新增明确的 tenant-scoped repair 接口 | 当前普通 admin 可触发全平台 Ability 变更。路径：`router/api-router.go`、`controller/channel.go`、`model/ability.go`。 |
| Topup 查看/人工补单 | `/api/user/topup*` | tenant-scoped | finance scoped，并允许 tenant-admin 在业务需要时持有 finance capability | 充值数据与额度变更属于财务动作，现有 tenant 校验只解决跨租户，不解决岗位隔离。路径：`controller/topup.go`。 |
| Redemption | `/api/redemption/*` | tenant-scoped | finance scoped 或 tenant-admin scoped | 兑换码能够发放额度，风险低于支付补单但仍属于财务资源。路径：`controller/redemption.go`。 |
| Subscription 用户实例 | `/api/subscription/admin/bind`、`users/:id/subscriptions`、`user_subscriptions/:id/*` | tenant-scoped | finance scoped；只读可给 auditor | 会创建、失效或硬删除用户权益。路径：`controller/subscription.go`。 |
| Subscription plan 定义 | `/api/subscription/admin/plans*` | `AdminAuth`，无 tenant ownership/scope | root-only；若以后支持租户套餐，先补 schema 与 ownership | `model.SubscriptionPlan` 没有 `tenant_id`，controller 对全表直接 CRUD。路径：`controller/subscription.go`、`model/subscription.go`。 |
| Log 列表与 stat | `GET /api/log/`、`GET /api/log/stat` | tenant-scoped | auditor/ops/tenant-admin 只读 scoped | 可用于审计与运行诊断，已有 tenant 聚合过滤。路径：`controller/log.go`、`model/log.go`。 |
| Log 历史删除 | `DELETE /api/log/`、`DeleteHistoryLogs` | `AdminAuth`，无 scope | root-only 立即止险；后续实现 tenant-scoped 清理再考虑 ops | 当前直接调用 `model.DeleteOldLog`，不带 `TenantScope`。路径：`router/api-router.go`、`controller/log.go`、`model/log.go`。 |
| quota_data 管理看板 | `/api/data/`、`/api/data/users` | `AdminAuth`，数据结构无 tenant ownership | root-only；暂不改成 tenant 化，等待数据模型决策 | `QuotaData` 仅有 `UserID/Username/ModelName` 等字段，查询按全表聚合，不能可靠按 tenant 过滤。路径：`controller/usedata.go`、`model/usedata.go`。 |
| Task / Midjourney 管理列表 | `GET /api/task/`、`GET /api/mj/` | tenant-scoped 只读 | ops/auditor scoped 只读 | 当前未暴露 admin 状态修改或删除路由；若增加操作端点必须按对象 tenant 再校验。路径：`controller/task.go`、`controller/midjourney.go`。 |
| Models / Vendors 元数据 | `/api/models/*`、`/api/vendors/*` | `AdminAuth`，直接 CRUD 全局表并刷新 pricing | root-only | 元数据会影响全平台模型目录、展示和定价计算，且 model/vendor 未体现 tenant ownership。路径：`controller/model_meta.go`、`controller/vendor_meta.go`、`model/model_meta.go`、`model/vendor_meta.go`。 |
| Ratio / Option | `/api/option/*`、`/api/ratio_sync/*` | 已 root-only；`/api/ratio_config` 可公开读取 | 写操作保持 root-only；公开读取政策需要人工复核 | 修改配置影响平台；公开 ratio 是否产品合同的一部分需要业务确认。路径：`router/api-router.go`、`controller/option.go`、`controller/ratio_sync.go`、`controller/ratio_config.go`。 |
| Performance metrics | `/api/perf-metrics/*` | `TryUserAuth`，可近似公开读取，无 tenant 字段证据 | root-only 或明确公开指标政策；本轮不 tenant 化 | 可能暴露全平台流量/性能，且此前暂不建议 tenant 化 `perf_metrics`。路径：`router/api-router.go`、`controller/perf_metrics.go`、`model/perf_metric.go`。 |
| Deployment 管理（额外发现） | `/api/deployments/*` | `AdminAuth`，直接使用共享 `OptionMap` API key 调用 io.net，无 tenant 校验 | root-only，直到定义 deployment ownership | 任意 admin 可列出、创建、更新、延长和删除平台外部部署；该能力与租户渠道并未建立归属映射。路径：`router/api-router.go`、`controller/deployment.go`。 |
| Group / Prefill group（额外发现） | `/api/group/`、`/api/prefill_group/*` | `AdminAuth`，读取/写入全局配置或全局表，无 tenant scope 证据 | root-only 暂时收口；若将来 tenant 自定义，再补 ownership | group ratio 与预填组会影响平台配置/表单行为，当前不是租户对象。路径：`router/api-router.go`、`controller/group.go`、`controller/prefill_group.go`。 |

### 2. 最小角色模型建议

本阶段不建议引入完整 ABAC。可先保留 `root` 平台 bypass，并新增明确的后台角色或 capability 判定层，将“登录角色”和“资源 scope”分开处理：

| 角色 | 最小职责 | 不应获得的能力 | 首批适用接口 |
| --- | --- | --- | --- |
| `root` | 平台全局配置、全租户排障、全局目录/定价/外部资源管理 | 无业务上的 tenant 限制，但必须有高敏审计 | option、ratio sync、models/vendors、subscription plans、deployment、全局清理 |
| `tenant_admin` | 当前 tenant 的用户、渠道和基础运营管理 | 全局配置、全平台统计、跨 tenant 对象、渠道 key 明文 | scoped user/channel；可按政策持有部分 redemption 能力 |
| `organization_admin` | 当前 tenant 下指定 organization 的成员与使用可见性管理 | tenant 范围渠道/财务配置、其他 organization、平台设置 | **需要人工复核并先实现 organization scope** 后再开放用户读取/有限管理 |
| `finance` | 当前 tenant 的充值、兑换、订阅实例与账务只读/操作 | 渠道密钥、模型配置、用户安全恢复、平台价格配置 | topup、redemption、user subscription instance |
| `ops` | 当前 tenant 的渠道运行维护、任务/日志诊断、安全恢复（可选） | 金额调整、全局 option、全局数据清理 | channel 操作、task/mj/log 只读；安全恢复需审计 |
| `auditor` | 当前 tenant 的只读审计和统计查看 | 任何写操作、凭证展示 | log/task/mj/topup/subscription 只读 |
| `user` | 自己的账号、token、日志、支付和订阅 | 后台管理接口 | 当前 `UserAuth()` self 路由 |

落地时必须维持两个条件：

1. 角色只能决定“可以执行哪类动作”，资源查询和写入仍必须经过 `TenantScope` 或对象 ownership 校验，不能仅靠路由角色名替代数据过滤。
2. `organization_admin` 在组织维度 scope helper、对象归属和测试齐备之前不得仅通过新角色常量开放路由，否则会把 tenant 内横向越权变成既成事实。

## 高风险区域

### P0：普通 admin 可触发平台级写操作或破坏性操作

| 风险 | 证据 | 影响 | 最小处置建议 |
| --- | --- | --- | --- |
| 全量重建 Ability | `POST /api/channel/fix` 位于 `channelRoute.Use(AdminAuth())`；`controller.FixChannelsAbilities()` 直接调用 `model.FixAbility()`，未传入 tenant scope。路径：`router/api-router.go`、`controller/channel.go`、`model/ability.go`。 | 单个 tenant admin 可改变全平台 relay 能力映射，破坏其他 tenant 的渠道使用。 | Iteration 5 第一批将路由收为 `RootAuth()`；若确有租户自修需求，另实现 scoped repair。 |
| 全平台日志删除 | `DELETE /api/log/` 使用 `AdminAuth()`；`DeleteHistoryLogs()` 调用 `model.DeleteOldLog(...)` 未传 scope。路径：`router/api-router.go`、`controller/log.go`、`model/log.go`。 | tenant admin 可删除其他 tenant 的审计证据。 | 立即改 root-only；在有 tenant-scoped delete 和审计记录前不下放。 |
| Subscription plan 全局改写 | `/api/subscription/admin/plans*` 使用 `AdminAuth()`；`SubscriptionPlan` 没有 tenant ownership，CRUD 直接作用全表。路径：`router/api-router.go`、`controller/subscription.go`、`model/subscription.go`。 | tenant admin 可修改全部客户可购买的套餐、额度和价格。 | 立即改 root-only；是否支持租户自定义套餐需要人工复核。 |
| Models / Vendors 全局改写 | `/api/models/*`、`/api/vendors/*` 使用 `AdminAuth()`；controller 直接 CRUD model/vendor 表，`UpdateModelMeta`/`DeleteModelMeta` 还会 `RefreshPricing()`。路径：`router/api-router.go`、`controller/model_meta.go`、`controller/vendor_meta.go`。 | tenant admin 可修改全平台模型目录、供应商元数据及价格展示派生数据。 | 立即改 root-only。 |
| Deployment 外部资产操作 | `/api/deployments/*` 使用 `AdminAuth()`；`controller/deployment.go` 读取共享 Option API key 并直接调用外部部署 create/update/extend/delete。 | tenant admin 可消费或销毁平台级外部计算资源，当前无 ownership 可约束。 | 立即改 root-only；后续若产品化，先设计租户归属和计费责任。 |

### P1：普通 admin 可看到全局统计或安全态势

| 风险 | 证据 | 影响 | 最小处置建议 |
| --- | --- | --- | --- |
| 2FA 全局统计泄露 | `Admin2FAStats()` 调用 `model.GetTwoFAStats()`，后者直接统计全表 `User` 和 `TwoFA`；路由仅为 `AdminAuth()`。路径：`router/api-router.go`、`controller/twofa.go`、`model/twofa.go`。 | tenant admin 得知其他租户用户数量和安全启用率。 | 在 scoped 聚合实现前改 root-only。 |
| quota_data 看板跨租户聚合 | `GetAllQuotaDates` / `GetQuotaDatesByUser` 直接调用无 scope model 查询；`QuotaData` 没有 `tenant_id`。路径：`controller/usedata.go`、`model/usedata.go`。 | tenant admin 可读取全平台用量按用户/模型聚合数据。 | 遵循暂不 tenant 化的约束，先改 root-only；数据模型调整另立迭代。 |
| perf_metrics 近公开读取 | `/api/perf-metrics/*` 使用 `TryUserAuth()`；`model/perf_metric.go` 查询未体现 tenant scope。 | 匿名或普通用户可能读取平台性能数据。 | 是否公开为产品能力需要人工复核；在结论前建议 root-only 或受控公开汇总。 |
| channel affinity usage cache stats | `/api/log/channel_affinity_usage_cache` 使用 `AdminAuth()`，handler 查询服务级缓存统计。路径：`router/api-router.go`、`controller/channel_affinity_cache.go`、`service/channel_affinity.go`。 | 缓存统计是否包含 tenant 安全 key 或全平台运行信息需人工复核。 | 复核数据结构前改 root-only。 |

### P1：边界未能表达 organization / 岗位分权

- 当前所有后台非 root 管理路由主要依赖 `AdminAuth()`；`middleware/auth.go` 不存在 finance、ops、auditor、organization admin 的动作判定。因此即使用户、渠道、充值等对象已有 tenant scope，同 tenant 内也无法做到财务与运维隔离。
- `organization_id` ownership 是否已经完整覆盖用户、日志、渠道和订阅实例，以及是否存在可复用的 `OrganizationScope` helper，本轮没有证据足以确认，标记为**需要人工复核**。在复核前不要开放 organization-admin 管理写操作。
- `controller/user.go` 的 sidebar permission 只区分 `RoleAdminUser` 与 `RoleRootUser`，属于展示权限而非服务端授权，不能作为后台边界保障依据。路径：`controller/user.go`：`calculateUserPermissions`、`generateDefaultSidebarConfig`。

### P2：产品公开接口与平台信息披露政策未明确

- `/api/pricing` 通过 `TryUserAuth()` 返回 `pricing`、`vendors`、`group_ratio` 等内容；`/api/ratio_config` 可按设置公开 `ratio_setting.GetExposedData()`。它们可能是面向购买决策的公开产品数据，也可能在企业租户场景下应隔离为合同价格，标记为**需要人工复核**。路径：`router/api-router.go`、`controller/pricing.go`、`controller/ratio_config.go`。
- `/api/rankings` 同样为公开入口，但本轮约束已将 rankings 排除在 tenant 化范围外；若其展示跨客户排行，是否继续公开属于产品/合规决定，标记为**需要人工复核**。路径：`router/api-router.go`、`model/usedata_rankings.go`。

## 后续迭代建议

### Iteration 5：优先做后台边界收口

建议下一轮仍避免完整 ABAC 重构，先做可验证的路由级止险和已有 scope 的覆盖测试。

#### 第一阶段：必须立即 root-only 的入口

将以下当前由 `AdminAuth()` 可访问、但资源明确为全局或尚无 tenant ownership 的入口收紧为 `RootAuth()`：

| 路由组 / 入口 | 函数 | 原因 | 文件路径 |
| --- | --- | --- | --- |
| `POST /api/channel/fix` | `FixChannelsAbilities` | 全量 Ability repair 没有 scope。 | `router/api-router.go`；`controller/channel.go` |
| `DELETE /api/log/` | `DeleteHistoryLogs` | destructive deletion 没有 scope。 | `router/api-router.go`；`controller/log.go` |
| `GET /api/user/2fa/stats` | `Admin2FAStats` | 全局安全统计没有 scope。 | `router/api-router.go`；`controller/twofa.go`；`model/twofa.go` |
| `GET /api/data/`、`GET /api/data/users` | `GetAllQuotaDates`、`GetQuotaDatesByUser` | `quota_data` 尚未 tenant 化且直接全局聚合。 | `router/api-router.go`；`controller/usedata.go`；`model/usedata.go` |
| `/api/subscription/admin/plans*` | plan CRUD/status handlers | 套餐定义没有 ownership，影响全平台购买与额度规则。 | `router/api-router.go`；`controller/subscription.go`；`model/subscription.go` |
| `/api/models/*`、`/api/vendors/*` | model/vendor CRUD 与 sync | 全局元数据和定价派生路径。 | `router/api-router.go`；`controller/model_meta.go`；`controller/vendor_meta.go`；`controller/model_sync.go` |
| `/api/deployments/*` | deployment handlers | 共享平台外部 API key，未建立 tenant ownership。 | `router/api-router.go`；`controller/deployment.go` |
| `/api/prefill_group/*`、`GET /api/group/` | prefill/group handlers | 当前读写全局配置/表，归属未定义。 | `router/api-router.go`；`controller/group.go`；`controller/prefill_group.go` |
| `GET /api/log/channel_affinity_usage_cache` | `GetChannelAffinityUsageCacheStats` | 服务级缓存统计边界需要复核。 | `router/api-router.go`；`controller/channel_affinity_cache.go`；`service/channel_affinity.go` |

`/api/perf-metrics/*`、`/api/pricing`、`/api/ratio_config` 是否一并收紧，应在产品公开策略评审后决定；当前判断均为**需要人工复核**，不要假设其公开行为已经满足企业多租户要求。

#### 第二阶段：保留 tenant-scoped 并拆分岗位权限

在第一阶段止险后，为以下已具备 tenant boundary 的入口增加最小 action authorization，而不改变其已有 tenant scope 语义：

| 目标角色 | 可下放接口 | 继续必须保留的检查 |
| --- | --- | --- |
| `tenant_admin` | tenant 内 user/channel 基础管理、必要的 redemption 管理 | `TenantScopeFromContext` 与 `require*TenantAccess`；root bypass 不变 |
| `finance` | topup list/complete、redemption、user subscription instance 管理 | tenant scope；写动作审计；不得触达 subscription plan 定义 |
| `ops` | channel 测试/余额刷新/tenant scoped upstream update、task/mj/log 只读 | channel 对象级校验；不得查看 key；不得执行无 scope 的清理 |
| `auditor` | scoped logs/stat、tasks、midjourney、财务只读汇总 | 只读路由与 tenant scope；不得具有补单/删除/更新动作 |

#### 第三阶段：organization-admin 设计前置工作

`organization_admin` 不应在下一轮直接获得现有 `AdminAuth()` 路由。应先完成：

1. 明确哪些对象以 `organization_id` 为授权边界，至少覆盖 user、channel、log、task、subscription instance 的业务预期，路径应从相应 `model/*.go` ownership 字段与 controller 查询链路逐一核对。
2. 设计类似 `TenantScope` 的 organization scope helper，并定义 root、tenant admin、organization admin 的 bypass/叠加规则；现有基础路径为 `model/tenant_scope.go` 与 `controller/tenant_access.go`。
3. 为 detail/update/delete/security recovery 增加组织级对象校验测试，避免只有列表查询被组织过滤而按 ID 操作仍越权。

### 暂时保留现状但必须标记风险的内容

| 内容 | 暂不 tenant 化原因 | 当下边界建议 | 文件路径 |
| --- | --- | --- | --- |
| `quota_data` | 数据表缺少 tenant ownership，直接补过滤无法可靠追溯旧数据 | 管理读取先 root-only；另立数据归属迭代 | `model/usedata.go`；`controller/usedata.go` |
| `rankings` | 属于公开产品展示还是企业隔离统计尚未确定 | 保持实现但做合规/产品复核 | `router/api-router.go`；`model/usedata_rankings.go` |
| `perf_metrics` | 当前无 tenant scope 依据，且可能属于平台运维指标 | 公开策略复核前建议限制读取 | `router/api-router.go`；`controller/perf_metrics.go`；`model/perf_metric.go` |
| Redis key namespace | 不属于本轮后台角色边界修改范围；Iteration 3 已处理 relay tenant-safe 选路缓存的核心风险 | 对管理面 cache stats 入口先按 root 保护；namespace 重构另立任务 | `service/channel_affinity.go`；相关 cache 模块 |
| payment reference prefix | 支付回调的归属与幂等性不可仅靠角色授权解决 | 保持现状并人工复核回调签名、订单 ownership 与重复通知处理 | `controller/topup*.go`；`controller/subscription*.go`；相关 payment controller |

## 下一轮 Codex 开发建议

下一轮应以“路由级最小收口 + 回归测试”开始，而非一次性引入完整角色系统：

1. 先将本报告 P0/P1 中无 tenant scope 的后台全局/破坏性入口迁移到 `RootAuth()`，优先覆盖 `channel/fix`、日志删除、2FA stats、`data`、subscription plans、models/vendors 与 deployments。
2. 为仍保留在 tenant admin 边界内的 user/channel/topup/redemption/subscription instance/log/task/midjourney 接口增加授权回归测试，证明非 root admin 无法通过 ID、批处理或统计端点触达其他 tenant。
3. 在上述止险稳定后，再引入最小岗位角色检查 helper，将 finance、ops、auditor 与 tenant admin 的动作范围分开；不要改变已经验证过的 `TenantScope` 行为。
4. 将 organization-admin、公开 pricing/ratio/perf/rankings 政策和 payment callback 安全性列为独立设计审核项，未得出明确业务结论前均标记为“需要人工复核”。
