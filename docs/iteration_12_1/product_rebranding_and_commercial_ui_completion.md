# Iteration 12.1 Product Rebranding & Commercial UI Completion

## 本轮目标

完成 Quinta AI Gateway 默认前端的产品品牌收尾、tenant_admin 企业控制台菜单补齐、商业化页面最小可用内容、i18n 修复、错误处理确认和提交前测试。

## Legacy Brand 痕迹清理范围

本轮清理用户可见产品品牌位置：

- 默认 HTML title 与 meta title。
- 默认系统名、Logo SVG title、Header/Sidebar/SystemBrand fallback。
- Footer 品牌文案与产品描述。
- About 空状态中的项目展示文案。
- 站点设置、邮件设置、Passkey 设置中的默认占位文案。
- 渠道配置提示里把产品名描述改为通用 gateway / compatible relay upstreams。
- locale 文件中未使用的旧产品品牌 key。

统一产品描述为 `Enterprise AI Gateway & MaaS Platform`。

## 保留不改的技术项

以下内容按项目约束保留：

- Go module 路径：`github.com/Chaoteen/quinta-ai-gateway`。
- GitHub 仓库/更新检查路径中的 legacy 技术来源标识。
- Fluent Chat 集成中的 legacy 协议/平台 id。
- OpenAI Chat、WeChat、Chat Completions 等第三方或协议术语。
- 后端业务能力、数据库模型和支付 SDK 均未变更。

## 菜单重构结果

tenant_admin 菜单调整为企业租户管理员视角，包含：

- 控制台
- 用户管理
- 渠道管理
- 订阅管理
- 账单中心
- 卡券管理
- 发票管理
- 使用日志
- 额度与用量
- 支付中心
- 收益分成
- 企业设置

同时保留 root 与其他角色原有菜单过滤逻辑，新增商业页面有对应 RBAC route guard，避免菜单隐藏后仍可通过 URL 弱访问。

## 新增页面清单

- `/quota-dashboard`
- `/usage-analytics`
- `/billing-dashboard`
- `/payment-center`
- `/revenue-share`

`routeTree.gen.ts` 已重新生成，以上页面均接入 TanStack Router。

## 页面与底层能力对齐关系

- Quota Dashboard：基于 billing summary 展示总额度、可用额度、冻结额度 foundation 状态、已消耗额度，并通过 billing records 展示最近额度变动。
- Usage Analytics：基于 billing summary 和 usage records 展示请求量、input tokens、output tokens、total tokens、模型排行、Provider 排行。
- Billing Dashboard：基于 billing summary 和 billing records 展示今日消费 foundation preview、本月消费、累计消费、最近账单、模型消费排行。
- Payment Center：基于 payment order 与 bank transfer admin API 展示支付订单、银行转账记录、支付状态、人工审核状态。
- Revenue Share：基于 finance summary 展示渠道收益、分销收益、待结算金额、已结算金额；结算能力未开启时显示 Foundation Preview，不显示空白页。

## i18n 修复

- 新增商业页面、菜单、品牌文案已补齐 `en / zh / fr / ja / ru / vi` locale。
- 中文菜单中的 `Billing`、`Wallet`、`Invoice` 已有中文值。
- 中文旧品牌可见值已清理，不再显示 legacy brand。
- `npm run i18n:sync` 已执行，报告无 missing key。报告中的 untranslated 计数主要来自既有品牌名、协议词和历史未翻译项，不属于本轮新增 key。

## 错误处理确认

- API 全局响应拦截器：`401` 清理 auth 并跳转 `/sign-in`，`403` 跳转 `/403`，`500+` 跳转 `/500`。
- `_authenticated` 父路由增加 role path guard，tenant_admin 访问受限个人开发者入口时跳转 `/403`。
- 新增商业页面使用 `requirePermission`，无权限时通过 router redirect 到 `/403`，页面不会因 403 崩溃。

## 测试结果

- `npm run typecheck`：通过。
- `npm run build`：通过。
- `go test ./model ./service ./controller ./router ./middleware`：通过。首次沙箱执行因 Go cache trim 写入 `/home/boris/.cache/go-build/trim.txt` 被拒失败，提权重跑后通过。

## 已知遗留问题

- Revenue share settlement、quota historical trend、daily billing breakdown 仍是 Foundation Preview，等待后续后端 API。
- i18n sync report 仍存在历史 untranslated 计数，需要单独翻译清理，不影响本轮新增 key 完整性。
- 技术集成中的 legacy 协议 id 和更新检查仓库路径按约束保留，不作为用户可见产品品牌处理。
