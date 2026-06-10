# Iteration 10.8 Billing Portal Foundation

## 本轮目标

本轮建设用户账单中心能力，补齐 Alpha 商业闭环中的查询与展示入口：

付款 -> 获得额度 -> 调用模型 -> 扣减额度 -> 查看账单、消费、支付和订阅状态。

本轮只做 Billing Portal 查询层和前端展示层，不修改 Billing Runtime 的计费、扣费、结算或账单生成逻辑。

## 架构

Billing Portal 延续现有分层：

- `router`：注册 `/api/billing/*` 用户账单中心路由。
- `controller`：解析分页、时间范围和筛选条件，返回统一 JSON 响应。
- `service`：聚合 `PaymentOrder`、`QuotaUsage`、`BillingRecord`、`UserSubscription`、`User` 等模型数据。
- `model`：复用现有模型和 ownership scope，不新增计费写入模型。
- `web/default`：新增 Billing Portal 页面和侧边栏入口。

核心服务为 `BillingPortalService`，所有查询都会先计算当前操作者的可见范围，再把 scope 下推到 GORM 查询。

## 统计逻辑

`GET /api/billing/summary` 返回当前用户或当前权限范围内的汇总信息：

- 当前余额：汇总可见用户的 `users.quota`。
- 当前 Subscription：查询可见范围内未过期且启用的 `UserSubscription`。
- 累计充值金额：汇总已支付 `PaymentOrder` 的 `amount`。
- 累计消费金额：汇总 `BillingRecord.quota_charged`，本轮口径为 quota 扣减量，返回货币标识为 `QUOTA`，不把它解释为法币。
- 累计 Token 消耗：汇总 `QuotaUsage.used_quota`。
- 累计请求次数：统计 `QuotaUsage` 记录数。
- 最近 30 天消费金额：按 `BillingRecord.created_at >= now - 30d` 汇总 `quota_charged`。
- 最近 30 天 Token 消耗：按 `QuotaUsage.created_at >= now - 30d` 汇总 `used_quota`。
- 最近 30 天请求次数：统计最近 30 天 `QuotaUsage` 记录数。
- 模型消费排行：按 `BillingRecord.model` 汇总 `quota_charged` 和请求数。
- Provider 消费排行：按 `BillingRecord.provider` 汇总 `quota_charged` 和请求数。

本轮不会回写账单、额度或订阅状态，因此不会影响 Billing Runtime 的既有行为。

## 支付记录

`GET /api/billing/payments` 基于 `PaymentOrder` 查询支付记录。

支持筛选：

- `p` / `page_size`：分页。
- `start_time` / `end_time`：创建时间范围，Unix 秒。
- `status`：支付状态。
- `payment_method` 或 `provider`：支付方式。

普通用户只能看到自己的支付订单；管理角色按 ownership scope 扩大查询范围。

## 消费明细

`GET /api/billing/usages` 基于 `QuotaUsage` 查询消费明细。

支持筛选：

- `p` / `page_size`：分页。
- `start_time` / `end_time`：创建时间范围，Unix 秒。
- `provider`：Provider。
- `model`：模型。

该接口用于展示模型调用后的额度消耗流水，不参与扣费。

## 账单记录

`GET /api/billing/records` 基于 `BillingRecord` 查询账单记录。

支持筛选：

- `p` / `page_size`：分页。
- `start_time` / `end_time`：创建时间范围，Unix 秒。
- `provider`：Provider。
- `model`：模型。
- `tenant_id`：租户筛选，仅在调用者 scope 允许时生效。

`BillingRecord` 仍由 Billing Runtime 生成，Billing Portal 只读取和展示。

## 订阅中心

`GET /api/billing/subscriptions` 基于 `UserSubscription` 查询订阅。

支持 `subscription` 参数：

- `active`：当前有效订阅。
- `history`：历史订阅。
- `expiring`：即将到期订阅，当前定义为 7 天内到期且仍启用。

不传参数时返回可见范围内的订阅列表。

## 权限边界

Billing Portal 复用现有 RBAC 与 ownership scope：

- `root`：查看全部。
- `tenant_admin`：查看本 tenant 范围。
- `organization_admin`：查看本 organization 范围。
- 普通 `user`：只能查看自己的账单、支付、消费和订阅。

前端 Billing 菜单对登录用户可见；真正的数据边界由后端 scope 强制执行。后端不会信任前端传入的 tenant、organization 或 user 参数来扩大权限。

## 接口说明

用户侧账单中心接口：

- `GET /api/billing/summary`：账单概览。
- `GET /api/billing/payments`：支付记录。
- `GET /api/billing/usages`：消费明细。
- `GET /api/billing/records`：账单记录。
- `GET /api/billing/subscriptions`：订阅中心。

所有接口需要登录态，返回内容均受当前用户 ownership scope 约束。

## 前端页面

新增 Billing Portal 页面：

- 路由：`/billing`
- 菜单：Personal -> Billing
- 页面标签：
  - Overview
  - Payments
  - Usage
  - Bills
  - Subscriptions

Overview 展示当前额度、累计充值、累计消费、最近 30 天消费、请求次数、模型消费排行和 Provider 消费排行。

## 测试说明

新增测试覆盖：

- Summary 统计。
- 支付记录分页查询。
- 消费、账单和订阅查询的基础筛选。
- 普通用户 ownership 隔离。
- `organization_admin`、`tenant_admin`、`root` 的查询边界。
- Router 层 Billing Portal 路由注册和登录访问。

建议执行：

```bash
GOCACHE=/tmp/quinta-go-build-cache go test ./model ./service ./controller ./router ./middleware
```

前端建议执行：

```bash
npm run typecheck
npm run build
```

本地环境如果安装了 Bun，也可以按项目约定使用 `bun run typecheck` 和 `bun run build`。
