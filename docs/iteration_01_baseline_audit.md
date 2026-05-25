# Iteration 1：多租户 Baseline Audit

## 审计范围与口径

- 审计分支：`feat/iteration-01-baseline-audit`。
- 本轮仅进行静态代码审计与文档记录，不修改业务逻辑或测试。
- 本文所称“已覆盖”，表示当前可见调用路径中已使用租户字段、`TenantScope` 或目标对象租户校验；不表示整个多租户改造已经闭环。
- `TenantScope` 当前只隔离 `tenant_id`，尚未按 `organization_id`、`department_id`、`distribution_channel_id` 做进一步权限边界；依据见 `model/tenant_scope.go`。
- root 用户被设计为跨租户可见：`TenantScope.AllowsTenant` 在 `IsRoot` 时直接返回 `true`；依据见 `model/tenant_scope.go`、`controller/tenant_access.go`。

## 已完成

### 1. 租户层级基础模型已经建立

| 能力 | 判断 | 依据与原因 |
| --- | --- | --- |
| Tenant 主体 | 已完成 | `model/tenant.go` 定义 `Tenant`，包含状态、备注、创建/更新时间及软删除字段。 |
| Organization 归属 | 已完成 | `model/organization.go` 定义 `TenantId` 与 `DistributionChannelId`，组织可归属到租户和分销渠道。 |
| Department 归属 | 已完成 | `model/department.go` 定义 `TenantId`、`OrganizationId`、`ParentId`、`DistributionChannelId`，能够表达租户内组织树。 |
| DistributionChannel 归属 | 已完成 | `model/distribution_channel.go` 定义 `TenantId`、`ParentId`、`OwnerUserId`，能够表达租户内渠道归属。 |
| 数据迁移注册 | 已完成 | `model/main.go` 已将 `Tenant`、`Organization`、`Department`、`DistributionChannel` 注册到迁移列表。 |

### 2. 通用 ownership 与 tenant scope 基础设施已经建立

| 能力 | 判断 | 依据与原因 |
| --- | --- | --- |
| Ownership 快照 | 已完成 | `model/ownership.go` 的 `OwnershipSnapshot` 包含四级 ownership 字段，并提供从 Gin Context、用户缓存、订阅订单和用户订阅读取归属的路径。 |
| 新数据 ownership 写入辅助 | 已完成但覆盖不完整 | `model/ownership.go` 的 `ApplyTo` 已支持 `Log`、`TopUp`、`Redemption`、`SubscriptionOrder`、`UserSubscription`、`SubscriptionPreConsumeRecord`、`Midjourney`；未包含 `User`、`Token`、`Channel`、`Task`、`Ability`，其影响见“待补齐”和“高风险区域”。 |
| 默认租户兼容 | 已完成但需收敛 | `model/ownership.go` 与 `model/tenant_scope.go` 都把 `tenant_id = 0` 归一到 `1`，可兼容旧数据；但也会掩盖新建数据漏写 ownership 的问题。 |
| Scope 查询过滤 | 已完成 | `model/tenant_scope.go` 对非 root 查询增加 `tenant_id = ?`，root 不增加过滤。 |
| 目标对象访问校验 | 已完成但对象有限 | `controller/tenant_access.go` 已提供 `User`、`Channel`、`Redemption`、`TopUp` 的目标对象租户校验函数。 |

### 3. Gin Context 到 RelayInfo 的租户传播已经建立

| 环节 | 判断 | 依据与原因 |
| --- | --- | --- |
| 用户身份写入 Context | 已完成 | `model/user_cache.go` 的 `WriteContext` 将 `TenantId`、`OrganizationId`、`DepartmentId`、`DistributionChannelId` 写入 Context；`middleware/auth.go` 的用户认证和 token 认证都会调用该方法。 |
| 缓存读取失败的默认归属 | 已完成但有风险 | `middleware/auth.go` 在后台认证读取用户缓存失败时写入默认 `tenant_id = 1`；这保证请求继续具有 scope，但可能把异常用户错误归入默认租户，需要后续决定是否应直接失败。 |
| RelayInfo 复制 ownership | 已完成 | `relay/common/relay_info.go` 的 `RelayInfo` 已包含四级 ownership，`genBaseRelayInfo` 从 Context 读取并填充，`tenant_id = 0` 时回落为 `1`。 |

### 4. 指定业务对象已具备 tenant ownership 字段

| 业务对象 | 字段状态 | 文件依据 | 说明 |
| --- | --- | --- | --- |
| users | 已具备 | `model/user.go` | `User` 含四级 ownership 字段。 |
| tokens | 已具备 | `model/token.go` | `Token` 含四级 ownership 字段；创建填充未闭环，见后文。 |
| channels | 已具备 | `model/channel.go` | `Channel` 含四级 ownership 字段；创建与 relay 使用未闭环，见后文。 |
| logs | 已具备 | `model/log.go` | `Log` 含四级 ownership 字段，主要写入路径已填充。 |
| topups | 已具备 | `model/topup.go` | `TopUp` 含四级 ownership 字段。 |
| redemptions | 已具备 | `model/redemption.go` | `Redemption` 含四级 ownership 字段。 |
| tasks | 已具备 | `model/task.go` | `Task` 含四级 ownership 字段；任务创建未填充，见后文。 |
| subscriptions | 已具备 | `model/subscription.go` | `SubscriptionOrder`、`UserSubscription`、`SubscriptionPreConsumeRecord` 均含四级 ownership；`SubscriptionPlan` 当前是共享套餐定义，不含租户字段。套餐是否未来需要租户私有化，**需要人工复核**。 |
| midjourney | 已具备 | `model/midjourney.go` | `Midjourney` 含四级 ownership 字段。 |

## 已覆盖

### 1. 后台列表接口的 tenant scope 覆盖

| 列表对象 | 覆盖状态 | 文件依据 | 为什么判定为已覆盖 |
| --- | --- | --- | --- |
| users | 已覆盖 | `controller/user.go`、`model/user.go` | `GetAllUsers` 和 `SearchUsers` 传入 `TenantScopeFromContext(c)`；模型查询对 `users.tenant_id` 应用 scope。 |
| topups | 已覆盖 | `controller/topup.go`、`model/topup.go` | 管理员全量及搜索列表均传入 scope；模型查询对 `top_ups.tenant_id` 应用 scope。 |
| logs | 已覆盖（列表与统计） | `controller/log.go`、`model/log.go` | `GetAllLogs` 和 `GetLogsStat` 传入 scope；`GetAllLogs`、`SumUsedQuota` 对 `logs.tenant_id` 应用过滤。日志删除不在该覆盖结论内，见高风险区域。 |
| channels | 已覆盖（列表与搜索） | `controller/channel.go`、`model/channel.go` | 普通列表在控制器直接对 `channels` 应用 scope；标签模式、搜索模式调用 `GetChannelsByTagScoped`、`SearchChannelsScoped`、`SearchTags` 和 `CountAllTags`。 |
| redemptions | 已覆盖 | `controller/redemption.go`、`model/redemption.go` | 全量、搜索列表均传入 scope，模型对 `redemptions.tenant_id` 应用过滤。 |
| tasks | 已覆盖（列表读取） | `controller/task.go`、`model/task.go` | `GetAllTask` 将 scope 传给 `TaskGetAllTasks` 与 `TaskCountAllTasks`；两者均过滤 `tasks.tenant_id`。任务写入正确性未覆盖，见高风险区域。 |
| subscriptions | 已覆盖（管理员用户订阅列表） | `controller/subscription.go`、`model/subscription.go` | `AdminListUserSubscriptions` 先校验目标用户租户，再调用带 scope 的 `GetAllUserSubscriptions` 过滤 `user_subscriptions.tenant_id`。套餐列表是共享配置，不属于租户订阅实例列表。 |
| midjourney | 已覆盖（列表读取） | `controller/midjourney.go`、`model/midjourney.go` | `GetAllMidjourney` 将 scope 传给 `GetAllTasks` 与 `CountAllTasks`，两者均过滤 `midjourneys.tenant_id`。 |

### 2. 详情、更新、删除和操作接口已有覆盖

| 操作范围 | 覆盖状态 | 文件依据 | 为什么判定为已覆盖 |
| --- | --- | --- | --- |
| user detail/update/delete/manage | 已覆盖主要管理员路径 | `controller/user.go`、`controller/tenant_access.go` | `GetUser`、`UpdateUser`、`DeleteUser`、`ManageUser` 在修改或返回目标用户前调用 `requireUserTenantAccess`，之后才执行角色判断和写操作。 |
| user binding / passkey / disable 2FA | 已覆盖目标用户操作 | `controller/user.go`、`controller/custom_oauth.go`、`controller/passkey.go`、`controller/twofa.go` | 管理员清理绑定、自定义 OAuth 查询/解绑、重置 Passkey、禁用目标用户 2FA 均先校验目标用户租户。 |
| channel detail/update/delete/test/balance/tag/batch/multi-key/Ollama/Codex | 已覆盖大部分显式目标操作 | `controller/channel.go`、`controller/channel-test.go`、`controller/channel-billing.go`、`controller/codex_oauth.go`、`controller/codex_usage.go`、`controller/channel_upstream_update.go`、`controller/tenant_access.go` | 单渠道路径在操作前调用 `requireChannelTenantAccess`；批量 ID 路径先取出全部 channel，再调用 `requireChannelsTenantAccess`；tag 操作和 disabled 清理将 scope 下传到模型。全量能力修复除外，见高风险区域。 |
| redemption detail/update/delete/invalid cleanup | 已覆盖管理员管理路径 | `controller/redemption.go`、`model/redemption.go`、`controller/tenant_access.go` | 详情、更新、按 ID 删除先校验目标兑换码租户；批量清理失效兑换码直接按 scope 删除。用户兑换行为除外，见高风险区域。 |
| subscription admin operation | 已覆盖用户订阅实例管理路径 | `controller/subscription.go`、`model/subscription.go` | 绑定或创建用户订阅前先校验目标用户租户；失效和删除订阅前先校验订阅实例租户。 |
| topup admin completion | 已覆盖已发现的管理员补单入口 | `controller/topup.go`、`controller/tenant_access.go` | `AdminCompleteTopUp` 先按订单号读取 `TopUp` 并调用 `requireTopUpTenantAccess`，通过后才执行 `ManualCompleteTopUp`。支付 webhook 属于无登录回调链路，依据订单号与 payment provider 防混用，不按管理员 tenant scope 工作。 |
| secure verification for channel key | 已覆盖为 root 专属操作 | `router/api-router.go`、`middleware/secure_verification.go`、`controller/channel.go` | 查看渠道密钥路由同时要求 `AdminAuth`、`RootAuth` 和安全验证；root 在当前 scope 设计中本就允许跨租户，因此该入口未单独做 channel tenant 校验不构成非 root 横向访问。 |

### 3. relay 中已经具备的归属写入

| 归属对象 | 覆盖状态 | 文件依据 | 为什么判定为已覆盖 |
| --- | --- | --- | --- |
| 同步 relay 消费日志 | 已覆盖 | `model/log.go`、`service/text_quota.go` | `RecordConsumeLog` 使用 `OwnershipFromContext(c).ApplyTo(log)`；文本结算完成后调用该写入路径。 |
| relay 错误日志 | 已覆盖 | `model/log.go`、`controller/relay.go` | `RecordErrorLog` 使用 Context ownership；relay 错误处理调用该函数。 |
| 任务消费/退款日志 | 已覆盖日志归属 | `model/log.go`、`service/task_billing.go` | `RecordTaskBillingLog` 使用用户缓存推导 ownership；异步结算/退款日志经过该路径。 |
| midjourney 记录写入 | 已覆盖 | `relay/mjproxy_handler.go`、`model/midjourney.go` | 提交任务前显式 `ApplyOwnershipFromContext`，模型 `Insert` 也能在缺失时从用户回填。 |
| topup 与 subscription order 写入 | 已覆盖主要创建路径 | `controller/topup.go`、`controller/topup_stripe.go`、`controller/topup_creem.go`、`controller/topup_waffo.go`、`controller/topup_waffo_pancake.go`、`controller/subscription_payment_epay.go`、`controller/subscription_payment_stripe.go`、`controller/subscription_payment_creem.go`、`model/subscription.go` | 已发现的充值和订阅下单入口在 insert 前应用 Context ownership；订阅完成时继续把订单 ownership 复制到订阅实例与对应 topup。 |
| subscription 额度预扣记录 | 已覆盖字段归属 | `model/subscription.go`、`model/ownership.go` | `PreConsumeUserSubscription` 建立记录时从选中的 `UserSubscription` 复制 ownership。 |

## 待补齐

### 1. 新建数据 ownership 写入未形成统一闭环

| 对象/场景 | 当前状态 | 文件依据 | 为什么需要补齐 |
| --- | --- | --- | --- |
| 管理员创建 user | 待补齐 | `controller/user.go`、`model/user.go` | `CreateUser` 构建的 `cleanUser` 未写入当前管理员的 ownership，`User.Insert` 也不做回填；非默认租户管理员创建用户时会依赖数据库默认值 `tenant_id = 1`。 |
| 注册与 OAuth 新建用户的租户归属策略 | 需要人工复核 | `controller/user.go`、`controller/oauth.go`、`model/user.go` | 公共注册入口不存在已认证租户上下文，默认归入 tenant 1 可能是产品设计，也可能需要邀请码、域名或渠道决定归属；静态代码无法确认目标策略。 |
| 新建 token | 待补齐 | `controller/token.go`、`model/token.go`、`controller/user.go` | `AddToken` 与注册生成默认 token 未写入 ownership，`Token.Insert` 不回填；字段虽然存在，但非默认租户用户的新 token 记录可能落为 tenant 1。 |
| 新建 channel | 待补齐且高优先级 | `controller/channel.go`、`model/channel.go` | `AddChannel` 直接接收请求体中的 `Channel` 并批量写库，没有强制覆盖为当前租户 ownership；未传值时落为 tenant 1，传值时还可能由请求方指定归属。 |
| 新建 ability | 待补齐且高优先级 | `model/ability.go` | `Channel.AddAbilities` 与 `UpdateAbilities` 构造 `Ability` 时没有复制 channel 的四级 ownership，导致 `abilities.tenant_id` 虽有字段但通常依赖默认值。 |
| 新建异步 task | 待补齐且高优先级 | `relay/common/relay_info.go`、`controller/relay.go`、`model/task.go` | `RelayInfo` 已有 ownership，但 `InitTask` 未复制四级 ownership，`Task.Insert` 也不回填；非默认租户任务在后台列表中可能不可见或错误归属到 tenant 1。 |

### 2. relay 归属传播尚未落到渠道隔离和完整计费归属

| 环节 | 当前状态 | 文件依据 | 为什么需要补齐 |
| --- | --- | --- | --- |
| 渠道选择 | 待补齐 | `middleware/distributor.go`、`service/channel_select.go`、`model/ability.go`、`model/channel_cache.go` | 选择逻辑依据 group/model/priority/weight 查找 ability/channel，未使用 Context 中的 `tenant_id` 或 `TenantScope`；内存缓存也按 group/model 聚合全量 channel。因此请求可能选到其他 tenant 的上游渠道。 |
| 指定 channel 的 relay 请求 | 待补齐 | `middleware/auth.go`、`middleware/distributor.go` | token 中指定 channel 的路径直接 `GetChannelById` 并校验状态，没有校验 channel 与调用者 tenant；管理员 token 能否跨租户指定渠道需要按产品权限重新确认，当前代码没有隔离约束。 |
| 用户/令牌额度扣减 | 部分覆盖，需补充一致性约束 | `service/pre_consume_quota.go`、`service/funding_source.go`、`model/user.go`、`model/token.go` | 消耗以认证得到的 `UserId`、`TokenId` 更新额度，能够扣到实际调用者记录；但更新语句不携带 tenant 条件，且 token 创建归属存在缺口，因此无法仅凭 ownership 字段证明租户账务一致。 |
| channel 使用额度 | 待补齐前置隔离 | `service/text_quota.go`、`service/task_billing.go`、`model/channel.go` | 用量按最终选中的 `ChannelId` 累加；在渠道选择未按 tenant 隔离前，该统计可能累计到跨租户渠道。 |

## 高风险区域

以下项目不是单纯缺少展示字段，而是可能造成非 root 租户管理员越权查看、修改或错误归属的风险，应优先于更细层级的组织/部门隔离处理。

| 风险等级 | 风险 | 文件依据 | 风险原因 |
| --- | --- | --- | --- |
| 高 | relay 可选择其他 tenant 的 channel | `middleware/distributor.go`、`service/channel_select.go`、`model/ability.go`、`model/channel_cache.go` | Context 已有 tenant，但分发选择和缓存索引完全未按 tenant 划分；这会直接影响上游密钥使用、渠道用量与故障禁用归属。 |
| 高 | channel/ability 创建归属可错写或由请求体影响 | `controller/channel.go`、`model/channel.go`、`model/ability.go` | `AddChannel` 不固化 Context ownership，ability 重建不继承 channel ownership；这是列表可见性与 relay 隔离的基础数据污染源。 |
| 高 | task 写入缺少 ownership | `controller/relay.go`、`model/task.go`、`relay/common/relay_info.go` | 已认证的非默认租户任务会在 insert 时落入默认 tenant，随后管理员列表按正确 tenant scope 查询时无法看到本租户任务，且异步账务审计无法可靠对账。 |
| 高 | redemption redeem 未验证兑换码与领取用户同 tenant | `model/redemption.go` | `Redeem` 仅按 key 获取兑换码后向传入 `userId` 加额度，不比较 `redemption.TenantId` 和用户 tenant；若 key 泄露或被转交，可能跨 tenant 兑付。该行为是否允许跨租户营销券，**需要人工复核**。 |
| 高 | 非 root admin 可触发全平台 ability 重建 | `router/api-router.go`、`controller/channel.go`、`model/ability.go` | `/api/channel/fix` 仅要求 `AdminAuth`，`FixAbility` 清空并重建全表且没有 scope；任一租户管理员可以影响全部租户 relay 路由。 |
| 中 | 管理员删除历史日志未使用 tenant scope | `router/api-router.go`、`controller/log.go`、`model/log.go` | `/api/log` 的 `DELETE` 使用 `AdminAuth`，`DeleteOldLog` 仅按时间删除；非 root admin 可删除其他 tenant 日志。 |
| 中 | 管理员 2FA 统计是全局数据 | `router/api-router.go`、`controller/twofa.go`、`model/twofa.go` | `/api/user/2fa/stats` 对 admin 开放，但统计 `users` 和 `two_fa` 全表且不接收 scope；这会向租户管理员泄露全平台安全采用率。 |
| 中 | user/token 新建归属依赖默认 tenant 1 | `controller/user.go`、`controller/token.go`、`model/user.go`、`model/token.go` | 新增对象无法稳定归属于操作者所在 tenant，后续 scope 查询、审计和额度归属将出现不一致。 |
| 中 | Context 缓存失败时后台认证静默归入 tenant 1 | `middleware/auth.go`、`model/user_cache.go` | 当缓存/查询失败发生在非默认租户管理员请求中，继续执行并给默认 scope 可能产生误操作或误判；是否应该 fail closed，**需要人工复核**。 |

### 需要人工复核的操作面

| 项目 | 文件依据 | 需要确认的问题 |
| --- | --- | --- |
| SubscriptionPlan 是否为平台共享目录 | `controller/subscription.go`、`model/subscription.go` | 当前套餐计划不含 tenant 字段且管理接口对 admin 开放；若未来允许租户自定义套餐，需要改为 root 专属或加入 ownership。 |
| payment webhook 的跨租户模型 | `router/api-router.go`、`model/topup.go`、`model/subscription.go` | 回调按不可预测订单号和支付提供方校验执行，通常不应使用登录 tenant scope；仍需确认订单号泄露场景、运营补单权限和回调审计要求。 |
| 自动后台任务的跨租户职责 | `model/midjourney.go`、`model/task.go`、`service/task_polling.go`、`controller/task_video.go` | 未完成任务轮询属于系统级任务，可读取全租户记录；需要确认其日志、故障禁用和结算操作是否必须保留原 ownership。 |

## 后续迭代建议

### 1. 建议优先修复的租户闭环顺序

| 优先级 | 建议 | 文件落点 | 原因 |
| --- | --- | --- | --- |
| P0 | 统一新建对象 ownership 写入策略，并禁止普通 admin 请求体覆盖归属 | `controller/user.go`、`controller/token.go`、`controller/channel.go`、`model/ownership.go`、`model/task.go`、`model/ability.go` | 若基础数据继续写错，后续 scope 与审计都无法可信。 |
| P0 | 让 relay 渠道选择和 channel affinity/缓存索引包含 tenant 维度 | `middleware/distributor.go`、`service/channel_select.go`、`model/ability.go`、`model/channel_cache.go`、`service/channel_affinity.go` | 这是运行时跨租户上游资源隔离的核心边界。 |
| P0 | 为兑换码领取确定并落实租户策略 | `model/redemption.go`、`controller/redemption.go` | 若兑换码应租户私有，必须在兑付事务中校验领取用户 ownership；若允许全局券，应显式建模。 |
| P1 | 将 destructive/admin aggregate 操作区分 root-only 与 tenant-scoped | `router/api-router.go`、`controller/channel.go`、`model/ability.go`、`controller/log.go`、`model/log.go`、`controller/twofa.go`、`model/twofa.go` | 全表重建、删除和全局安全指标不应默认暴露给租户 admin。 |
| P1 | 增加租户隔离回归测试 | 对应 controller/model/service 测试目录 | 重点覆盖非 root admin 无法查看/修改其他 tenant 记录、relay 不选取其他 tenant channel、异步任务正确归属。 |

### 2. 暂时不建议 tenant 化的内容

这些对象当前承载平台聚合、基础设施或外部协议定位信息；在运行时资源隔离闭环之前直接追加 tenant 维度，会扩大迁移面且不解决当前最高风险。是否长期维持全局语义，仍应在产品权限模型确定后复核。

| 内容 | 暂不 tenant 化的建议 | 文件依据 | 原因与边界 |
| --- | --- | --- | --- |
| `quota_data` | 暂不改表结构 | `model/usedata.go`、`controller/usedata.go` | 当前按 `user_id/username/model_name/hour` 聚合，并被排行榜消费；直接增加 tenant 会影响历史聚合和看板查询。注意：后台 `/api/data` 当前是 admin 可读的全局聚合；若租户管理员不得查看全局数据，应先限制接口权限或改用 tenant-scoped `logs` 派生报表，而不是立即迁移表。 |
| `rankings` | 保持平台级公开聚合 | `controller/rankings.go`、`service/rankings.go`、`model/usedata_rankings.go`、`router/api-router.go` | 排行榜当前无认证公开读取，基于 `quota_data` 聚合模型热度；租户化会改变产品语义并拆分公共榜单。 |
| `perf_metrics` | 保持模型广场/平台级指标 | `model/perf_metric.go`、`controller/perf_metrics.go`、`router/api-router.go` | 数据按 model/group/time bucket 聚合且接口为 `TryUserAuth`，用于全局性能展示，不是账务归属来源。 |
| Redis key namespace | 暂不整体加入 tenant 前缀 | `model/user_cache.go`、`model/token_cache.go`、`common/redis.go` | 用户 ID、token key 当前被用作全局身份定位键；在 ID/key 唯一性保持成立时不存在单纯 key 冲突收益。应先修正数据库 ownership 与 channel selection，再评估按租户隔离可观测性或清理能力。 |
| payment reference prefix | 暂不加入 tenant 前缀 | `controller/topup.go`、`controller/topup_stripe.go`、`controller/topup_creem.go`、`controller/topup_waffo.go`、`controller/topup_waffo_pancake.go`、`controller/subscription_payment_epay.go`、`controller/subscription_payment_stripe.go`、`controller/subscription_payment_creem.go` | 支付回调通过全局唯一订单引用与 provider 校验定位订单；订单记录自身已经携带 ownership。改变外部 reference 格式会影响支付渠道兼容和回调追踪，不能替代服务端权限校验。 |

### 3. 本轮结论

- 数据模型、后台主要列表查询、部分目标操作校验，以及 Context 到 `RelayInfo` 的传播已经具备多租户基础；依据集中在 `model/tenant_scope.go`、`model/ownership.go`、`middleware/auth.go`、`relay/common/relay_info.go` 和各列表控制器/模型文件。
- 当前尚不能认定 relay 与管理面已经实现安全租户隔离。阻断性原因是 channel/ability/task 创建归属不闭环，以及 relay channel selection 没有 tenant 过滤；依据见 `controller/channel.go`、`model/ability.go`、`model/task.go`、`middleware/distributor.go`、`model/channel_cache.go`。
- 下一迭代应先修复 P0 数据归属与运行时渠道隔离，再考虑组织/部门级授权及全局聚合对象的权限产品化。
