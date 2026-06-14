# Iteration 12.1 Landing Docs Brand Fix

## 问题来源

线上首页仍显示旧品牌，主要来自两个入口：

- Landing Header 左上角品牌读取 `/api/status` 返回的 `system_name`，当后端或浏览器持久化缓存仍为 `New API` / `NewAPI` 时，前端默认值不会生效。
- 顶部导航“文档”读取 `/api/status` 返回的 `docs_link`，当配置值仍为 `https://docs.newapi.pro/zh` 时会继续跳转外部旧文档站。

本轮只处理用户可见品牌、首页、导航、文档入口和用户文档，不修改 Go module、仓库名、import path、OpenAPI 技术 schema、`New-Api-User` 请求头或数据库模型。

## New API 残留位置

| 分类 | 位置 | 处理结果 |
| --- | --- | --- |
| 用户可见 | `web/default/src/components/layout/components/public-header.tsx` 通过 `useSystemConfig()` 展示站点名 | `useSystemConfig()` 返回值统一归一化，旧 `system_name` 展示为 `Quinta AI Gateway` |
| 用户可见 | `web/default/src/components/layout/components/system-brand.tsx` 通过 `/api/status` 展示系统品牌 | 增加旧品牌兼容归一化 |
| 用户可见 | `web/default/src/main.tsx` 通过 `/api/status` 或缓存设置页面 title | 增加旧品牌兼容归一化 |
| 用户可见 | `web/default/src/features/home/components/sections/hero.tsx` 首页副标题 | 改为企业级 AI Gateway 与 MaaS 平台描述 |
| 用户可见 | `web/default/src/i18n/locales/*.json` 新增首页和文档页翻译 | 补齐 en / zh / fr / ja / ru / vi |

## 文档跳转残留位置

| 分类 | 位置 | 处理结果 |
| --- | --- | --- |
| 用户可见 | `web/default/src/hooks/use-top-nav-links.ts` 使用 `status.docs_link` 生成顶部“文档”链接 | 对 `docs.newapi.pro` 做兼容归一化，统一跳转 `/docs` |
| 用户可见 | 缺少 Quinta AI Gateway 自有文档入口 | 新增 `/docs` 前端路由和静态文档目录 |

## 修复文件清单

- `web/default/src/lib/branding.ts`
- `web/default/src/hooks/use-system-config.ts`
- `web/default/src/hooks/use-top-nav-links.ts`
- `web/default/src/main.tsx`
- `web/default/src/components/layout/components/system-brand.tsx`
- `web/default/src/features/home/components/sections/hero.tsx`
- `web/default/src/routes/docs/index.tsx`
- `web/default/src/routeTree.gen.ts`
- `web/default/src/i18n/locales/en.json`
- `web/default/src/i18n/locales/zh.json`
- `web/default/src/i18n/locales/fr.json`
- `web/default/src/i18n/locales/ja.json`
- `web/default/src/i18n/locales/ru.json`
- `web/default/src/i18n/locales/vi.json`

## 新增文档清单

- `docs/user-guide/quick_start.md`
- `docs/user-guide/api_access.md`
- `docs/user-guide/account_and_api_key.md`
- `docs/user-guide/quota_and_billing.md`
- `docs/user-guide/subscription.md`
- `docs/user-guide/voucher.md`
- `docs/user-guide/invoice.md`
- `docs/user-guide/admin_console.md`
- `docs/user-guide/faq.md`

## `/docs` 页面内容

新增 Quinta AI Gateway 文档中心，包含以下静态入口：

- 快速开始
- API 接入说明
- 账号与 API Key
- 额度与计费
- 订阅管理
- 卡券管理
- 发票管理
- 管理员控制台
- 常见问题

当前页面展示标题、简短说明和对应 Markdown 路径，不跳转外部 New API 文档站。

## 是否仍存在技术路径残留

本轮审计后，用户可见 Landing Header、首页、页面 title、顶部文档入口和新增用户文档不再依赖旧品牌或外部 `docs.newapi.pro` 链接。

仍保留的技术路径或审计记录如下：

- `web/default/src/lib/branding.ts` 保留 `docs.newapi.pro` 字符串，仅用于把旧配置兼容归一化为 `/docs`。
- `web/default/src/features/chat/lib/send-to-fluent.ts` 中的 `fluent-new-api-container`、`id: 'new-api'` 是 chat/fluent 集成协议标识，不作为用户可见品牌处理。
- `web/default/src/features/chat/lib/chat-links.ts` 中的 `id: 'new-api'`、`platform: 'new-api'` 是聊天客户端协议标识，不作为用户可见品牌处理。
- `web/default/src/i18n/locales/*` 中 `Unified API Gateway for` / `统一 API 网关，服务于` 是首页主标题表达；本轮需求明确允许首页主标题保留“统一 API 网关，服务于所有 AI 模型”。
- `docs/iteration_12_1/brand_cleanup_audit.md` 和本文件中的旧品牌词仅为审计说明。

## 测试结果

- `cd web/default && npm run i18n:sync`：通过。
- `cd web/default && npm run typecheck`：首次因新增 `/docs` 路由尚未生成类型失败；构建刷新 `routeTree.gen.ts` 并清理未使用 import 后，复跑通过。
- `cd web/default && npm run build`：通过。
- `git diff --check`：通过。

## 后续建议

- 后台系统设置中的 `system_name` 和 `docs_link` 建议在部署配置层同步改为 `Quinta AI Gateway` 与 `/docs`，前端兼容层只负责防止旧配置继续外显。
- 后续可将 `/docs` 目录页升级为内部 Markdown 渲染或分章节路由，直接承载 `docs/user-guide/*.md`。
