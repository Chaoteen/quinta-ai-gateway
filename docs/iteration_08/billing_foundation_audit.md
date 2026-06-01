# Iteration 8.1 Billing Foundation Audit

审计日期：2026-06-01

目标：审计当前 Billing / TopUp / Redemption / Subscription / Wallet / Order / Invoice 相关代码和路由。本轮只新增审计文档，不修改业务代码、router、测试或功能。

## 1. 当前账务域划分

当前系统的账务能力可以分为以下几类：

| 领域 | 当前实现 | 主要模型/位置 | 结论 |
| --- | --- | --- | --- |
| 充值 | 在线充值、Stripe/Creem/Waffo/Waffo Pancake、管理员补单 | `model.TopUp`、`controller/topup*.go` | 已有充值订单与回调完成链路 |
| 卡券/兑换码 | 兑换码创建、查询、核销、失效清理 | `model.Redemption`、`controller/redemption.go` | 已有卡券模型和核销事务 |
| 订阅 | 订阅计划、订阅订单、用户订阅、订阅预扣费 | `SubscriptionPlan`、`SubscriptionOrder`、`UserSubscription`、`SubscriptionPreConsumeRecord` | 已有基础订阅能力 |
| 钱包/余额 | 无独立 Wallet 表；使用 `users.quota` 作为钱包余额 | `model.User.Quota`、`service.WalletFunding` | 有钱包语义，但缺少独立 wallet ledger |
| 账单/订单 | 充值订单 `TopUp`；订阅订单 `SubscriptionOrder`；无通用 Order 表 | `model.TopUp`、`model.SubscriptionOrder` | 订单能力分散在业务表 |
| 开票/发票 | 未发现 Invoice 模型或发票路由 | 无 | 缺失 |
| 审计日志 | topup/consume/manage/system/refund 日志 | `model.Log`、`RecordTopupLog`、`RecordConsumeLog` | 有审计日志基础，但账务 ledger 不完整 |
| 财务统计 | 日志列表、日志统计、QuotaData 聚合 | `/api/log/**`、`QuotaData` | 有消费统计，不等价于财务账本 |

## 2. 路由状态

### 充值 TopUp

- 用户自助：
  - `GET /api/user/topup/info`
  - `GET /api/user/topup/self`
  - `POST /api/user/topup`
  - `POST /api/user/pay`
  - `POST /api/user/amount`
  - `POST /api/user/stripe/pay`
  - `POST /api/user/stripe/amount`
  - `POST /api/user/creem/pay`
  - `POST /api/user/waffo/amount`
  - `POST /api/user/waffo/pay`
- 管理只读：
  - `GET /api/user/topup` 使用 `operationalFinanceReadAuth`，已支持 tenant/organization scoped 读取。
- 高风险 mutation：
  - `POST /api/user/topup/complete` 使用 `AdminAuth`，会人工完成充值并增加用户额度。
- Webhook：
  - `/api/stripe/webhook`
  - `/api/creem/webhook`
  - `/api/waffo/webhook`
  - `/api/user/epay/notify`

### 兑换码 Redemption

- 只读：
  - `GET /api/redemption/` 使用 `operationalFinanceReadAuth`，支持 organization_admin scoped 只读。
  - `GET /api/redemption/search` 使用 `financeReadAuth`，当前是 tenant scoped，不含 organization_admin。
  - `GET /api/redemption/:id` 使用 `financeReadAuth`，当前是 tenant scoped，不含 organization_admin。
- mutation：
  - `POST /api/redemption/`
  - `PUT /api/redemption/`
  - `DELETE /api/redemption/invalid`
  - `DELETE /api/redemption/:id`
  - 以上仍为 `AdminAuth`。
- 用户核销：
  - 代码存在 `model.Redeem(key, userId)`，会增加 `users.quota` 并标记兑换码已使用；本次未在 `api-router.go` 审计到对应显式 `/api/redemption/redeem` 路由。

### 订阅 Subscription

- 用户自助：
  - `GET /api/subscription/plans`
  - `GET /api/subscription/self`
  - `PUT /api/subscription/self/preference`
  - `POST /api/subscription/epay/pay`
  - `POST /api/subscription/stripe/pay`
  - `POST /api/subscription/creem/pay`
- 管理只读：
  - `GET /api/subscription/admin/plans` 使用 `billingReadAuth`。
  - `GET /api/subscription/admin/users/:id/subscriptions` 已包含 `organization_admin`，并使用 `AccessScope` 校验目标用户和订阅。
- 计划写：
  - `POST /api/subscription/admin/plans`
  - `PUT /api/subscription/admin/plans/:id`
  - `PATCH /api/subscription/admin/plans/:id`
  - 均为 `RootAuth`。
- 用户订阅 mutation：
  - `POST /api/subscription/admin/bind`
  - `POST /api/subscription/admin/users/:id/subscriptions`
  - `POST /api/subscription/admin/user_subscriptions/:id/invalidate`
  - `DELETE /api/subscription/admin/user_subscriptions/:id`
  - 当前为 `AdminAuth`。

### 日志与财务统计

- `GET /api/log/` 使用 `operationalFinanceReadAuth`，已支持 tenant/organization scoped。
- `GET /api/log/stat` 使用 `operationalFinanceReadAuth`，已支持 tenant/organization scoped。
- `GET /api/data/`、`GET /api/data/users` 仍为 `RootAuth`，且 `QuotaData` 当前没有 tenant/organization/department ownership 字段。

## 3. 分类输出

### A. 已有 billing foundation 能力

| 能力 | 当前状态 | 说明 |
| --- | --- | --- |
| 充值订单 | 已有 | `TopUp` 包含 `trade_no`、`payment_method`、`payment_provider`、`money`、`amount`、`status`、时间戳和 ownership 字段 |
| 多支付网关 | 已有 | Epay、Stripe、Creem、Waffo、Waffo Pancake 均有创建/回调/完成逻辑 |
| 支付网关防串单 | 已有 | `expectedPaymentProvider`、`ErrPaymentMethodMismatch`、guard test 覆盖部分路径 |
| 管理补单 | 已有 | `ManualCompleteTopUp` 行级锁 + 状态校验 + 用户 quota 增加 |
| 兑换码 | 已有 | `Redemption` 支持 quota、status、expired_time、核销用户、软删除 |
| 兑换码核销事务 | 已有 | `Redeem` 使用事务和行级锁，增加用户 quota 后标记兑换码 used |
| 订阅计划 | 已有 | `SubscriptionPlan` 支持价格、周期、购买上限、升级 group、总额度、重置周期 |
| 订阅订单 | 已有 | `SubscriptionOrder` 支持支付 provider、payload、状态和 ownership |
| 用户订阅 | 已有 | `UserSubscription` 支持额度、使用量、有效期、状态、重置周期、group upgrade |
| 订阅预扣费 | 已有 | `SubscriptionPreConsumeRecord` 按 request_id 幂等记录预扣费 |
| 统一计费会话 | 已有 | `BillingSession` 抽象 wallet/subscription funding，支持预扣、结算、退款 |
| 钱包计费来源 | 部分已有 | `WalletFunding` 使用 `users.quota` 作为钱包余额 |
| 审计日志 | 部分已有 | `RecordTopupLog`、`RecordConsumeLog`、`RecordTaskBillingLog` |
| tiered billing expression | 已有 | `pkg/billingexpr` 支持表达式计费、预扣估算、结算、日志展示 |

### B. ownership 已覆盖但业务边界不完整

| 对象 | ownership 字段 | 当前问题 |
| --- | --- | --- |
| `TopUp` | tenant/organization/department/distribution_channel | 读路径已支持 scoped；补单仍是 AdminAuth，缺少 finance 专属 mutation 边界和更细审计 |
| `Redemption` | tenant/organization/department/distribution_channel | 创建/更新/删除支持 tenant access，但兑换码额度发行属于财务 mutation，缺少预算/审批/ledger |
| `SubscriptionOrder` | tenant/organization/department/distribution_channel | 订单完成会生成订阅和 topup 影子记录，但没有统一 Order/Payment ledger |
| `UserSubscription` | tenant/organization/department/distribution_channel | 只读已覆盖组织范围；创建/失效/删除仍是 tenant scoped AdminAuth，缺少 finance/organization 写边界 |
| `SubscriptionPreConsumeRecord` | tenant/organization/department/distribution_channel | 有 ownership 字段和幂等 request_id，但没有管理查询/审计接口 |
| `Log` | tenant/organization/department/distribution_channel | 消费统计已 scoped；但日志是审计记录，不是不可变账本 |
| `QuotaData` | 无 ownership 字段 | 只能 root 全局读；不能安全开放 tenant/org 财务报表 |
| `User.Quota` | 用户有 ownership 字段 | quota 调整散落在充值、兑换码、钱包预扣、用户管理中，缺少统一 wallet transaction ledger |

### C. 缺失的核心 billing 模型

| 模型 | 当前状态 | 缺口 |
| --- | --- | --- |
| Wallet | 无独立表 | `users.quota` 承担钱包余额；缺少钱包账户、冻结金额、可用余额、币种、状态 |
| WalletTransaction / Ledger | 缺失 | 充值、兑换码、消费、退款、管理员调整无法统一追踪每一笔余额变化 |
| Generic Order | 缺失 | `TopUp` 和 `SubscriptionOrder` 分离，缺少统一订单抽象、订单类型、金额、支付状态、关联业务对象 |
| PaymentAttempt / PaymentEvent | 缺失 | Webhook payload 部分保存在 SubscriptionOrder；TopUp 没有统一 provider event 表 |
| Invoice | 缺失 | 未发现 Invoice 模型、发票状态、开票抬头、税务信息、发票下载/作废 |
| CreditAdjustment | 缺失 | 管理员加减用户额度没有独立审批/原因/前后余额快照模型 |
| Refund | 部分缺失 | Log 有 refund 类型，subscription preconsume 有 refund；缺少支付退款/钱包退款统一模型 |
| FinanceReportSnapshot | 缺失 | 当前依赖 Log/QuotaData 即时统计，缺少可复核财务快照 |

### D. 高风险 mutation

| 能力 | 路由/函数 | 风险 |
| --- | --- | --- |
| 管理补单 | `POST /api/user/topup/complete`、`ManualCompleteTopUp` | 直接完成订单并增加用户 quota；需要强审计、二次验证、幂等和支付凭据校验 |
| 充值 webhook 完成 | `Recharge*`、`CompleteSubscriptionOrder` | 外部回调驱动余额或订阅生成；需防串单、幂等、签名校验、payload 留存 |
| 兑换码创建 | `POST /api/redemption/` | 直接发行可兑换 quota 的资产；应有预算/审批/创建者/批次号 |
| 兑换码核销 | `Redeem` | 增加用户 quota；当前有事务锁，但缺少 ledger 明细 |
| 兑换码更新/删除/失效清理 | `PUT/DELETE /api/redemption/**` | 可改变或删除财务资产，删除会影响追溯 |
| 订阅绑定/创建 | `AdminBindSubscription`、`AdminCreateUserSubscription` | 无支付直接授予订阅额度和可能升级 group |
| 订阅失效 | `AdminInvalidateUserSubscription` | 取消订阅并可能回退用户 group |
| 订阅硬删除 | `AdminDeleteUserSubscription` | 删除账务权益记录，破坏审计链，风险最高 |
| 用户额度调整 | `UpdateUser`、`ManageUser` 相关用户管理入口及 `IncreaseUserQuota/DecreaseUserQuota` | quota 可能被直接改写或加减，缺少统一 adjustment ledger |
| 钱包预扣/退款 | `WalletFunding`、`PostConsumeQuota` | 基于用户 quota 原子加减，但没有独立交易流水 |

### E. 可优先迁移的 finance read

| 能力 | 当前状态 | 建议 |
| --- | --- | --- |
| 充值记录列表 | `GET /api/user/topup` 已有 finance/auditor/tenant/org 只读 | 保持；可补 search/filter 测试和导出限制 |
| 兑换码列表 | `GET /api/redemption/` 已有 finance/auditor/tenant/org 只读 | 保持；可补 organization_admin detail/search scoped 后再扩展 |
| 兑换码搜索/详情 | 当前 `financeReadAuth`，tenant scoped，不含 organization_admin | 可作为 8.2 低风险 read 迁移，改用 AccessScope |
| 订阅计划列表 | `GET /api/subscription/admin/plans` 已有 billing read | 保持 tenant_admin/finance/auditor/root；计划是全局产品，组织管理员不应看到管理面全量 |
| 用户订阅列表 | 已支持 organization scoped | 保持；可补 finance/organization edge cases |
| 日志列表/统计 | 已支持 operational finance read | 保持；但明确不是发票或账本 |
| SubscriptionPreConsumeRecord 查询 | 无管理路由 | 可新增 root/finance 只读审计接口，但需先确认数据敏感性 |
| QuotaData 报表 | root only | 暂不迁移，因缺少 ownership 字段 |

### F. 必须保持 root/admin 的账务能力

| 能力 | 建议边界 |
| --- | --- |
| 订阅计划创建/更新/启停 | 必须 root；计划是全局产品和价格配置 |
| 管理补单 topup complete | 当前至少保持 AdminAuth；建议未来仅 root/finance_admin + 二次验证 |
| 兑换码创建/更新/删除/失效清理 | 当前保持 AdminAuth；Billing Foundation 完成前不下放 organization_admin |
| 订阅绑定/创建 | 当前保持 AdminAuth；需要 ledger、审批、预算后再考虑 tenant finance |
| 订阅失效 | 当前保持 AdminAuth；可未来给 tenant finance，但必须审计 |
| 订阅硬删除 | 建议长期 root-only 或废弃，改为取消/作废，不应下放 |
| 用户额度直接调整 | 保持 admin/root；应改为独立 credit adjustment 能力 |
| QuotaData 全局报表 | 保持 root，直到补 ownership 或重建 scoped 报表 |
| 支付 webhook 配置和 provider secret | 必须 root |

## 4. 权限边界建议

### root

- 管理全局产品目录：订阅计划、模型价格、ratio、支付配置。
- 访问全局财务统计、全局 QuotaData。
- 执行高风险修复、迁移、删除类操作。
- 可查看跨 tenant 账务审计。

### tenant_admin

- 可读取本 tenant 的充值、兑换码、用户订阅、消费日志、统计。
- 可执行部分 tenant 级账务 mutation 的候选，但必须等 ledger/审计/二次验证：
  - 创建兑换码；
  - 绑定订阅；
  - 失效订阅；
  - 管理补单不建议优先开放。

### finance

- 应优先获得 tenant scoped read：
  - topup list/search；
  - redemption list/search/detail；
  - user subscription read；
  - log/stat；
  - future ledger read。
- mutation 应非常有限：
  - 可考虑“发起调整申请/作废申请”，不直接完成。
  - 不应直接拥有 topup complete、硬删除订阅、删除兑换码。

### organization_admin

- 仅适合组织内只读：
  - 本组织 topup；
  - 本组织 redemption list；
  - 本组织 user subscriptions；
  - 本组织 logs/stat。
- 不应拥有：
  - 兑换码发行；
  - 订阅授予；
  - 用户额度调整；
  - 补单；
  - 发票/订单全局配置。

### auditor

- 只读，不执行 mutation。
- 可读取 tenant scoped 财务记录、日志、未来 ledger。

## 5. 领域结论

### 充值

已有 `TopUp` 订单模型和多 provider 完成链路；回调与补单会直接增加 `users.quota`。缺少统一 payment event、wallet transaction 和不可变 ledger。

### 卡券/兑换码

已有兑换码资产模型和核销事务；核销会增加用户 quota。创建、更新、删除都是高风险财务资产 mutation，需要预算与审计。

### 订阅

已有从 Plan 到 Order 到 UserSubscription，再到 PreConsumeRecord 的基础闭环。订阅消费支持幂等预扣与退款。但管理写操作仍缺少审批、ledger、变更原因和更细角色边界。

### 钱包/余额

钱包语义存在于 `WalletFunding`，余额存放在 `users.quota`。这不是完整 Wallet Foundation，因为缺少账户表、冻结余额、交易流水、余额快照。

### 账单/订单

充值订单是 `TopUp`，订阅订单是 `SubscriptionOrder`。没有通用 Order 模型；`TopUp` 同时承担充值订单和订阅完成后的 topup 影子记录，语义混合。

### 开票/发票

未发现 Invoice 模型、路由或服务。当前没有发票 foundation。

### 审计日志

`Log` 覆盖 topup、consume、manage、system、error、refund 类型，并带 ownership。它可用于操作审计和消费统计，但不能替代不可变财务流水。

## 6. Iteration 8.2 推荐开发清单

建议 8.2 仍以只读和 foundation 补齐为主，不开放高风险 mutation：

1. 补账务只读边界：
   - 将 `GET /api/redemption/search`、`GET /api/redemption/:id` 改为 `AccessScope`，允许 organization_admin 读取本组织范围。
   - 补 finance/auditor/organization_admin 的 topup/redemption/subscription/log scoped 测试。
2. 新增 Billing Foundation 文档/模型设计，不立即接入生产写路径：
   - `WalletLedger` / `WalletTransaction`
   - `CreditAdjustment`
   - `PaymentEvent`
   - `Invoice`
3. 为高风险 mutation 增加审计字段设计：
   - actor_user_id、reason、before/after quota、source object、request_id、idempotency_key。
4. 暂缓开放 mutation：
   - topup complete；
   - redemption create/update/delete；
   - subscription bind/create/invalidate/delete；
   - user quota direct adjustment。
5. 评估 `QuotaData` ownership：
   - 若要开放 tenant/org 财务报表，需要增加 ownership 字段或改用 `logs` scoped 聚合。
6. 明确 `TopUp` 与 `SubscriptionOrder` 的关系：
   - 是否保留订阅完成后 upsert TopUp 作为收入记录；
   - 是否引入通用 Order/PaymentEvent 后将 TopUp 降级为充值业务视图。

## 7. 验收

本轮只新增文档，预期验收命令：

```bash
go test ./common ./model ./controller ./service ./router ./middleware
```

如果默认 Go build cache 只读失败，使用：

```bash
GOCACHE=/tmp/quinta-go-build-cache go test ./common ./model ./controller ./service ./router ./middleware
```
