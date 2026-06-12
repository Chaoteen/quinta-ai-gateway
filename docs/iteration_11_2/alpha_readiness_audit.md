# Iteration 11.2 Alpha Readiness Audit

## 本轮边界

本轮只做 Alpha 发布前的产品化收口审计与上线清单，不新增业务模块，不扩展数据库模型，不新增收费逻辑，不新增 Provider。

已明确排除：Payment、Voucher、Invoice、Finance、Revenue Share、Settlement、Agent Marketplace、Skill Marketplace、Video Model 等新功能扩展。

## 当前能力矩阵

| 能力 | 当前状态 | Alpha 可用性 |
| --- | --- | --- |
| Payment Gateway | 已具备支付订单、Mock、银行转账确认与回调日志 | 可用 |
| Subscription | 已具备套餐、用户订阅、续费与状态查询 | 可用 |
| Quota Engine / Runtime | 已具备额度配置、补额、扣减和运行期校验 | 可用 |
| Usage Metering | 已具备用量记录和消费明细 | 可用 |
| Billing Runtime | 已具备账单生成和计费记录 | 可用 |
| Revenue Share | 已具备渠道收益记录和统计基础 | 可用 |
| Billing Portal | 已具备余额、支付、用量、账单、订阅查询 | 可用 |
| Voucher | 已具备批次、兑换码、核销与履约 | 可用 |
| Invoice Workflow | 已具备开票资料、申请、审核、人工开票登记和文件记录 | 可用 |
| Finance Console | 已具备收入、消费、排行和最近活动统计 | 可用 |
| Tenant / Organization / Department / Channel | 已具备多租户组织归属模型 | 可用 |
| RBAC / Admin Console | 已具备角色权限和后台入口控制 | 可用 |
| API Key | 已具备用户 API Key 管理入口 | 可用 |

## 菜单矩阵

当前前端 Sidebar 结构不超过两层：分组加菜单项。建议 Alpha 统一使用以下命名与入口。

### 未登录用户

未登录状态不进入后台 Sidebar，保留公开入口：

| 一级入口 | 说明 |
| --- | --- |
| Home | 产品首页 |
| Pricing | 套餐和价格说明 |
| Rankings | 公开排行 |
| About | 项目信息 |
| Legal | Terms / Privacy |
| Sign in / Sign up | 登录注册 |

### 普通用户

| 分组 | 菜单 |
| --- | --- |
| Workspace | Playground、Chat |
| Console | Overview、Dashboard、API Keys、Usage Logs、Task Logs |
| Account | Wallet、Billing、Voucher、Invoice、Profile |

收口建议：Wallet 定位为充值入口，Billing 定位为账单中心，避免都使用“支付/账单”类命名。

### 企业管理员

企业管理员包括 `tenant_admin` 和 `organization_admin`。菜单由 RBAC 过滤后展示：

| 角色 | 推荐菜单 |
| --- | --- |
| tenant_admin | Console、Channels、Users、Voucher、Finance、Invoice Management、Subscription Management、Logs |
| organization_admin | Console、Users、Subscription Management、Logs |

收口建议：企业管理员不展示平台级 Tenants、Models、System Settings，避免误导为平台管理权限。

### 平台管理员

平台管理员包括 `root`、`finance`、`ops`、`auditor`，其中 `root` 可见全部入口。

| 角色 | 推荐菜单 |
| --- | --- |
| root | Tenants、Organizations、Departments、Distribution Channels、Channels、Models、Users、TopUp、Redemption Codes、Voucher、Finance、Invoice Management、Subscription Management、Logs、Statistics、System Settings |
| finance | TopUp、Redemption Codes、Voucher、Finance、Invoice Management、Subscription Management、Statistics |
| ops | Channels、Logs |
| auditor | Logs、Statistics、Subscription Management |

当前重复或易混入口：

| 问题 | 影响 | 建议 |
| --- | --- | --- |
| Dashboard 与 Statistics 都承载统计语义 | 管理员不易区分用户概览和平台统计 | Dashboard 保留用户工作台，Statistics 仅保留平台统计 |
| Usage Logs 同时在普通 Console 和 Admin Logs 中出现 | 角色切换后入口重复 | 用户侧叫 Usage Logs，管理侧叫 Admin Logs |
| Wallet、TopUp、Billing 存在支付语义重叠 | 用户和财务视角混淆 | Wallet 是用户充值，TopUp 是财务补额，Billing 是账单中心 |

## 角色矩阵

| 角色 | 菜单权限 | 页面权限 | API 权限边界 |
| --- | --- | --- | --- |
| root | 全部 | 全部 | 全局数据 |
| tenant_admin | 租户管理、财务概览、发票、卡券、订阅、日志 | 本租户后台页面 | 仅本租户数据 |
| organization_admin | 用户、订阅、日志 | 本组织后台页面 | 仅本组织或授权范围 |
| finance | 财务、充值、卡券、发票、订阅、统计 | 财务后台页面 | 财务相关全局数据 |
| ops | 渠道、日志 | 运营后台页面 | 运营相关授权数据 |
| auditor | 日志、统计、订阅只读视角 | 审计页面 | 审计范围只读数据 |
| user | 用户中心 | 自己的账单、卡券、发票、订阅、Key | 仅自己的数据 |

审计结论：前端菜单、页面路由和后端 API 已按角色分层，关键财务入口使用 `FINANCE_CONSOLE`、`VOUCHERS`、`INVOICES`、`SUBSCRIPTIONS` 等权限。普通用户入口使用认证态加用户归属过滤。

## 页面矩阵

| 页面 | 当前能力 | 一致性状态 | Alpha 风险 |
| --- | --- | --- | --- |
| Payment / Wallet | 支付订单、充值入口、支付状态 | 与 Billing 有语义重叠 | Warning |
| Billing Portal | Summary、Payments、Usage、Bills、Subscriptions | 使用卡片、表格、分页 | 可用 |
| Voucher | 用户核销、历史；后台批次、列表、核销记录 | 使用卡片、表格、状态标签 | 可用 |
| Invoice | 开票资料、申请、文件；后台审核和开票登记 | 使用卡片、表格、状态标签 | 可用 |
| Finance | KPI、排行、最近活动、支付/卡券/收益统计 | 使用 KPI 卡片和表格 | 可用 |
| Subscription | 套餐和订阅管理 | DataTable 标准化程度较高 | 可用 |
| Dashboard | 用户概览和系统统计入口并存 | 与 Finance / Statistics 风格需继续统一 | Warning |

## Dashboard 收口审计

当前 Billing Portal、Finance Console、Dashboard 均已使用卡片和表格表达核心数据，但实现层面仍有本地重复组件。

建议统一组件：

| 组件 | 当前状态 | 建议 |
| --- | --- | --- |
| KPI Card | Finance 和 Billing 各自实现 | 抽取统一 MetricCard |
| Table | 多页面使用表格但空态和加载态不同 | 统一 TableShell / EmptyRows |
| Status Badge | Voucher、Invoice、Billing 各自映射 | 保留业务映射，统一视觉组件 |
| Pager | 多页面本地实现 | 抽取统一 Pager |
| Loading State | Subscription 较完整，Billing / Voucher / Invoice 偏弱 | 统一 Skeleton / LoadingRows |
| Error State | 多数页面使用 toast，页面内错误态不足 | 增加 PageError / Retry |

## 中文化与 i18n 审计

已检查 `web/default/src` 中 `t(...)` 使用和语言包覆盖情况。当前发现：

| 类型 | 结果 |
| --- | --- |
| 缺失翻译 key | 约 120 个，集中在 Finance、Invoice、Subscription、Admin Console 新页面 |
| zh 未翻译值 | 少量，主要是产品名、协议名、技术指标等可保留词 |
| fr / ja / ru / vi 未翻译值 | 存在较多英文回退，需要后续补齐 |
| 前端中文硬编码 | 少量，集中在 OAuth 错误提示、系统设置支付示例和模型预设名称 |
| 后端中文错误信息 | 多处历史接口存在中文错误文本，本轮未改动 |

Alpha 建议：如果 Alpha 只承诺中文和英文，需优先补齐 en / zh 中 Invoice、Finance、Voucher、Billing、Subscription 的缺失 key；其他语言可在 Beta 前补齐。

## 页面可用性审计

### Critical

| 问题 | 影响 | 建议 |
| --- | --- | --- |
| 多语言严格发布时存在缺失翻译 key | 非英文/中文用户可能看到 key 或英文回退 | Alpha 前至少补齐 en / zh，Beta 前补齐全语言 |

### Warning

| 问题 | 影响 | 建议 |
| --- | --- | --- |
| Dashboard、Statistics、Finance Console 边界不够清晰 | 平台管理员容易误解统计入口 | Dashboard 用户化，Finance 财务化，Statistics 平台运行指标化 |
| Wallet、Billing、TopUp 命名语义接近 | 用户充值、财务补额、账单查询容易混淆 | 菜单文案和页面标题明确职责 |
| Billing / Voucher / Invoice / Finance 的 Loading 和 Error 表现不完全一致 | 异常场景体验不稳定 | 抽取统一加载、错误、空状态组件 |
| 表格分页和筛选栏存在本地重复实现 | 后续维护成本高 | 抽取商业后台通用 FilterBar / Pager |

### Suggestion

| 建议 | 价值 |
| --- | --- |
| 建立商业中心统一布局组件 | Billing、Voucher、Invoice、Subscription 形成统一用户中心体验 |
| 建立 Admin Console 统一列表页模板 | Finance、Voucher、Invoice、Subscription 后台体验一致 |
| 状态标签统一颜色语义 | Pending、Approved、Paid、Rejected、Disabled 等跨模块一致 |
| 增加导出和审计日志入口 | Beta 阶段提升财务运营可审计性 |

## 已完成能力

- 商业闭环：支付、额度、模型调用、计费、账单查看已打通。
- 用户中心：Billing、Voucher、Invoice、Subscription、API Key 已具备基础页面。
- 管理后台：Finance、Voucher、Invoice、Subscription、日志、用户、渠道具备后台入口。
- RBAC：菜单、页面和 API 均已有角色边界。
- Invoice Workflow：仅人工开票流程，无真实税务系统、数电票、税控或第三方服务商对接。

## 未完成能力

- 全语言翻译补齐。
- 统一商业中心组件库。
- 统一后台列表页模板。
- 更完整的页面内错误态和加载态。
- Finance Console 与 Statistics 的信息架构进一步拆分。
- Beta 阶段的导出、审计、对账、结算和真实发票 Provider Integration。

## Alpha 上线阻塞项

| 阻塞项 | 严重级别 | 当前判断 |
| --- | --- | --- |
| 后端测试失败 | Critical | 需以验收命令结果为准 |
| 前端 typecheck / build 失败 | Critical | 需以验收命令结果为准 |
| en / zh 核心页面翻译缺失 | Critical / Warning | 若 Alpha 面向中英文用户则为 Critical；若允许英文回退则为 Warning |
| 普通用户越权访问财务后台 | Critical | 当前前后端均有权限边界，仍建议回归验证 |
| Invoice 外部税务系统接入 | Critical | 当前不应存在，本轮只允许人工开票工作流 |

## Beta 规划

- Invoice Provider Integration：真实数电票、税控、第三方服务商对接单独迭代。
- Settlement：渠道结算、对账和财务导出。
- Commercial UI Kit：统一 KPI、表格、筛选、分页、空态、加载态、错误态。
- Full i18n：补齐 en、zh、fr、ja、ru、vi 全语言。
- Audit Log：财务审核、开票登记、卡券禁用、后台操作审计。
- Admin Analytics：更细粒度租户、渠道、模型、Provider 趋势分析。
- Product Onboarding：面向 Alpha 用户的首次配置和商业闭环引导。

## 验收说明

本轮验收以以下命令为准：

```bash
go test ./model ./service ./controller ./router ./middleware
cd web/default
npm run typecheck
npm run build
```

通过后可判定当前代码具备 Alpha Readiness 的基础发布条件；UI 和 i18n 的深度统一建议进入 Alpha 后续 polish 或 Beta 前置任务。
