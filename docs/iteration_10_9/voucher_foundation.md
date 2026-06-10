# Iteration 10.9 Voucher Foundation

## 目标

本轮建设统一卡券体系，支持渠道发卡、代理销售、活动赠送、Subscription 月卡和 Token 充值卡。

核心闭环：

生成兑换码 -> 发放兑换码 -> 用户核销 -> 增加 Quota 或开通 Subscription -> 生成核销记录。

本轮只实现 Voucher Foundation，不做 Settlement、Invoice、Voucher 财务核销报表或完整营销活动系统。

## 架构

后端延续 Router -> Controller -> Service -> Model 分层：

- `model.VoucherBatch`：卡券批次。
- `model.Voucher`：单个兑换码。
- `model.VoucherRedemption`：核销审计记录。
- `service.VoucherService`：批次创建、批量生成、核销履约、禁用和查询。
- `controller/voucher.go`：用户侧和后台侧 API。
- `router/api-router.go`：接入用户和后台路由，复用现有 RBAC。

前端新增：

- 用户侧 `/vouchers`：兑换码输入和个人核销历史。
- 管理侧 `/admin/vouchers`：批次、卡券列表、核销历史。
- 侧边栏菜单：
  - Personal -> Voucher
  - Admin -> Voucher

## 模型设计

### VoucherBatch

用于管理批次，包含 `batch_no`、`name`、`description`、`voucher_type`、`quantity`、`status`、ownership 字段和 `created_by`。

状态：

- `DRAFT`
- `ACTIVE`
- `DISABLED`
- `FINISHED`

本轮后台创建批次时默认置为 `ACTIVE`，便于 Alpha 阶段直接生成和核销；模型层仍保留 `DRAFT` 作为后续审批/发布流程预留。

### Voucher

用于表示单个兑换码，包含 `batch_id`、`voucher_code`、`voucher_type`、`quota_amount`、`subscription_plan_id`、`status`、`activated_by`、`activated_at`、`expired_at`。

类型：

- `TOKEN`
- `SUBSCRIPTION`

状态：

- `UNUSED`
- `REDEEMED`
- `EXPIRED`
- `DISABLED`

### VoucherRedemption

用于记录核销结果，包含 `voucher_id`、`voucher_code`、`user_id`、ownership 字段、`redemption_type`、`redemption_result` 和 `created_at`。

本轮成功核销写入 `SUCCESS`。重复同一用户提交已成功核销的同一兑换码时返回已有核销记录，不重复写入。

## 兑换码生成规则

`GenerateVouchers` 支持：

- 指定 `quantity` 批量生成随机码。
- 指定 `codes` 生成固定码，主要用于测试或外部渠道导入。
- 兑换码统一 `TrimSpace` 并转大写。
- 生成前检查批次内和全局表内重复，`voucher_code` 有唯一约束。
- 随机码格式为 `VCH` 加 16 位随机字符串。

TOKEN 卡必须提供 `quota_amount > 0`。

SUBSCRIPTION 卡必须提供 `subscription_plan_id > 0`，并且该套餐存在。

## 核销流程

`RedeemVoucher` 在事务内执行：

1. 按 `voucher_code` 锁定 Voucher。
2. 查询所属 VoucherBatch。
3. 检查批次或兑换码是否禁用。
4. 检查是否过期。
5. 检查是否已经核销。
6. 根据 `voucher_type` 执行履约。
7. 更新 Voucher 为 `REDEEMED`。
8. 写入 `VoucherRedemption`。

如果兑换码已由同一用户成功核销，直接返回既有 `VoucherRedemption`，不重复履约。

如果兑换码已由其他用户核销，返回已核销错误。

## Quota 履约

TOKEN 卡核销时在同一事务中对 `users.quota` 执行原子增加：

```text
quota = quota + voucher.quota_amount
```

本轮复用现有用户 quota 闭环，审计依据为 `Voucher` 和 `VoucherRedemption`。后续如引入钱包 ledger，可将该履约改为调用 ledger/Quota Runtime 的补额入口，但仍应保持 `RedeemVoucher` 的统一幂等边界。

## Subscription 履约

SUBSCRIPTION 卡核销时读取 `SubscriptionPlan`，调用现有：

```go
model.CreateUserSubscriptionFromPlanWithOwnershipTx(tx, userId, plan, "voucher", ownership)
```

创建的 `UserSubscription` 带有核销用户的 ownership，`source` 标记为 `voucher`。

## 权限边界

用户侧：

- `POST /api/vouchers/redeem`
- `GET /api/vouchers/history`

普通用户只能核销兑换码并查看自己的核销记录。

后台侧：

- `POST /api/admin/vouchers/batches`
- `GET /api/admin/vouchers/batches`
- `GET /api/admin/vouchers`
- `GET /api/admin/vouchers/redemptions`
- `POST /api/admin/vouchers/:id/disable`
- `POST /api/admin/voucher-batches/:id/generate`
- `POST /api/admin/voucher-batches/:id/disable`

权限规则：

- `root`：全部。
- `tenant_admin`：本 tenant 内创建、生成、查询、禁用。
- `finance`：只能查看核销记录。
- `user`：不能访问后台接口。

所有后台查询和变更都通过 `AccessScope` 和 ownership 校验限制范围。`Voucher` 本身通过所属 `VoucherBatch` 继承批次 ownership，`VoucherRedemption` 自带核销用户 ownership。

## 前端页面

用户侧 `Voucher Redemption` 页面提供：

- 输入兑换码。
- 提交核销。
- 展示最近一次核销结果。
- 查看个人核销历史。

管理侧 `Voucher` 页面提供三个标签：

- `Voucher Batch`：创建批次、生成兑换码、批次分页、搜索、状态筛选、类型筛选、禁用批次。
- `Voucher List`：卡券分页、搜索、状态筛选、类型筛选、批次 ID 筛选、禁用未使用兑换码。
- `Redemption History`：核销记录分页、搜索、结果状态筛选、类型筛选。

前端 RBAC：

- `root`、`tenant_admin` 可以进入完整管理页。
- `finance` 可以进入管理页但只显示核销历史。
- 普通用户只显示用户侧核销入口和个人历史。

## 幂等设计

幂等边界在 `RedeemVoucher`：

- 事务内锁定兑换码。
- 只有 `UNUSED` 可执行履约。
- 履约完成后才更新为 `REDEEMED` 并写核销记录。
- 同一用户重复请求已核销兑换码时返回已有成功核销记录。
- 其他用户重复使用已核销兑换码会被拒绝。

禁用操作不会删除历史记录。已核销兑换码不能再禁用为未使用状态。

## 测试说明

新增测试覆盖：

- Voucher 模型迁移和默认规范化。
- 批次创建。
- 批量生成兑换码。
- 重复兑换码校验。
- Token 卡核销增加 quota。
- Subscription 卡核销开通订阅。
- 重复核销保护。
- 过期码校验。
- 禁用码校验。
- 后台 RBAC 校验。
- tenant ownership 隔离。

已验证命令：

```bash
GOCACHE=/tmp/quinta-go-build-cache go test ./model ./service ./controller ./router ./middleware
npm run typecheck
npm run build
```
