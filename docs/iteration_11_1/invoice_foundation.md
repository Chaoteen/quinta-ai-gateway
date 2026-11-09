# Iteration 11.1 发票体系基础能力

## 2026-11-09 实现更新与商用接入设计

本节优先于下文历史设计。产品方案及官方资料见 `docs/user-guide/invoice.md`。已确认平台为一般纳税人、可开普票/专票，没有现成开票服务商。

### 本次实际修改

- 新申请限人民币、合法票种和支付订单来源；拒绝 NaN、无穷大、负数、分以下金额及超过订单字段容量的金额。个人抬头不能申请专票。
- `InvoiceApplication` 新增 nullable `amount_minor`（整数分）、`snapshot_version`、`buyer_snapshot` 及 nullable unique `issued_identity`。申请固定购买方资料，后续修改抬头资料不会覆盖申请；审核和登记校验快照完整性。
- 可申请金额计算不再使用 SQLite 可能以浮点累加的 `SUM(amount)`，改为验证历史行后按整数分扣减。创建仍锁定付款订单；审核及登记采取条件状态更新，状态与文件记录在同一事务内提交。
- 单个平台销售方的票号通过唯一身份约束，避免一张票登记到多个申请；同时检查历史已登记票号。新申请未登记时为 NULL，不会因大量空票号冲突。上线前仍需核对历史重复票号，不能自动为未知旧票补造新身份。
- root 或明确属于 `PLATFORM_TENANT_ID` 的 finance 可以审核/登记，且不得审核自己的申请；请求中的操作人必须匹配当前身份。企业 tenant_admin 和企业 finance 不代表平台销售方。
- 未配置 `PLATFORM_TENANT_ID` 时普通 finance 不再获得跨企业查询权限。明确属于平台的财务和审计可以跨企业读取发票资料；审计不能代客户创建资料、申请、审核或登记。该平台租户必须由受控运维保留给工作人员，不能是普通注册用户共享的默认租户。
- `GET /api/invoices/capabilities` 返回当前操作权限，前端据此禁用登记、隐藏不可执行的审核操作，并限制重复提交；新增中英文提示。非人民币旧订单不出现在新申请候选列表。
- 文件登记拒绝非 HTTPS、带用户凭据或 fragment 的 URL。这不是私有文件存储实现，尚不能保证已登记外链不被转发。

旧申请的新增字段保持空值。已开历史票继续可查；缺少可信快照的未开申请不能自动审核通过或登记，需财务核对、拒绝旧申请并重新提交，不能用当前抬头静默补齐旧历史。

### 本次验证与尚未完成

- Go 1.26.10：`service`、`controller`、`router`、`model` 完整包测试通过。新增七组服务安全测试以及六种身份的权限接口测试；12 个并发申请竞争 3 分额度，仅 3 个成功。
- 前端类型检查、发票模块 ESLint、格式化及默认主题生产构建通过。
- 本地数据库测试使用 SQLite（单连接）；MySQL/PostgreSQL 的并发锁定、迁移与索引需执行已有 CI，不能用本地测试替代。尚未进行真实登录浏览器端到端验收或真实税票验收。
- 仍只支持支付订单额度占用，没有消费可开票台账、按月合并申请、退款/补贴开票规则、自动税务接口及私有文件下载。本次不宣称已达到商业上线条件。

### 后续账务与开票架构

1. `InvoiceableItem`：由服务结算生成的可开票明细，关联客户主体、消费、实付资金来源、合同/税务规则版本、退款和补贴来源。套餐购买与其内部额度消费必须选择唯一确认路径，不能双重计入。
2. `InvoiceAllocation`：申请占用明细和整数分金额；在事务中按固定次序锁定明细并分配，拒绝并发重复开票。支付订单来源与消费来源共享同一额度占用，不得各算一套。
3. `InvoiceSeller`：平台真实销售方、税号、授权和凭据引用；企业租户为买方。首期单销售方；多主体后应按销售方与票号联合去重，不沿用当前单主体唯一票号假设。
4. `InvoiceIssueJob` / outbox：审核通过后在事务中创建唯一开票任务；事务外请求服务商。对方已受理但本地超时进入 `QUERY_PENDING`，先查询结果，不用新请求号盲目重开。
5. `InvoiceProviderEvent`：验签/验身份后保存唯一事件，核对销售方、申请号、票号、价税合计及票种，再幂等更新；异步通知与主动查询使用同一状态推进规则。
6. 红字工作流：保留蓝票，关联退款、确认单、红票及净开票额；按真实结果更新额度。申请撤销不等于税票作废，开票不明状态不能直接释放额度给下一次申请。
7. `InvoiceDocument`：私有对象存储文件引用，下载前重新鉴权，短时链接或流式代理交付；记录来源、校验摘要和下载审计，不把永久公共 URL 当成交付权限。
8. 每日核对消费可开票台账、申请占用、真实开票结果及红字结果；月度账单展示客户支付、商业优惠、政府补贴和税票分摊，差异进入财务待处理列表。

开发顺序为消费台账和额度分配 → 财务人工开票完整交付 → 退款/红字及对账 → 选定服务商的自动化适配。税收编码、税率、补贴对应发票主体与金额由财务确认后配置成版本化规则；不根据竞品宣传写死。

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
