# Iteration 10.7 Payment Gateway Foundation v2

## 本轮目标

本轮建立统一支付网关基础能力，提供 Alpha 商业闭环的支付入口：

用户创建支付订单 -> 订单待支付 -> Mock 或银行转账确认成功 -> 幂等处理支付结果 -> 按业务类型履约 -> 写入回调/确认日志 -> 后台查询和人工审核。

本轮不接入完整微信支付 SDK、支付宝 SDK、真实退款、Settlement、Invoice、Voucher 或完整 Finance Console。

## 模型设计

### PaymentOrder

统一支付订单，核心字段：

- `order_no`：唯一业务订单号。
- `tenant_id` / `organization_id` / `department_id` / `distribution_channel_id` / `user_id`：ownership 元数据。
- `provider`：`MOCK`、`WECHAT_PAY`、`ALIPAY`、`BANK_TRANSFER`。
- `business_type`：`TOKEN_RECHARGE`、`SUBSCRIPTION_PURCHASE`、`SUBSCRIPTION_RENEWAL`。
- `business_id`：业务对象 ID。本轮约定 `TOKEN_RECHARGE` 时表示补入用户钱包的 quota 数量；如果为 0，则按 `amount * QuotaPerUnit` 兜底换算。订阅购买时为 `plan_id`，订阅续费时为 `user_subscription_id`。
- `amount` / `currency`：支付金额。
- `status`：支付状态。
- `fulfillment_status` / `fulfilled_at` / `fulfillment_message`：履约状态和可追踪信息。

### PaymentCallbackLog

记录支付回调或后台确认日志，包括 `payment_order_id`、`order_no`、`provider`、`event_type`、`raw_payload`、`signature_valid`、`process_status`、`process_message`。

### BankTransferRecord

记录银行转账凭证和人工审核结果，包括 `payment_order_id`、`tenant_id`、`user_id`、转账账户、金额、凭证 URL、`review_status`、审核人和审核备注。

## 支付订单状态机

订单初始状态为 `PENDING`。

允许流转：

- `PENDING -> PAID`：确认支付成功且履约成功。
- `PENDING -> EXPIRED`：确认时订单已过期。
- `PENDING -> FAILED`：本轮仅在银行转账拒绝且审核请求显式设置 `failed_status=true` 时使用。

不允许直接确认：

- `FAILED -> PAID`
- `CANCELED -> PAID`
- `EXPIRED -> PAID`
- `REFUNDED -> PAID`

已 `PAID` 的订单重复确认只写 `PaymentCallbackLog`，不会重复履约。

## 支付履约链路

`ConfirmPayment` 在事务中锁定 `PaymentOrder`，检查状态并调用履约：

- `TOKEN_RECHARGE`：原子增加 `users.quota`。本轮使用最小闭环，不新增钱包 ledger；审计依赖 `PaymentOrder` 和 `PaymentCallbackLog`。
- `SUBSCRIPTION_PURCHASE`：根据 `business_id=plan_id` 创建 `UserSubscription`。
- `SUBSCRIPTION_RENEWAL`：根据 `business_id=user_subscription_id` 找到原订阅套餐，并创建新的 `UserSubscription`。

履约成功后订单写入 `status=PAID`、`fulfillment_status=SUCCESS`、`paid_at` 和 `fulfilled_at`。

## 银行转账审核流程

用户先创建 `provider=BANK_TRANSFER` 的 `PaymentOrder`，再提交 `BankTransferRecord`。

后台审核：

- 通过：`review_status=APPROVED`，随后调用 `ConfirmPayment`，触发同一条履约链路。
- 拒绝：`review_status=REJECTED`。默认保持 `PaymentOrder=PENDING`，方便用户补充凭证或财务复核；如审核请求设置 `failed_status=true`，订单进入 `FAILED`。

重复审核会被拒绝，不会重复调用支付确认。

## Mock Provider

`MOCK` 用于 Alpha 阶段和测试环境。它不调用外部支付渠道，只通过后台或测试代码调用 `ConfirmPayment` 模拟支付成功。

Mock Provider 仍写入完整订单、履约状态和 callback log，因此它与真实 provider 共享同一履约和幂等逻辑。

## 微信/支付宝预留点

模型层已经保留 `WECHAT_PAY` 和 `ALIPAY` provider 常量。

后续真实接入时应补充：

- provider-specific 下单参数和支付链接/二维码返回。
- webhook 验签，写入 `signature_valid`。
- provider transaction id 或原始载荷字段扩展。
- 失败、关闭、退款事件的状态映射。

真实 provider 接入后仍应只通过 `ConfirmPayment` 进入履约，不能绕过统一幂等链路。

## 幂等设计

幂等边界在 `ConfirmPayment`：

- 事务内锁定订单。
- 只有 `PENDING` 可以执行履约。
- 履约成功后才更新为 `PAID`。
- 重复回调看到 `PAID` 后直接返回，不重复增加 quota 或创建订阅。
- 每次确认尝试都会写 `PaymentCallbackLog`，便于排查重复回调或异常确认。

银行转账审核也有幂等保护：只有 `review_status=PENDING` 的记录可以审核。

## 权限边界

用户侧 API：

- `POST /api/payment/orders`
- `GET /api/payment/orders`
- `GET /api/payment/orders/:id`
- `POST /api/payment/bank-transfer`

普通用户只能查询和提交自己的订单/凭证。

管理/财务 API：

- `GET /api/admin/payment/orders`
- `GET /api/admin/payment/callback-logs`
- `GET /api/admin/payment/bank-transfers`
- `POST /api/admin/payment/bank-transfers/:id/review`

权限规则：

- root 可看全部。
- tenant_admin 和 finance 只能看本 tenant。
- 普通 user、ops、auditor、organization_admin 不能访问本轮财务审核 API。

查询实现复用现有 `AccessScope` 和 ownership scope。

## 测试说明

新增测试覆盖：

- PaymentOrder 创建成功。
- `amount <= 0` 拒绝创建。
- `ConfirmPayment` 幂等。
- 已 `PAID` 重复确认不重复履约。
- `FAILED` / `CANCELED` / `EXPIRED` 不能确认成 `PAID`。
- 银行转账审核通过触发 `ConfirmPayment`。
- 银行转账重复审核不能重复履约。
- 用户只能查询自己的支付订单。
- finance / tenant_admin / root 查询范围符合权限边界。
- `PaymentCallbackLog` 正确写入。

已执行：

```bash
GOCACHE=/tmp/quinta-go-build-cache go test ./model ./service ./controller ./router ./middleware
```
