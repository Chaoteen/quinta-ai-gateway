# Iteration 12.0 tenant_admin UI / RBAC Repair

## 当前问题

线上 tenant_admin 控制台仍接近个人开发者控制台，菜单包含 Chat、Playground、API Keys、Wallet、Task Logs、Profile 等个人入口，不符合 Quinta AI Gateway 企业租户管理员视角。

已确认的问题：

- tenant_admin 菜单没有独立的企业租户管理员信息架构。
- Chat / Playground / API Keys / Wallet / Task Logs 等入口对 tenant_admin 暴露。
- 隐藏或无权限 URL 缺少统一前端直达保护。
- API 返回 403 时页面依赖各页面自行处理，存在崩溃风险。
- API 返回 401 时仅 toast，没有统一跳转登录。
- API 返回 500 时缺少统一错误页落点。
- 中文菜单中存在 Billing、Wallet、Invoice 等中英文混杂。
- tenant_admin 首页仍以创建 API Key / Playground 为主引导，不像企业后台。

## 设计目标

tenant_admin 控制台调整为企业租户管理员视角，重点服务：

- 用户管理
- 渠道管理
- 订阅管理
- 账单中心
- 卡券管理
- 发票管理
- 使用日志
- 额度与用量
- 企业设置

边界：

- 本轮只修复前端菜单、页面守卫、错误处理、业务菜单文案和 tenant_admin 首页结构。
- 不新增支付、发票、结算、额度核心业务功能。
- 不修改后端计费、支付、发票模型。
- 受仓库保护策略限制，不做受保护项目品牌标识的全局删除或替换。

## tenant_admin 菜单矩阵

| 菜单 | 路由 | 权限来源 | 说明 |
| --- | --- | --- | --- |
| 控制台 | `/dashboard/overview` | authenticated + tenant_admin 菜单分支 | 企业运营概览 |
| 用户管理 | `/users` | `USERS` | 租户用户管理 |
| 渠道管理 | `/channels` | `CHANNELS` | 上游渠道和路由管理 |
| 订阅管理 | `/subscriptions` | `SUBSCRIPTIONS` | 租户订阅管理 |
| 账单中心 | `/billing` | authenticated | 企业余额、账单、用量视图 |
| 卡券管理 | `/admin/vouchers` | `VOUCHERS` | 租户卡券批次、兑换码、核销记录 |
| 发票管理 | `/admin/invoices` | `INVOICES` | 发票资料、申请、审核和人工开票登记 |
| 使用日志 | `/usage-logs/common` | `USAGE_LOGS` | 请求和计费日志 |
| 额度与用量 | `/dashboard/models` | authenticated | 模型调用和用量概览 |
| 企业设置 | `/profile` | authenticated | 当前复用账户资料页作为最小入口 |

tenant_admin 隐藏入口：

| 入口 | 处理 |
| --- | --- |
| `/playground` | 菜单隐藏，直达 403 |
| `/chat/*` | 菜单隐藏，直达 403 |
| `/keys` | 菜单隐藏，直达 403 |
| `/wallet` | 菜单隐藏，直达 403 |
| `/vouchers` | 菜单隐藏，使用 `/admin/vouchers` |
| `/invoices` | 菜单隐藏，使用 `/admin/invoices` |
| `/usage-logs/task` | 菜单隐藏，直达 403 |
| `/usage-logs/drawing` | 菜单隐藏，直达 403 |

## 修复内容

### 菜单收口

- 在 `useSidebarData` 中为 `role_key = tenant_admin` 增加独立菜单树。
- tenant_admin 不再展示 Chat、Playground、API Keys、Wallet、Task Logs 等个人开发者入口。
- root 和普通 user 仍保留原有菜单结构，不受 tenant_admin 分支影响。

### 权限过滤

- 新增 `isPathAllowedForRole`，对 tenant_admin 的隐藏入口做路径级限制。
- 在 `_authenticated` 父路由中统一调用 `requirePathAllowed`。
- tenant_admin 直接访问隐藏 URL 时进入 403 页面。

### 错误处理

- API 返回 401：清理本地登录态并跳转 `/sign-in`，保留当前 redirect。
- API 返回 403：跳转 `/403` 友好无权限页。
- API 返回 500：跳转 `/500` 友好错误页。

### 首页结构

tenant_admin 的 Dashboard Overview 改为企业概览，优先展示：

- 企业账户余额
- 本月用量
- 近 30 天请求
- 活跃订阅
- 用户数量
- 渠道状态
- Token 用量
- 订阅状态
- 最近账单
- 最近使用日志入口

普通 user 仍保留原来的 API Key / Playground 引导。

### 文案

- tenant_admin 菜单使用中文业务命名：控制台、用户管理、渠道管理、订阅管理、账单中心、卡券管理、发票管理、使用日志、额度与用量、企业设置。
- 中文语言包中将 Billing 调整为“账单中心”，Wallet 调整为“余额中心”，Dashboard 调整为“控制台”。
- 补齐 tenant_admin 新首页和菜单所需 en、zh、fr、ja、ru、vi 翻译 key。

## 测试结果

已执行：

```bash
cd web/default
npm run typecheck
npm run build
```

结果：

- `npm run typecheck`：通过。
- `npm run build`：通过。

后端未修改业务模型和接口，仍执行回归：

```bash
go test ./model ./service ./controller ./router ./middleware
```

结果以最终命令输出为准。

## 后续建议

- 为 tenant_admin 增加真正的企业设置页面，替代当前最小复用的 `/profile`。
- 为 tenant_admin 增加 Playwright 角色菜单回归测试：tenant_admin、user、root 各登录一次并校验菜单。
- 将 API 403/500 页面跳转从 `window.location.assign` 升级为路由级错误边界，减少整页刷新。
- 后续部署时关闭 TanStack Router Devtools、React DevTools 提示和 i18next debug 日志，避免线上控制台噪音。
- 线上账号需要确认可用性；本次线上登录 API 对提供账号返回用户名或密码错误/用户禁用，无法完成登录后实测。
