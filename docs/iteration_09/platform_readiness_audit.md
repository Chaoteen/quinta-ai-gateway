# Iteration 9.1 Platform Readiness Audit

## 1. 当前完成度评估

结论：Quinta AI Gateway 当前具备“受控 Alpha 技术测试”能力，但不具备“开放式多租户自助 Alpha”能力。

建议 Alpha 范围限定为：

- 单平台 root / 少量 tenant_admin 内测；
- 后端 API、Relay、渠道、模型目录、用户、订阅、充值、日志等核心链路验证；
- 不开放租户自助创建组织、部门、分销渠道；
- 不开放真实生产财务结算、发票、账本、钱包流水等能力；
- 不把 finance / ops / auditor / organization_admin 作为完整前端角色体验上线。

总体完成度：

| 维度 | 完成度 | 评估 |
| --- | --- | --- |
| Multi-Tenant Readiness | 中 | 模型层和部分后端 scope 已存在，但租户/组织/部门/分销渠道缺少完整管理 API 与 UI |
| RBAC Readiness | 中 | 后端 `role_key` 已生效于部分路由，前端仍主要依赖 legacy 数值角色 |
| Admin Console Readiness | 中低 | 用户、渠道、模型、订阅、充值、兑换码已有页面；租户/组织/部门/角色管理缺失 |
| Billing Readiness | 中低 | TopUp、Redemption、Subscription 基础可用；Wallet/Ledger/Invoice 缺失 |
| Product Marketplace Readiness | 低 | 模型定价/目录存在；Token/Agent/Skill 产品市场未形成 |
| Deployment Readiness | 中 | Docker / Compose / env / setup / root 初始化存在；HTTPS/Nginx 主要依赖外部部署 |
| Observability | 中 | 日志、用量统计、健康检查、性能指标存在；审计日志和业务指标体系不完整 |

Alpha 可行性判断：

- 技术 Alpha：可以。
- 多租户运营 Alpha：谨慎，仅限人工配置后的少量租户。
- 商业化 Alpha：不建议，Billing Foundation 仍缺账本、发票和钱包模型。
- 公网生产 Alpha：不建议，除非前置 Nginx/HTTPS、默认密码替换、回调安全和权限边界完成专项检查。

## 2. 后端能力完成度

### A. Multi-Tenant Readiness

| 对象 | Model | Controller | Router | UI 入口 | 结论 |
| --- | --- | --- | --- | --- | --- |
| Tenant | 有 `model.Tenant`，迁移包含 | 未发现独立租户 CRUD controller | 未发现 `/api/tenant` 管理路由 | 无租户管理页面 | 仅模型就绪，管理面缺失 |
| Organization | 有 `model.Organization`，含 `tenant_id` 和 `distribution_channel_id` | 未发现独立组织 CRUD controller | 未发现 `/api/organization` 管理路由 | 无组织管理页面 | 仅模型和 scope 基础就绪 |
| Department | 有 `model.Department`，含 `tenant_id`、`organization_id`、`parent_id` | 未发现独立部门 CRUD controller | 未发现 `/api/department` 管理路由 | 无部门管理页面 | 仅模型就绪，Department Scope 未产品化 |
| DistributionChannel | 有 `model.DistributionChannel`，含 owner 与佣金字段 | 未发现独立分销渠道 CRUD controller | 未发现 `/api/distribution-channel` 管理路由 | 无分销渠道管理页面 | 仅模型就绪，分销业务未形成 |
| User | 有完整 `model.User`，含四类 ownership 和 `role_key` | 用户列表、搜索、详情、创建、更新、删除、绑定、额度调整等存在 | `/api/user/*` 存在，部分只读已按 RoleAuth 下放 | 有 `/users` 页面和菜单 | 用户管理可 Alpha，但细粒度角色体验不足 |

后端多租户基础已经覆盖到 User、Channel、Log、TopUp、Redemption、Subscription 等对象的一部分读取链路。Relay 侧已有 tenant scope 严格处理，避免缺失 tenant context fallback。Organization 只读迁移已覆盖部分用户、日志、订阅、统计读取，但写操作和组织/部门实体管理尚未完成。

主要缺口：

- Tenant / Organization / Department / DistributionChannel 作为平台实体没有完整管理路由。
- Department Scope 仍未成为稳定授权边界。
- DistributionChannel 目前更像 ownership 字段和模型预留，不是可运营的分销后台。
- 多租户对象仍存在历史 `tenant_id=1` fallback 风险，Billing 审计已单独指出。

### B. RBAC Readiness

| role_key | 后端生效 | 页面可见 | 菜单可见 | 结论 |
| --- | --- | --- | --- | --- |
| root | 是；`RootAuth()` 与 `RoleAuth()` 中 root bypass 生效 | 是；数值角色 `ROLE.SUPER_ADMIN` 可访问系统设置等 | 是；Admin 菜单可见 | 平台 root 可 Alpha |
| admin | 部分；legacy admin 被映射为 `tenant_admin` | 是；前端把 `role >= ADMIN` 视作后台用户 | 是；Admin 菜单可见 | 兼容角色可用，但语义不清 |
| tenant_admin | 是；用户、渠道只读/执行、订阅/计费只读、日志等路由已有覆盖 | 不完整；前端未按 `role_key` 精准控制页面 | 不完整；菜单仍按数值 admin 判断 | 后端可测，前端体验未就绪 |
| organization_admin | 是；用户只读、组织运营只读、订阅只读、统计只读已覆盖部分路由 | 不完整 | 不完整 | 后端只读 Alpha 可测，管理体验未就绪 |
| finance | 是；充值、兑换码、订阅计划只读、余额刷新等部分路由 | 不完整 | 不完整 | API 可测，页面权限不准 |
| ops | 是；渠道只读、渠道单项执行、运营日志等部分路由 | 不完整 | 不完整 | API 可测，页面权限不准 |
| auditor | 是；多类只读路由已覆盖 | 不完整 | 不完整 | API 可测，页面权限不准 |
| user | 是；普通用户鉴权、钱包、API Key、日志、订阅自助能力存在 | 是 | 是 | 普通用户 Alpha 可测 |

后端 RBAC 已进入细粒度阶段，`common/rbac.go` 定义了 `root / tenant_admin / organization_admin / finance / ops / auditor / user`，`router/api-router.go` 已在多处使用 `RoleAuth()` 组合授权。

前端仍是主要短板：

- `web/default/src/lib/roles.ts` 仍以 `GUEST / USER / ADMIN / SUPER_ADMIN` 数值角色为主。
- Admin 菜单通过 `role >= ROLE.ADMIN` 控制可见性，不识别 `role_key`。
- 用户创建/编辑表单仍主要提供 User/Admin，不能完整维护 tenant_admin、organization_admin、finance、ops、auditor。
- 因此前端可能出现“菜单可见但 API 拒绝”或“API 允许但菜单不可达”的不一致。

## 3. 前端能力完成度

### C. Admin Console Readiness

| 模块 | 页面 | API | 菜单入口 | 结论 |
| --- | --- | --- | --- | --- |
| 租户管理 | 无 | 无独立租户管理 API | 无 | Alpha 阻塞项，若要多租户自助必须补齐 |
| 组织管理 | 无 | 无独立组织管理 API | 无 | 组织实体不可运营 |
| 部门管理 | 无 | 无独立部门管理 API | 无 | Department Scope 无前端入口 |
| 用户管理 | 有 `/users` | 有 `/api/user/*` | 有 Users | 可测，但 role_key 和 ownership 编辑不完整 |
| 角色管理 | 无独立页面 | 无独立角色管理 API | 无 | 只能依附用户管理，RBAC 不可运营 |
| 渠道管理 | 有 `/channels` | 有 `/api/channel/*` | 有 Channels | 可测，部分高危操作仍应 root/admin |
| 模型管理 | 有 `/models/metadata` 与系统模型设置 | 有 `/api/models/*`、`/api/vendors/*` | 有 Models | 可测，写操作保持 root |
| 订阅管理 | 有 `/subscriptions` | 有 `/api/subscription/admin/*` | 有 Subscription Management | 可测，但订阅 mutation 不宜下放 |
| 充值管理 | 无独立后台页面；有旧 `/console/topup` 跳转和钱包历史 | 有 `/api/user/topup` 管理查询与 complete | 无明确 Admin 菜单 | 财务后台不完整 |
| 兑换码管理 | 有 `/redemption-codes` | 有 `/api/redemption/*` | 有 Redemption Codes | 可测，但核销 ownership 风险需后续修复 |
| 系统配置 | 有 `/system-settings/*` | 有 `/api/option/*`、custom OAuth、performance 等 | 有 System Settings | root 可测，不适合 tenant admin |

前端已经能支撑 root/admin 做技术 Alpha，但还不能支撑多角色运营：

- Admin 导航没有按 finance / ops / auditor / organization_admin 分组。
- 系统设置页面 root-only，但菜单过滤只靠数值角色，低权限 admin 可能看到不可用入口。
- 租户、组织、部门、分销渠道缺少页面，导致多租户只能通过后端字段或人工数据库配置落地。

## 4. 上线阻塞项

以下事项阻塞“开放式 Alpha”：

| 阻塞项 | 严重度 | 原因 |
| --- | --- | --- |
| 前端未按 `role_key` 控制菜单和页面 | P0 | 多角色用户会看到错误菜单或无法进入已授权功能 |
| Tenant / Organization / Department / DistributionChannel 管理面缺失 | P0 | 无法自助创建和维护多租户结构 |
| Billing 缺 Wallet / Ledger / Invoice | P0 | 无法支撑真实商业结算、对账、退款和开票 |
| 兑换码核销 ownership 校验不足 | P0 | 可能产生跨 tenant / organization 核销风险 |
| Billing 创建链路存在 `tenant_id=1` fallback 风险 | P0 | 账务对象可能被写入错误 tenant |
| 角色管理缺失 | P1 | 无法运营 finance / ops / auditor / organization_admin |
| 财务后台不完整 | P1 | TopUp、SubscriptionOrder、UserSubscription 缺统一管理视图和审计链路 |
| 审计日志不完整 | P1 | 高危操作缺 actor、target、ownership、before/after 记录 |
| HTTPS / Nginx 未内置生产方案 | P1 | 公网部署需要外部反代、TLS、回调 URL 和安全头配置 |
| Product Marketplace 缺 Token/Agent/Skill 产品 | P2 | 不影响 API Gateway Alpha，但影响产品商业化体验 |

## 5. 可立即上线测试功能

在受控 Alpha 环境可以测试：

- 用户注册、登录、2FA、Passkey、OAuth 登录和绑定。
- 普通用户 API Key 创建、查询、删除和密钥查看。
- Relay 请求、渠道选择、tenant scoped channel cache、用量扣费。
- 渠道管理：root/admin 创建、更新、删除；tenant_admin/ops 单渠道测试、拉取模型、Ollama version、余额刷新。
- 模型目录与公开定价页：模型列表、搜索、详情、pricing 页面。
- 用户管理只读：tenant_admin 读本 tenant，organization_admin 读本 organization。
- 订阅计划读取、用户订阅自助购买、订阅偏好。
- 管理端订阅只读：tenant_admin/finance/auditor/root，organization_admin 只读本组织用户订阅。
- 充值自助、充值历史、兑换码管理与用户兑换，但兑换码跨 ownership 风险需在内测中严格限制。
- 用量日志、任务日志、统计卡片和 `/api/log/stat`。
- Docker Compose 部署、数据库迁移、root 初始化、setup wizard。

## 6. 必须补齐功能

### 多租户与组织结构

- Tenant CRUD API 与页面。
- Organization CRUD API 与页面。
- Department CRUD API 与页面。
- DistributionChannel CRUD API 与页面。
- 用户与组织/部门/分销渠道的可视化绑定。
- Department Scope 的查询、写入、迁移测试。

### RBAC 与 Admin Console

- 前端读取并使用 `role_key`。
- 菜单按 root / tenant_admin / organization_admin / finance / ops / auditor / user 精准过滤。
- 页面级 guard 与操作按钮级 guard 同步后端授权。
- 角色管理页面或至少用户编辑中的完整 role_key 选择。
- 针对 finance、ops、auditor 的只读页面体验。

### Billing

- Wallet 独立模型，不再只用 `User.Quota` 表达余额。
- Ledger 账本模型，记录所有余额变动、订阅扣减、退款、补单和人工调整。
- Invoice 发票模型和开票状态。
- Order 统一模型，或明确 SubscriptionOrder / TopUp Order 的边界。
- TopUp / Redemption / Subscription mutation 的审计日志。
- 兑换码核销 ownership fail-closed。
- 账务创建链路禁止 `tenant_id=1` fallback。

### Product Marketplace

- Product Catalog 模型和 API。
- Token 产品展示。
- Agent 产品展示。
- Skill 产品展示。
- 产品与订阅计划、额度、价格、可见范围的绑定。

### 部署与安全

- 生产部署文档明确 Nginx、HTTPS、回调域名、SESSION_SECRET、默认密码替换。
- Docker Compose 默认密码不得用于公网。
- 支付 webhook 的生产验签与重放保护专项检查。
- Root 初始化后的强制改密或 setup 流程约束。

## 7. Alpha 上线建议

建议 Alpha 分两档：

### Alpha-Internal

适合立即启动，条件：

- 仅 root 和少量 tenant_admin 参与；
- 数据库使用独立测试库；
- 支付使用测试模式或小额白名单；
- 不开放租户/组织/部门自助管理；
- 不承诺账务对账和发票；
- 所有高危操作保留 root；
- 通过人工记录补充审计日志。

可开放功能：

- Relay 主流程；
- 用户与 API Key；
- 渠道配置和单渠道测试；
- 模型目录和 pricing；
- 用量日志；
- 钱包充值测试；
- 订阅购买测试；
- 兑换码内部测试。

### Alpha-External-Controlled

需要先完成：

- 前端 `role_key` 菜单和页面 guard；
- 账务 fail-closed 修复；
- 兑换码核销 ownership 修复；
- TopUp / Subscription / Redemption 高危 mutation 审计日志；
- Nginx/HTTPS/SESSION_SECRET/默认密码生产部署检查；
- 租户、组织、部门至少具备后台可维护入口，或明确由运维人工配置。

不建议 Alpha 期间开放：

- organization_admin 写操作；
- finance mutation；
- auditor 任何 mutation；
- tenant_admin 修改全局模型、供应商、系统配置、支付配置；
- 硬删除账务对象；
- OAuth provider 配置给非 root；
- Deployment 外部资源管理给非 root。

## 8. Beta 上线建议

Beta 前建议达到以下状态：

- 多租户实体管理完整：Tenant / Organization / Department / DistributionChannel 均有 API、页面、菜单和测试。
- RBAC 完整：所有页面、菜单、按钮、API 都按 `role_key` 对齐。
- Billing Foundation 完整：Wallet、Ledger、Invoice、Order 或等价模型完成。
- 财务可审计：充值、兑换、订阅、退款、人工调整都有审计日志和账本记录。
- 组织级运营可用：organization_admin 可管理本组织用户、查看本组织订阅和统计，但不能触达渠道密钥、支付配置和全局模型配置。
- 运维角色可用：ops 可执行本 tenant 低风险渠道操作，不能查看 credential。
- 财务角色可用：finance 可看账务报表和余额，不直接执行未审计 mutation。
- 审计角色可用：auditor 可读日志、账务、配置快照，不可写。
- Product Marketplace 至少具备模型产品和订阅计划展示；Token/Agent/Skill 产品可以先只读展示。
- 生产部署文档和健康检查成熟，支持 Docker Compose + Nginx + HTTPS 标准路径。

## 9. Iteration 9.2 建议

建议 Iteration 9.2 聚焦 “Frontend RBAC & Tenant Admin Console Foundation”：

1. 前端 auth store 增加 `role_key`、`tenant_id`、`organization_id`、`department_id`、`distribution_channel_id` 使用链路。
2. 重构菜单过滤：Admin 菜单按 root / tenant_admin / organization_admin / finance / ops / auditor 精准显示。
3. 页面 guard 对齐后端 RoleAuth：Channels、Users、Models、Subscriptions、Redemption、System Settings 分别设置角色矩阵。
4. 用户管理页面支持显示 ownership 字段和 `role_key`。
5. 新增只读 Tenant / Organization / Department / DistributionChannel 管理页面雏形，先不开放写操作。
6. 财务页面拆分：TopUp 列表、Redemption 列表、Subscription 列表、Log stat 只读入口按 finance/auditor 可见。
7. 补充前端 e2e 或路由 guard 测试，证明低权限角色不会看到 root-only 菜单。
8. 在 9.2 完成后，再进入 9.3 的 Tenant / Organization / Department CRUD 与 Billing fail-closed 修复。

