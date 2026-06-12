# Iteration 11.1 发票体系基础能力

## 本轮目标

本轮建设 Invoice Workflow Foundation，目标是形成企业客户开票资料、发票申请、财务审核、人工开票结果登记、文件 URL 记录和用户查看状态的基础闭环。

本轮只做平台内部工作流，不做任何真实开票系统对接。

## 模型设计

### InvoiceProfile

企业或个人开票资料，保存：

- 租户、组织、部门、用户 Ownership 字段。
- `profile_type`：`COMPANY`、`PERSONAL`。
- 抬头、税号、开户行、银行账号、地址、电话。
- 收件人姓名、电话、邮箱、地址。
- `is_default`：同一用户下默认资料互斥。
- `status`：`ACTIVE`、`DISABLED`。

### InvoiceApplication

发票申请记录，保存：

- `application_no` 申请编号。
- Ownership 字段和 `user_id`。
- `invoice_profile_id`。
- `amount`、`currency`。
- `invoice_type`：`VAT_NORMAL`、`VAT_SPECIAL`。
- `status`：`PENDING`、`APPROVED`、`REJECTED`、`ISSUED`、`CANCELED`。
- `source_type`：本轮主要使用 `PAYMENT_ORDER`。
- `source_id`：PaymentOrder ID。
- 审核人、审核时间、审核备注。
- 人工登记的 `invoice_no`、`invoice_date`、`issued_at`。

### InvoiceFile

发票文件记录，保存：

- `invoice_application_id`
- `file_name`
- `file_url`
- `file_type`：`PDF`、`IMAGE`、`OFD`、`OTHER`
- `uploaded_by`
- `created_at`

文件本轮只记录 URL，不实现对象存储上传、不对接税务系统。

## 开票资料设计

- 普通用户只能创建和查看自己的开票资料。
- tenant_admin 可在本租户范围查看和创建资料。
- finance/root 可查看财务范围资料。
- 当 `is_default = true` 时，同一用户下其他资料自动取消默认。
- COMPANY 类型要求 `title` 和 `tax_no`。
- PERSONAL 类型要求 `title`。
- 禁用资料不会删除历史记录，只将 `status` 改为 `DISABLED` 并取消默认。

## 发票申请状态机

```text
PENDING
  ├── APPROVED
  │     └── ISSUED
  └── REJECTED
```

- 用户提交申请后初始为 `PENDING`。
- 财务或 tenant_admin 可审核为 `APPROVED` 或 `REJECTED`。
- 只有 `APPROVED` 状态可以人工登记开票结果。
- 登记开票结果后状态变为 `ISSUED`，同时写入发票号码、开票日期和文件记录。

## 可开票金额规则

本轮采用最小闭环：

- 只基于 `PaymentOrder` 开票。
- `PaymentOrder.status = PAID` 的 `amount` 是可开票总额。
- 已占用/已开票金额按同一 PaymentOrder 下 `PENDING`、`APPROVED`、`ISSUED` 的 `InvoiceApplication.amount` 汇总。
- 可申请金额 = 已支付金额 - 已占用/已开票金额。
- 未支付订单不能申请发票。
- 超额申请会被拒绝。
- 审核通过前再次校验剩余可开票金额。

本轮不处理：

- 退款后的可开票金额调整。
- 多订单合并开票。
- 跨币种开票。
- 税率、税额计算。
- 发票红冲、作废、冲正。

## 权限边界

### user

- 创建自己的开票资料。
- 查看自己的开票资料。
- 禁用自己的开票资料。
- 基于自己的已支付订单提交发票申请。
- 查看自己的申请和文件。

### tenant_admin

- 查看本租户发票资料、申请和文件。
- 审核本租户申请。
- 人工登记本租户已审核申请的开票结果。

### finance

- 查看全部财务发票资料、申请和文件。
- 审核发票申请。
- 人工登记开票结果和文件 URL。

### root

- 全部权限。

所有查询复用 Ownership Scope。普通用户始终附加 `user_id = 当前用户` 条件。

## API 说明

### 用户侧

- `POST /api/invoices/profiles`
- `GET /api/invoices/profiles`
- `POST /api/invoices/profiles/:id/disable`
- `POST /api/invoices/applications`
- `GET /api/invoices/applications`
- `GET /api/invoices/files`

### 管理侧

- `GET /api/admin/invoices/applications`
- `POST /api/admin/invoices/applications/:id/review`
- `POST /api/admin/invoices/applications/:id/issue`
- `GET /api/admin/invoices/profiles`
- `POST /api/admin/invoices/profiles`
- `GET /api/admin/invoices/files`

## 前端说明

新增用户侧菜单：

- `Invoice`

页面包含：

- Invoice Profiles：创建、查看、禁用开票资料。
- Invoice Applications：选择已支付 PaymentOrder、选择开票资料、提交申请、查看状态。
- Invoice Files：查看文件记录和打开文件 URL。

新增管理侧菜单：

- `Invoice Management`

页面包含：

- Applications：状态筛选、审核通过、审核拒绝、人工开票登记。
- Profiles：查看开票资料。
- Files：查看发票文件记录。

## 测试说明

新增测试覆盖：

- model：表迁移和字段规范化。
- service：资料创建、默认资料互斥、已支付校验、未支付拒绝、超额拒绝、审核、开票、文件记录、Ownership 隔离。
- controller：资料创建和申请提交响应。
- router：RBAC 和租户隔离。

验收命令：

```bash
go test ./model ./service ./controller ./router ./middleware
cd web/default && npm run typecheck
cd web/default && npm run build
```

## 本轮不做的边界

本轮明确不实现：

- 真实数电票接口。
- 税控设备接口。
- 电子税务局接口。
- 乐企平台、航信、百望、税友、票通等服务商 SDK。
- 税务供应商模拟接口。
- 自动开票 API。
- 真实发票查验、红冲、作废。
- 对象存储上传。

真实数电票、税控设备、电子税务局和第三方服务商对接，留到后续 `Invoice Provider Integration` 迭代。
