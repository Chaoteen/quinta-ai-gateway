# Iteration 12.1 Full Frontend E2E Audit

## 测试范围

本轮基于当前 `main` 分支未提交改动进行前端巡检，范围覆盖：

- 未登录公开页面：`/`、`/pricing`、`/docs`、`/docs/*`、`/about`、`/sign-in`、`/sign-up`、`/forgot-password`、`/privacy-policy`、`/user-agreement`
- 注册流程：`/sign-up` 页面结构、表单字段、公开注册关闭场景的风险点
- 登录流程：`/sign-in` 页面结构、失败提示、测试账号登录入口
- 登录后 tenant_admin 菜单：控制台、用户管理、渠道管理、订阅管理、账单中心、卡券管理、发票管理、使用日志、额度与用量、用量分析、支付中心、收益分成、企业设置
- Devtools：`@tanstack/react-query-devtools`、`@tanstack/react-router-devtools`
- 品牌残留：`New API`、`NewAPI`、`docs.newapi.pro`、`One API`、`one-api`

## 测试账号

- UID：`QTadmin001`
- Password：`QTTesting001`

当前本地默认后端 `http://127.0.0.1:3000/api/status` 不可达，无法完成真实登录、role_key 获取和登录后网络请求巡检。本轮已完成可执行的前端构建、路由、菜单、静态品牌和本地公开路由 HTTP 巡检；登录后页面以路由文件、菜单配置和 RBAC 静态覆盖为准。

## 页面巡检表

| 页面 | 巡检方式 | 结果 | 备注 |
| --- | --- | --- | --- |
| `/` | 本地 dev server HTTP | 200 | HTML title 为 `Quinta AI Gateway` |
| `/pricing` | 本地 dev server HTTP | 200 | HTML title 为 `Quinta AI Gateway` |
| `/docs` | 本地 dev server HTTP + 代码检查 | 200 | 卡片已改为内部可点击链接 |
| `/docs/quick-start` | 本地 dev server HTTP + 路由检查 | 200 | 新增 `/docs/$slug` 详情页 |
| `/docs/api-access` | 本地 dev server HTTP + 路由检查 | 200 | 新增 `/docs/$slug` 详情页 |
| `/about` | 本地 dev server HTTP | 200 | HTML title 为 `Quinta AI Gateway` |
| `/sign-in` | 本地 dev server HTTP + 代码检查 | 200 | AuthLayout 已恢复公共 Header |
| `/sign-up` | 本地 dev server HTTP + 代码检查 | 200 | AuthLayout 已恢复公共 Header |
| `/forgot-password` | 本地 dev server HTTP + 代码检查 | 200 | AuthLayout 已恢复公共 Header |
| `/privacy-policy` | 本地 dev server HTTP | 200 | 使用 PublicLayout |
| `/user-agreement` | 本地 dev server HTTP | 200 | 使用 PublicLayout |

## 登录后菜单巡检表

| 菜单 | URL | 静态路由 | 权限/菜单结果 |
| --- | --- | --- | --- |
| 控制台 | `/dashboard/overview` | 存在 `_authenticated/dashboard/$section` | tenant_admin 菜单可见 |
| 用户管理 | `/users` | 存在 `_authenticated/users/index.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 渠道管理 | `/channels` | 存在 `_authenticated/channels/index.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 订阅管理 | `/subscriptions` | 存在 `_authenticated/subscriptions/index.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 账单中心 | `/billing-dashboard` | 存在 `_authenticated/billing-dashboard/index.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 卡券管理 | `/admin/vouchers` | 存在 `_authenticated/admin/vouchers/index.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 发票管理 | `/admin/invoices` | 存在 `_authenticated/admin/invoices/index.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 使用日志 | `/usage-logs/common` | 存在 `_authenticated/usage-logs/$section.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 额度与用量 | `/quota-dashboard` | 存在 `_authenticated/quota-dashboard/index.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 用量分析 | `/usage-analytics` | 存在 `_authenticated/usage-analytics/index.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 支付中心 | `/payment-center` | 存在 `_authenticated/payment-center/index.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 收益分成 | `/revenue-share` | 存在 `_authenticated/revenue-share/index.tsx` | tenant_admin 菜单可见，RBAC 允许 |
| 企业设置 | `/profile` | 存在 `_authenticated/profile/index.tsx` | tenant_admin 菜单可见 |

真实登录后的 403、404、白屏、console error、network 4xx/5xx 需要在可用后端和浏览器环境下复测。

## 发现问题

### P0

| 问题 | 状态 | 处理 |
| --- | --- | --- |
| TanStack React Query Devtools / Router Devtools 在开发模式默认显示，容易被带到线上环境配置 | 已修复 | `web/default/src/routes/__root.tsx` 改为默认关闭，只有 `import.meta.env.DEV && VITE_ENABLE_DEVTOOLS === 'true'` 时显示 |
| 登录、注册、找回密码页顶部公共菜单消失 | 已修复 | `AuthLayout` 改为挂载 `PublicHeader` |
| 登录/注册页仍可能显示后端返回的旧品牌 | 已修复 | 继续复用 `useSystemConfig()` 的 `systemName` 归一化能力，`New API` / `NewAPI` 展示为 `Quinta AI Gateway` |
| 文档入口仍可能跳转 `docs.newapi.pro` | 已修复 | 旧 docs link 在 `normalizeDocsLink()` 中归一化到 `/docs` |

### P1

| 问题 | 状态 | 处理 |
| --- | --- | --- |
| `/docs` 卡片不可点击 | 已修复 | 文档卡片改为 TanStack Router 内部 `Link` |
| `/docs/*` 无详情路由 | 已修复 | 新增 `/docs/$slug` 文档详情页 |
| `/docs` 中文环境中展示英文 Markdown 路径和英文 fallback | 已修复 | 不再展示 Markdown 路径；修正 i18n 写入位置到 `translation` 下 |
| 公开页菜单不一致 | 已修复主要入口 | Auth 页面恢复公共 Header，其他公开页继续使用 PublicLayout |

### P2

| 问题 | 状态 | 处理 |
| --- | --- | --- |
| 文档详情页为静态内容，未直接渲染 Markdown 文件 | 未修复 | 当前满足最小可用；后续可接入内部 Markdown 渲染 |
| 非中文 locale 的新增长文档内容质量需要人工校对 | 未修复 | 本轮优先保证中文用户可见体验和 key 完整性 |
| 真实登录后的 console/network 巡检缺少浏览器自动化 | 未修复 | 当前项目未安装 Playwright/Cypress，且本地后端不可达 |

## 已修复内容

- Devtools 默认关闭，新增显式开关语义：`VITE_ENABLE_DEVTOOLS=true`。
- 登录、注册、找回密码页恢复顶部公共 Header 和导航。
- `/docs` 卡片改为可点击内部路由，不再暴露英文 Markdown 路径。
- 新增 `/docs/$slug` 详情页，支持快速开始、API 接入、账号与 API Key、额度与计费、订阅、卡券、发票、管理员控制台、常见问题。
- 修复新增文档 i18n key 被写到 locale JSON 顶层的问题。
- 刷新 TanStack Router 生成文件，新增 `/docs/$slug` 类型。
- 继续保留旧 `system_name` 和 `docs_link` 的前端兼容归一化。

## 未修复内容

- 未引入真实支付 SDK，未改动后端核心业务和数据库模型。
- 未进行真实账号登录后的浏览器交互巡检，因为当前后端不可达，项目也没有 Playwright/Cypress 依赖。
- 文档详情页暂未直接读取 `docs/user-guide/*.md`，当前使用前端静态文案承载。

## 品牌与 Devtools 审计

用户可见品牌残留搜索结果：

- `web/default/src/lib/branding.ts` 保留 `docs.newapi.pro`，仅用于把旧配置归一化为 `/docs`。
- `web/default/src/features/chat/lib/*` 保留 `new-api`，属于 chat/fluent 协议 id，不作为用户可见品牌。
- `docs/iteration_12_1/*.md` 中旧品牌词仅为审计说明。

Devtools 搜索结果：

- `web/default/src/routes/__root.tsx` 仍保留 Devtools import 和组件，但默认不渲染。
- 生产构建后的 `web/default/dist` 未搜索到 `ReactQueryDevtools`、`TanStackRouterDevtools`、`React Query Devtools`、`TanStack Router Devtools` 文案。

## 测试结果

- `cd web/default && npm run i18n:sync`：通过。
- `cd web/default && npm run typecheck`：首次因新增 `/docs/$slug` 尚未进入生成路由类型失败；执行 build 刷新 `routeTree.gen.ts` 后复跑通过。
- `cd web/default && npm run build`：通过。
- `git diff --check`：通过。
- 本地公开路由 HTTP 巡检：`/`、`/pricing`、`/docs`、`/docs/quick-start`、`/docs/api-access`、`/about`、`/sign-in`、`/sign-up`、`/forgot-password`、`/privacy-policy`、`/user-agreement` 均返回 200，HTML title 为 `Quinta AI Gateway`。

## 后续建议

- 在可用后端环境启动后，用 `QTadmin001` 完成真实登录，记录 role_key，并对 tenant_admin 所有菜单执行浏览器级 console/network 巡检。
- 引入或恢复 Playwright E2E 测试套件，覆盖公开页、登录失败、登录成功、tenant_admin 菜单、403 页面和 Devtools 不显示断言。
- 将 `/docs/$slug` 后续升级为内部 Markdown 渲染，直接消费 `docs/user-guide/*.md`。
