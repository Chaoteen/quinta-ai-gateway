# Iteration 8.2 Billing Ownership Audit

## 1. 当前状态

本轮审计覆盖 `TopUp`、`Redemption`、`SubscriptionOrder`、`UserSubscription`、`SubscriptionPreConsumeRecord` 五类 Billing 对象，重点检查 `tenant_id`、`organization_id`、`department_id`、`distribution_channel_id` 的创建、查询、更新、删除和 webhook 传播完整性。

当前代码已经具备基础 ownership 传播能力：

- `model.OwnershipSnapshot` 统一承载四类 ownership 字段。
- `ApplyOwnershipFromContext()` 用于从当前登录上下文写入 ownership。
- `OwnershipByUserId()` / `ApplyOwnershipFromUser()` 用于从用户维度推导 ownership。
- `ApplyOwnershipScope()` 用于基于 `AccessScope` 过滤查询。
- `TenantScopeFromContext()` / `TenantScope.Apply()` 用于 tenant 级别过滤或校验。
- `ownershipFromSubscriptionOrder()` 和 `ownershipFromUserSubscription()` 用于从订单、订阅继续传播 ownership。

主要缺口集中在三类：

- `NormalizeOwnership()` 会把空 `tenant_id` 归一化为 `tenant_id=1`，导致创建链路或用户缓存异常时存在 fallback 风险。
- 部分更新、删除链路只做 tenant 校验，没有 organization 级别校验。
- `Redeem()` 兑换码核销链路没有校验兑换码 ownership 与核销用户 ownership 是否一致。

## 2. Ownership Matrix

| 对象 | 创建链路 | 查询链路 | 更新链路 | 删除链路 | Webhook / 内部链路 | 当前结论 |
| --- | --- | --- | --- | --- | --- | --- |
| TopUp | 用户充值创建时 controller 调用 `ApplyOwnershipFromContext()`；`TopUp.Insert()` 在 `tenant_id=0` 时 fallback 到 `ApplyOwnershipFromUser()` | `GetAllTopUpsByAccessScope()` / `SearchAllTopUpsByAccessScope()` 使用 `ApplyOwnershipScope()` | webhook 与手动完成通过 `trade_no` 更新状态；`POST /api/user/topup/complete` controller 先做 `requireTopUpTenantAccess()` | 未发现面向管理端的删除链路 | Epay / Stripe / Creem / Waffo / WaffoPancake 完成充值时继承已存在 TopUp ownership，不重新计算 | 查询 scoped 较完整；创建 fallback 和手动完成 tenant-only 校验需关注 |
| Redemption | 管理端创建时 controller 调用 `ApplyOwnershipFromContext()`；`Redemption.Insert()` 在 `tenant_id=0` 时 fallback 到 `ApplyOwnershipFromUser()` | 列表使用 `ApplyOwnershipScope()`；搜索使用 `TenantScope.Apply()`；详情使用 tenant 校验 | 更新前通过 `requireRedemptionTenantAccess()` 做 tenant 校验 | 删除前通过 `requireRedemptionTenantAccess()` 做 tenant 校验；批量删除失效码使用 `TenantScope.Apply()` | `Redeem()` 按 key 核销，不校验核销用户与兑换码 ownership 是否匹配 | 兑换码核销是最高风险点；更新删除目前仅 tenant 边界 |
| SubscriptionOrder | 用户订阅支付创建时 controller 调用 `ApplyOwnershipFromContext()`；`SubscriptionOrder.Insert()` 在 `tenant_id=0` 时 fallback 到 `ApplyOwnershipFromUser()` | 当前主要由支付完成链路按 `trade_no` 查找；管理查询能力有限 | `CompleteSubscriptionOrder()` / `ExpireSubscriptionOrder()` 通过 `trade_no` 更新状态并校验 provider/status | 未发现面向管理端的删除链路 | 完成订单时 `UserSubscription` 和订阅对应 `TopUp` 继承订单 ownership | 传播设计合理；若订单创建 ownership 错误，后续会继续放大错误 |
| UserSubscription | 订单完成时继承 `SubscriptionOrder` ownership；管理绑定通过 `CreateUserSubscriptionFromPlanTx()` 使用 `OwnershipByUserId()` | `GetAllUserSubscriptionsByAccessScope()` 使用 `ApplyOwnershipScope()`；controller 对目标用户使用 `AllowsOwnership()` | 失效前 controller 调用 `ensureAdminSubscriptionInTenant()`；model 层不自行校验 | 删除前 controller 调用 `ensureAdminSubscriptionInTenant()`；model 层硬删除 | 预扣记录从订阅继承 ownership | 列表读取支持 organization scope；管理 mutation 仍是 tenant-only，且模型层无防线 |
| SubscriptionPreConsumeRecord | `PreConsumeUserSubscription()` 从 `UserSubscription` 继承 ownership | 未发现管理端查询路由；内部按 request_id 查询 | `RefundSubscriptionPreConsume()` 按 request_id 更新状态，无外部 ownership 校验 | `CleanupSubscriptionPreConsumeRecords()` 全局清理旧记录 | 内部预扣、退款生命周期 | 依赖订阅 ownership 正确性；若未来开放查询需使用 `ApplyOwnershipScope()` |

## 3. 风险分析

### 3.1 创建链路风险

`TopUp`、`Redemption`、`SubscriptionOrder` 的用户入口在 controller 层通常已经调用 `ApplyOwnershipFromContext()`，正常请求路径可以写入完整 ownership。

风险来自模型层 fallback：

- `TopUp.Insert()`
- `Redemption.Insert()`
- `SubscriptionOrder.Insert()`
- `CreateUserSubscriptionFromPlanTx()` 间接使用 `OwnershipByUserId()`

这些路径在无法从上下文或用户缓存拿到 ownership 时，可能经由 `NormalizeOwnership()` 将 `tenant_id=0` 归一化为 `tenant_id=1`。这会把缺失 ownership 的账务对象写入默认 tenant，属于 fail-open 风险。

### 3.2 查询链路风险

列表类查询整体较好：

- TopUp 列表和搜索使用 `ApplyOwnershipScope()`。
- Redemption 列表使用 `ApplyOwnershipScope()`。
- UserSubscription 列表使用 `ApplyOwnershipScope()`。

仍需注意：

- Redemption 搜索链路使用 `TenantScope.Apply()`，只有 tenant 边界，没有 organization 边界。
- Redemption 详情、更新、删除通过 tenant 校验，不支持 organization scope。
- `TenantScopeFromContext()` 本身也会归一化空 tenant 到 `tenant_id=1`，如果上下文缺失 tenant，可能出现默认 tenant fallback。

### 3.3 更新链路风险

高风险更新链路包括：

- `Redeem()`：按兑换码 key 直接核销，不校验兑换码 ownership 与当前用户 ownership。
- `AdminInvalidateUserSubscription()`：controller 有 tenant 校验，model 层无 ownership 校验。
- `ManualCompleteTopUp()`：controller 有 TopUp tenant 校验，model 层只按 `trade_no` 完成。
- `CompleteSubscriptionOrder()`：webhook 按 `trade_no` 完成订单，依赖订单创建时 ownership 正确。

当前设计把权限边界主要放在 controller 层，model mutation 本身大多不防越权。如果未来新增调用入口，容易绕过既有权限检查。

### 3.4 删除链路风险

已发现的删除链路主要集中在：

- Redemption 删除：删除前有 tenant 校验，但没有 organization 校验。
- UserSubscription 删除：删除前有 tenant 校验，但 model 层执行硬删除，审计追踪能力弱。
- SubscriptionPreConsumeRecord 清理：内部全局清理旧记录，当前未暴露管理端入口。

账务类对象删除应优先软删除或状态化，不建议继续扩大硬删除能力。

### 3.5 Webhook 链路风险

Stripe、Creem、Waffo、Epay 的充值或订阅完成链路均以 provider 回调为入口，主要通过 `trade_no` 定位本地 TopUp 或 SubscriptionOrder。

当前行为：

- TopUp 完成时继承已存在 TopUp ownership，不重新计算 ownership。
- SubscriptionOrder 完成时，`UserSubscription` 继承订单 ownership。
- 订阅支付生成的 TopUp 继承订单 ownership。
- provider/status 校验可以降低跨支付通道误完成风险。

主要风险：

- webhook 无登录上下文，无法重新做 tenant / organization scope 校验，只能信任本地订单或充值单。
- 如果创建时 ownership 错误，webhook 会把错误 ownership 继续传播到订阅、充值记录和用户额度变化。
- 若未来支持后台补单、重放或人工修复，需要额外审计日志和二次确认。

## 4. 高风险链路

| 优先级 | 链路 | 风险 | 建议 |
| --- | --- | --- | --- |
| P0 | `Redeem()` 兑换码核销 | 用户可能使用其他 tenant / organization 的有效兑换码；当前仅按 key 和状态判断 | 核销前使用 `RequiredOwnershipByUserId()` 获取用户 ownership，并与兑换码 ownership 比对 |
| P0 | `tenant_id=1` fallback | 创建链路、用户缓存异常或上下文缺失时可能写入默认 tenant | Billing mutation 应改为 fail closed，避免在账务对象上自动 fallback |
| P1 | `SubscriptionOrder` webhook 完成 | 订单创建 ownership 错误会传播到 UserSubscription 和 TopUp | 完成前校验订单用户 ownership 与订单 ownership 一致，异常时进入人工处理 |
| P1 | `AdminBindSubscription` / `AdminCreateUserSubscription` | controller 做目标用户 tenant 校验，但 model 使用 `OwnershipByUserId()` fallback | 使用严格版 ownership 获取，失败时拒绝创建 |
| P1 | UserSubscription 失效 / 删除 | controller tenant-only；model 无 ownership 防线；删除为硬删除 | 增加 mutation 前 ownership 校验，删除改为状态化或保留审计记录 |
| P2 | Redemption 更新 / 删除 / 搜索 | tenant-only，无 organization scope | 若未来开放 organization_admin，需要升级为 `AccessScope` + `AllowsOwnership()` |
| P2 | SubscriptionPreConsumeRecord 退款 | 内部按 request_id 更新，无 ownership 校验 | 保持内部调用；如开放管理入口必须加 scope |

## 5. 修复优先级

### P0：先修 fail-open 与兑换码越权

1. Billing 对象创建链路禁止 `tenant_id=1` fallback。
2. `Redeem()` 核销前校验兑换码 ownership 与核销用户 ownership。
3. 将账务创建链路从 `OwnershipByUserId()` 升级为 `RequiredOwnershipByUserId()` 或等价 fail-closed helper。

### P1：补齐账务 mutation 的 ownership 防线

1. `AdminBindSubscription` / `AdminCreateUserSubscription` 在创建前校验目标用户完整 ownership。
2. `AdminInvalidateUserSubscription` / `AdminDeleteUserSubscription` 从 tenant-only 校验升级为对象 ownership 校验。
3. `ManualCompleteTopUp` 在 model 或 service 层增加可复用的 ownership 校验入口，避免未来 controller 绕过。
4. `CompleteSubscriptionOrder` 完成前校验订单用户 ownership 与订单 ownership 是否一致。

### P2：扩展 organization scope 与审计日志

1. Redemption 搜索、详情、更新、删除统一支持 `AccessScope`。
2. UserSubscription 删除改为状态化删除或记录审计日志。
3. SubscriptionPreConsumeRecord 如需开放管理查询，必须先接入 `ApplyOwnershipScope()`。

## 6. Iteration 8.3 建议

建议 Iteration 8.3 聚焦 “Billing Ownership Fail-Closed”：

1. 新增严格 ownership helper：从 user、context、object 获取 ownership 失败时返回错误，不做 `tenant_id=1` fallback。
2. 优先改造 `Redeem()`：核销前校验兑换码与用户处于同一 tenant，并为 organization / department / distribution_channel 预留一致性校验。
3. 改造订阅绑定创建：`AdminBindSubscription` 和 `AdminCreateUserSubscription` 创建前使用严格 ownership，不允许用户不存在、缓存失败或 ownership 缺失时继续创建。
4. 改造订阅订单完成：webhook 完成订单前校验订单 ownership 与订单用户当前 ownership 一致；不一致时拒绝自动完成并记录异常。
5. 改造 TopUp 手动完成：保留 root 全局能力，tenant_admin / finance 只能完成本 tenant TopUp；organization_admin 暂不开放。
6. 为账务 mutation 增加审计日志：兑换码核销、手动完成充值、订阅绑定、订阅失效、订阅删除均应记录 actor、target、ownership 和来源。

暂不建议在 8.3 开放以下能力：

- organization_admin 执行充值完成、订阅绑定、订阅失效、兑换码创建或删除。
- tenant_admin 跨 tenant 查询或修复 webhook 订单。
- finance 执行任何会改变额度、订阅状态、兑换码状态的 mutation。
- 任意角色硬删除账务对象。

