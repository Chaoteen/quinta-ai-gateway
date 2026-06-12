# Quinta AI Gateway 产品路线图

## 1. 项目定位

Quinta AI Gateway 的长期定位是面向企业和平台运营方的 AI 商业化基础设施，逐步从统一网关演进为 MaaS 平台、Agent Marketplace 和 Enterprise AI Operating System。

长期目标：

- AI Gateway：统一接入 OpenAI、Claude、Gemini、Azure、Bedrock 等多类模型和 Provider，提供认证、限流、转发、日志和治理能力。
- MaaS Platform：提供订阅、额度、计量、账单、支付、卡券、发票、收益分成和财务运营能力，支持模型即服务商业化。
- Agent Marketplace：在模型网关和商业闭环之上，支持 Agent 上架、购买、分发、计费和运营。
- Enterprise AI Operating System：面向企业客户提供组织、权限、知识库、审计、分析、成本治理和内部 AI 服务运营能力。

当前 11.2 阶段的核心判断：项目已经具备 Alpha 商业化闭环，下一阶段重点不应继续堆叠模块，而应提升生产支付、结算、审计、国际化和产品体验成熟度。

## 2. 当前能力矩阵

| 模块 | 当前阶段 | 说明 |
| --- | --- | --- |
| 多租户 | Alpha Ready | 已具备 Tenant、Organization、Department、Distribution Channel 归属基础，可支撑租户级隔离和运营范围控制 |
| RBAC | Alpha Ready | 已具备 root、tenant_admin、organization_admin、finance、ops、auditor、user 等角色边界 |
| Subscription | Alpha Ready | 已具备套餐、订阅开通、续费、状态查询和后台管理基础 |
| Quota | Alpha Ready | 已具备额度配置、补额、扣减、运行时校验和用户余额查看基础 |
| Usage Metering | Alpha Ready | 已具备用量采集、请求数、Token 消耗和消费明细查询 |
| Billing Runtime | Alpha Ready | 已具备计费记录和账单生成基础，不建议在 Alpha 前重构计费主链路 |
| Revenue Share | Foundation | 已具备收益分成记录和基础统计，尚未进入完整结算和对账阶段 |
| Payment Gateway Foundation | Foundation | 已具备支付订单、Mock Provider、银行转账人工审核和支付履约，不包含真实微信/支付宝生产接入 |
| Billing Portal | Alpha Ready | 用户可查看余额、支付记录、消费记录、账单和订阅状态 |
| Voucher | Alpha Ready | 已具备批次、兑换码、核销、Quota/Subscription 履约和后台管理 |
| Invoice Workflow | Foundation | 已具备开票资料、发票申请、财务审核、人工开票登记和文件记录，不包含真实税务系统对接 |
| Finance Console | Alpha Ready | 已具备收入、消费、活跃度、排行和最近活动等运营概览 |

阶段定义：

- Foundation：核心模型和主流程已建立，可支撑后续迭代，但仍缺生产级集成或完整运营能力。
- Alpha Ready：可以在受控 Alpha 环境中给真实用户试用，功能闭环成立，仍允许存在体验和运营效率短板。
- Beta Ready：具备更完整的生产集成、审计、国际化、异常处理和运营工具。
- Production Ready：具备生产 SLA、风控、对账、审计、监控、数据导出和长期维护能力。

## 3. Alpha 能力矩阵

### 用户能力

| 能力 | 当前说明 |
| --- | --- |
| 注册与登录 | 已具备用户认证和后台访问基础 |
| API Key 管理 | 用户可管理自己的 API Key |
| 充值与支付记录 | 用户可通过支付订单和钱包入口查看充值相关记录 |
| 额度余额查看 | 用户可在 Billing Portal 查看余额和消费概览 |
| 模型调用与用量查看 | 用户调用模型后可查看 Usage、Token 和请求记录 |
| 账单查看 | 用户可查看 Billing Records |
| 订阅查看 | 用户可查看当前订阅、历史订阅和即将到期订阅 |
| 卡券核销 | 用户可输入兑换码获得 Quota 或 Subscription |
| 发票状态查看 | 用户可维护开票资料、提交申请并查看开票状态和文件记录 |

### 企业能力

| 能力 | 当前说明 |
| --- | --- |
| 多租户归属 | 已支持租户、组织、部门、渠道维度 |
| 企业管理员 | tenant_admin 和 organization_admin 可在授权范围内管理相关资源 |
| 组织范围查看 | 企业管理员可查看组织或租户范围内用户、订阅、日志等数据 |
| 成本和用量追踪 | 通过 Billing、Usage 和 Finance 相关视图支持基础成本观察 |
| 订阅和额度运营 | 支持企业范围内订阅、额度、卡券和发票工作流 |

### 平台能力

| 能力 | 当前说明 |
| --- | --- |
| Provider 网关 | 已具备多 Provider 模型转发基础 |
| 管理后台 | 已具备 Admin Console、Finance Console 和权限控制 |
| 支付履约 | 已具备支付成功后开通 Subscription 或增加 Quota 的基础链路 |
| 卡券运营 | 支持批次生成、核销记录和禁用管理 |
| 财务运营 | 支持收入、消费、租户、渠道、模型、Provider 维度概览 |
| 发票工作流 | 支持人工开票流程闭环 |
| 权限隔离 | root、finance、tenant_admin、organization_admin、ops、auditor、user 权限边界已建立 |

## 4. Alpha 阻塞项

### Critical

| 问题 | 影响 | 建议 |
| --- | --- | --- |
| 验收测试或前端构建失败 | 直接影响 Alpha 发布 | 每次发布前必须通过后端测试、前端 typecheck 和 build |
| 核心财务页面缺失 en / zh 翻译 key | 面向中英文 Alpha 用户时会影响基本可用性 | Alpha 发布前至少补齐 Billing、Voucher、Invoice、Finance、Subscription 的 en / zh 文案 |
| 菜单权限、页面权限、API 权限不一致 | 可能导致越权访问或误展示 | 发布前执行一次 RBAC 回归测试 |

### Warning

| 问题 | 影响 | 建议 |
| --- | --- | --- |
| Payment Gateway 仍为 Foundation | 不支持生产微信/支付宝自动支付 | Alpha 使用 Mock 和人工银行转账；Beta 做生产支付集成 |
| Invoice Workflow 仅支持人工登记 | 不支持真实数电票、税控或第三方开票 | Beta 或后续独立做 Invoice Provider Integration |
| Dashboard、Finance、Statistics 信息架构仍需收口 | 管理员可能不易区分不同统计入口 | Alpha 可用，Beta 前统一信息架构 |
| 商业页面 UI 组件重复 | 维护成本和体验一致性不足 | Beta 前建设 Commercial UI Kit |
| 多语言包未完全补齐 | 非中英文市场体验不足 | Beta 前补齐全语言 |

## 5. Beta 路线图

| 优先级 | 目标 | 说明 |
| --- | --- | --- |
| P0 | Payment Production Integration | 接入真实微信支付、支付宝或 Stripe 等生产支付能力，补齐签名、回调、安全校验、异常重试和支付对账 |
| P0 | Audit Log | 建立后台操作审计，覆盖财务审核、开票登记、卡券禁用、订阅调整、手工补额等关键动作 |
| P1 | Settlement | 建立渠道、代理、分销和平台收益结算能力，支持结算周期、结算状态和对账 |
| P1 | Commercial UI Kit | 统一 KPI Card、Table、FilterBar、Pager、Status Badge、Empty、Loading、Error 等商业后台组件 |
| P1 | Full i18n | 补齐 en、zh、fr、ja、ru、vi 全语言翻译，建立翻译缺失检查流程 |
| P2 | Invoice Provider Integration | 在当前人工开票工作流之后，独立接入真实数电票、税控或第三方开票服务商 |

Beta 阶段目标：从“功能闭环可用”提升到“生产运营可控”，重点是支付生产化、审计可追踪、财务可结算、界面一致和国际化可发布。

## 6. V1.0 路线图

| 方向 | 阶段目标 |
| --- | --- |
| Agent Marketplace | 支持 Agent 上架、定价、订阅、购买、评价、分发和收益分成 |
| Skill Marketplace | 支持 Skill 发布、安装、版本管理、权限控制和商业化 |
| Video Model Commercialization | 支持视频模型套餐、额度、计费、任务日志和成本分析 |
| Enterprise Knowledge Base | 支持企业知识库、权限隔离、检索增强、用量计费和审计 |
| Advanced Analytics | 支持租户、渠道、模型、Provider、用户、成本和收入的多维分析 |

V1.0 阶段目标：在 MaaS 商业底座稳定后，扩展到 Agent、Skill、Video 和企业知识运营，形成完整的企业 AI 操作系统。

## 7. 产品成熟度评估

| 维度 | 评分 | 说明 |
| --- | --- | --- |
| 技术成熟度 | 7 / 10 | 网关、多租户、计费、额度、账单、卡券、发票和财务基础完整；仍需加强生产支付、审计、监控和异常恢复 |
| 商业成熟度 | 7 / 10 | 已形成充值、额度、调用、计费、账单、卡券、发票的 Alpha 闭环；结算和生产支付仍是 Beta 关键点 |
| 运营成熟度 | 6 / 10 | Finance Console 和 Admin Console 已可用；仍需审计日志、导出、对账、运营报表和统一后台体验 |
| 国际化成熟度 | 5 / 10 | 多语言框架已具备，但新商业页面翻译缺口较多，尚未达到多区域发布标准 |

综合判断：当前适合进入受控 Alpha 发布；Beta 前应优先补齐生产支付、审计、结算、UI 统一和完整 i18n。

## 8. 建议下一阶段开发顺序

11.2 完成后，建议路线如下：

### 12.0 Payment Production Integration

目标：将 Payment Gateway Foundation 推进到 Beta Ready。

范围：

- 真实支付 Provider 接入。
- 回调验签和幂等重试。
- 支付异常处理。
- 支付对账基础。
- 支付安全审计。

### 12.1 Audit Log Foundation

目标：建立平台关键操作的审计底座。

范围：

- 财务审核审计。
- 卡券管理审计。
- 开票登记审计。
- 订阅和额度变更审计。
- 后台管理员操作审计。

### 12.2 Settlement Foundation

目标：补齐 Revenue Share 后的结算闭环。

范围：

- 渠道结算单。
- 结算周期。
- 结算状态。
- 对账和导出基础。
- 平台、总代理、分销收益汇总。

### 12.3 Commercial UI Kit

目标：统一商业后台和用户中心体验。

范围：

- KPI Card。
- Table / FilterBar / Pager。
- Status Badge。
- Empty / Loading / Error。
- Billing、Voucher、Invoice、Finance、Subscription 页面统一。

### 12.4 Full i18n Readiness

目标：达到 Beta 多语言发布标准。

范围：

- 补齐 en、zh、fr、ja、ru、vi。
- 建立缺失 key 检查。
- 清理硬编码文案。
- 统一商业术语表。

### 12.5 Invoice Provider Integration

目标：在人工开票工作流稳定后，独立建设真实开票 Provider 集成。

范围：

- 数电票、税控或第三方服务商接入预留。
- Provider 抽象。
- 外部回调。
- 失败重试。
- 文件回写。

注意：Invoice Provider Integration 必须独立迭代，不应混入当前 Invoice Workflow Foundation。

### 13.0 Agent Marketplace Foundation

目标：开始从 MaaS Platform 迈向 Agent Marketplace。

范围：

- Agent 商品模型。
- Agent 上架和审核。
- Agent 购买或订阅。
- Agent 使用计费。
- Agent 收益分成。

## 结论

Quinta AI Gateway 当前已经具备 Alpha 商业化闭环，可以进入受控 Alpha 发布准备阶段。下一阶段应优先提升生产支付、审计、结算、UI 一致性和完整国际化，而不是继续新增分散业务模块。
