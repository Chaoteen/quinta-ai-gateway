# Iteration 12.2 Tenant Admin Console I18N, RBAC and Currency Audit

## 线上发现问题

- tenant_admin 登录后菜单中存在英文 `Invoice`、`Voucher`。
- Invoice 页面字段、按钮、表格列、状态和空状态存在英文或后端枚举直出。
- Voucher 页面存在英文标题、状态、类型和兑换结果枚举直出。
- “任务日志”和“使用日志”线上点击出现 403，需要确认菜单与权限边界。
- 商业化、财务、订阅页面存在默认 `$` 或 `USD` 金额显示，中文环境不符合预期。

## 修复范围

- 前端菜单与页面显示：发票、卡券、账单、支付、收益分成、额度与用量、用量分析、使用日志相关入口。
- 前端 i18n：补齐 en / zh / fr / ja / ru / vi 新增 key，重点修复 zh。
- 前端 RBAC 静态巡检：确认 tenant_admin 菜单和路由权限映射。
- 金额展示：新增统一商业展示格式化工具，替换页面内本地 `formatMoney(..., 'USD')` 和硬编码 `$`。

## 菜单中文化结果

tenant_admin 菜单当前可见入口：

| 中文菜单 | URL | 结果 |
| --- | --- | --- |
| 控制台 | `/dashboard/overview` | 可见 |
| 用户管理 | `/users` | 可见 |
| 渠道管理 | `/channels` | 可见 |
| 订阅管理 | `/subscriptions` | 可见 |
| 账单中心 | `/billing-dashboard` | 可见 |
| 卡券管理 | `/admin/vouchers` | 可见 |
| 发票管理 | `/admin/invoices` | 可见 |
| 使用日志 | `/usage-logs/common` | 可见 |
| 额度与用量 | `/quota-dashboard` | 可见 |
| 用量分析 | `/usage-analytics` | 可见 |
| 支付中心 | `/payment-center` | 可见 |
| 收益分成 | `/revenue-share` | 可见 |
| 企业设置 | `/profile` | 可见 |

菜单分组标题同步中文化为“企业管理”“商业中心”“运营中心”。

## Invoice / Voucher 中文化结果

- Invoice 页面改为通过统一枚举显示函数渲染状态和类型，避免 `ACTIVE`、`PENDING`、`VAT_NORMAL`、`COMPANY` 等枚举穿透到中文界面。
- Invoice 金额改为统一金额格式化，默认按语言选择显示货币。
- Voucher 管理页标题改为 `Voucher Management`，中文为“卡券管理”。
- Voucher 状态、类型、兑换结果和筛选下拉全部改为翻译后显示，避免 `UNUSED`、`REDEEMED`、`TOKEN`、`SUBSCRIPTION` 等枚举直出。

## 403 权限分析

- tenant_admin 菜单不展示“任务日志”。
- tenant_admin 菜单展示“使用日志”，指向 `/usage-logs/common`。
- `RBAC_PERMISSION.USAGE_LOGS` 包含 tenant_admin，`/usage-logs/common` 映射到 `USAGE_LOGS`。
- tenant_admin 受限路径包含 `/usage-logs/task` 和 `/usage-logs/drawing`，直接访问会跳转 `/403`。
- `/usage-logs/common` 页面本身不展示 task/drawing 切换，避免用户从可用使用日志页点到无权限任务日志。

## 金额格式化策略

新增 `web/default/src/lib/commercial-display.ts`：

- `formatDisplayMoney(value, currency)`：金额显示统一入口。
- 中文环境默认币种为 `CNY`，显示 `¥`。
- 非中文环境默认币种为 `USD`，显示 `$`。
- 若接口明确返回 currency，则优先使用接口币种。
- `formatDisplayNumber` 和 `formatDisplayDateTime` 统一数字与时间本地化。

已替换范围：

- Billing Dashboard / Billing Portal
- Payment Center
- Invoice
- Voucher 相关枚举显示
- Subscription 列表、购买弹窗、用户订阅弹窗
- Revenue Share
- Finance Console

硬编码 `$` 搜索结果中剩余命中为 API 模板字符串、`#id` 展示、百分比模板或普通字符串拼接，不再是页面金额货币符号。

## 全菜单巡检表

| URL | 路由存在 | 菜单可见 | RBAC 允许 | 可能 403 | 中英文混杂修复 | 金额修复 | 品牌残留 |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `/dashboard/overview` | 是 | 是 | 是 | 否 | 本轮未发现新增问题 | 不涉及 | 未命中 |
| `/users` | 是 | 是 | 是 | 否 | 本轮未发现新增问题 | 不涉及 | 未命中 |
| `/channels` | 是 | 是 | 是 | 否 | 本轮未发现新增问题 | 不涉及 | 未命中 |
| `/subscriptions` | 是 | 是 | 是 | 否 | 补充 Subscription key | 已修复 `$` | 未命中 |
| `/billing-dashboard` | 是 | 是 | 是 | 否 | 已补齐 key | 已修复 | 未命中 |
| `/admin/vouchers` | 是 | 是 | 是 | 否 | 已修复 | 不涉及金额符号 | 未命中 |
| `/admin/invoices` | 是 | 是 | 是 | 否 | 已修复 | 已修复 | 未命中 |
| `/usage-logs/common` | 是 | 是 | 是 | 否 | 使用日志入口中文化 | 使用现有计费格式 | 未命中 |
| `/quota-dashboard` | 是 | 是 | 是 | 否 | 已补齐 key | 不涉及金额符号 | 未命中 |
| `/usage-analytics` | 是 | 是 | 是 | 否 | 已补齐 key | 不涉及金额符号 | 未命中 |
| `/payment-center` | 是 | 是 | 是 | 否 | 已补齐 key | 已修复 | 未命中 |
| `/revenue-share` | 是 | 是 | 是 | 否 | 已补齐 key | 已修复 | 未命中 |
| `/profile` | 是 | 是 | 是 | 否 | 本轮未发现新增问题 | 不涉及 | 未命中 |

## 测试结果

- `npm run i18n:sync`：通过，en / zh / fr / ja / ru / vi missingCount 均为 0。
- `npm run typecheck`：通过。
- `npm run build`：通过。
- `git diff --check`：通过。

## 遗留问题

- fr / ja / ru / vi 仍存在历史 untranslated 统计，本轮保证 missing key 为 0，未做全量专业翻译。
- zh 仍存在历史 untranslated 统计，本轮已覆盖 tenant_admin 控制台重点菜单、发票、卡券、商业化页面和状态词。
- 本轮为静态和构建验证，未启动真实后端用测试账号执行浏览器登录巡检。
