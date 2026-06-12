# Iteration 11.0 财务运营后台基础能力

## 本轮目标

本轮建设 Finance Console Foundation，为 root、finance、tenant_admin 提供统一运营数据入口。系统在现有支付、卡券、订阅、Usage、Billing Runtime、Revenue Share 基础上，仅新增查询和聚合能力，不修改既有计费、履约、扣费逻辑。

Finance Console 覆盖：

- 收入 KPI：累计充值、本月充值、近 30 天充值、订单数、支付成功率。
- 消费 KPI：累计消费、本月消费、近 30 天消费、请求数、Token 消耗。
- 活跃度 KPI：活跃租户、用户、订阅、渠道。
- Top 排行：租户、模型、Provider、渠道。
- 最近活动：支付、卡券核销、订阅、账单记录。
- Payment、Voucher、Revenue Share、Tenant 维度运营概览。

## 架构

新增后端服务：

- `service/finance_console.go`

新增后端 Controller：

- `controller/finance_console.go`

新增路由：

- `GET /api/admin/finance/summary`
- `GET /api/admin/finance/top-tenants`
- `GET /api/admin/finance/top-models`
- `GET /api/admin/finance/top-providers`
- `GET /api/admin/finance/top-channels`
- `GET /api/admin/finance/recent-payments`
- `GET /api/admin/finance/recent-redemptions`
- `GET /api/admin/finance/recent-subscriptions`
- `GET /api/admin/finance/recent-billing`

新增前端：

- `web/default/src/features/finance-console`
- `web/default/src/routes/_authenticated/admin/finance/index.tsx`
- Sidebar 菜单 `Finance`
- RBAC 权限 `FINANCE_CONSOLE`

## 统计口径说明

### 收入口径

收入基于 `PaymentOrder`：

- 累计充值金额：`status = PAID` 的 `amount` 汇总。
- 本月充值金额：本月 1 日 00:00 起，`status = PAID` 的 `amount` 汇总。
- 近 30 天充值金额：最近 30 天内，`status = PAID` 的 `amount` 汇总。
- 支付订单数：当前权限范围内全部支付订单数。
- 支付成功率：`PAID` 订单数 / 全部支付订单数。

支付 Dashboard 支持 `7/30/90` 天窗口，统计支付金额、订单数、成功率、Provider 占比和按天趋势。

### 消费口径

消费基于 `BillingRecord`：

- 累计消费金额：`quota_charged` 汇总。
- 本月消费金额：本月 1 日 00:00 起的 `quota_charged` 汇总。
- 近 30 天消费金额：最近 30 天内的 `quota_charged` 汇总。
- 累计请求次数：`request_count` 汇总。
- 累计 Token 消耗：`total_tokens` 汇总。

当前消费金额单位为 `QUOTA`，不做法币换算。

### 活跃度口径

- 活跃租户：`Tenant.status = 1`。
- 活跃用户：`User.status = common.UserStatusEnabled`。
- 活跃订阅：`UserSubscription.status = active` 且 `end_time > now`。
- 活跃渠道：`Channel.status = common.ChannelStatusEnabled`。

### Voucher 口径

Voucher Dashboard 基于 `VoucherBatch`、`Voucher`、`VoucherRedemption`：

- 发卡总量：权限范围内 Voucher 数量。
- 核销总量：`Voucher.status = REDEEMED`。
- 未核销总量：`Voucher.status = UNUSED`。
- 核销率：核销总量 / 发卡总量。
- 批次数量：权限范围内 VoucherBatch 数量。
- 活跃批次数量：`VoucherBatch.status = ACTIVE`。

### Revenue Share 口径

Revenue Share Dashboard 基于 `RevenueShareRecord`：

- 平台收益：`platform_amount` 汇总。
- 总代理收益：`master_distributor_amount` 汇总。
- 分销收益：`distributor_amount` 汇总。
- 渠道排行：按 `gross_amount` 汇总倒序。

## Dashboard 设计

前端 Finance 页面包含三个 Tab：

- Dashboard：收入 KPI、消费 KPI、活跃度 KPI、Payment Dashboard、Voucher 和 Revenue Share 概览。
- Top Ranking：Top Tenants、Top Models、Top Providers、Top Channels。
- Recent Activity：Recent Payments、Recent Redemptions、Recent Subscriptions、Recent Billing。

页面使用现有 UI 组件体系：`SectionPageLayout`、`Card`、`Tabs`、`Table`、`Badge`、`Button`。页面不展示原始 JSON，而是以 KPI 卡片和表格呈现运营数据。

## 权限边界

后端路由使用：

- `middleware.RoleAuth(common.RoleKeyTenantAdmin, common.RoleKeyFinance)`

权限范围：

- root：通过现有 root bypass 访问全部数据。
- finance：访问全部财务运营数据。
- tenant_admin：仅访问本租户数据。
- user、ops、auditor：禁止访问 Finance Console。

数据查询复用 `model.AccessScopeFromContext` 与 `model.ApplyOwnershipScope`。由于 `tenants` 表主键为 `id`，不包含 `tenant_id`，租户活跃数使用专门的 tenant scope：tenant_admin 仅统计 `tenants.id = scope.tenant_id`。

## API 说明

### `GET /api/admin/finance/summary`

参数：

- `days`：支付 Dashboard 时间窗口，支持 `7`、`30`、`90`，默认 `30`。

返回：

- `revenue`
- `consumption`
- `activity`
- `payment`
- `voucher`
- `revenue_share`
- `tenant`

### Top 排行接口

通用参数：

- `p`
- `page_size`

接口：

- `GET /api/admin/finance/top-tenants`
- `GET /api/admin/finance/top-models`
- `GET /api/admin/finance/top-providers`
- `GET /api/admin/finance/top-channels`

### 最近活动接口

通用参数：

- `p`
- `page_size`

接口：

- `GET /api/admin/finance/recent-payments`
- `GET /api/admin/finance/recent-redemptions`
- `GET /api/admin/finance/recent-subscriptions`
- `GET /api/admin/finance/recent-billing`

## 测试说明

新增测试覆盖：

- `service/finance_console_test.go`
  - Summary 统计正确性。
  - Top Provider 分页。
  - tenant_admin Ownership Scope 隔离。
  - finance 全局财务视图。
- `controller/finance_console_test.go`
  - Summary Controller 响应结构和统计数据。
- `router/finance_console_test.go`
  - root、finance、tenant_admin 可访问。
  - user、ops、auditor 不可访问。
  - tenant_admin 仅看到本租户统计。

验收命令：

```bash
go test ./model ./service ./controller ./router ./middleware
cd web/default && npm run typecheck
cd web/default && npm run build
```

## 后续边界

本轮不做：

- 真实财务结算。
- Invoice、Voucher、Settlement 的完整财务凭证体系。
- 收入和 QUOTA 之间的法币换算。
- 多币种汇率折算。
- 完整 Finance Console 配置中心。

后续可在当前查询层基础上扩展导出、趋势图、筛选条件、渠道分润结算和 Invoice/Voucher 财务凭证能力。
