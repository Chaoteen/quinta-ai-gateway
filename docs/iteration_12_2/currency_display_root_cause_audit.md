# Iteration 12.2.1 Currency Display Root Cause Audit

## 背景

线上 `/api/status` 已返回：

- `custom_currency_symbol = ¥`
- `docs_link = /docs`
- `system_name = Quinta AI Gateway`

后端系统配置已正确，但登录后控制台的 Billing、Subscription、Voucher、Invoice、Payment、Revenue Share 相关金额仍显示 `$`。

## 根因

上一轮已把主要商业页面金额显示集中到 `commercial-display.ts`，但 `formatDisplayMoney(value, currency)` 仍优先使用页面或接口传入的 `currency` 字段。

因此当接口行数据携带 `currency = USD` 时，前端仍执行：

- `Intl.NumberFormat(..., { style: 'currency', currency: 'USD' })`
- 输出 `$123.45`

这绕过了 `/api/status` 中已经正确返回的 `custom_currency_symbol = ¥`。

## 搜索与残留分类

执行搜索范围：

- `$`
- `USD`
- `currency: 'USD'`
- `currency="USD"`
- `Intl.NumberFormat`
- `formatCurrency`
- `formatQuota`
- `formatAmount`

### 用户可见金额路径

以下页面金额已统一经过 `commercial-display.ts`：

- Billing Portal: `formatDisplayMoney`
- Billing Dashboard: `formatDisplayMoney`
- Payment Center: `formatDisplayMoney`
- Invoice: `formatDisplayMoney`
- Subscription 列表与弹窗: `formatDisplayMoney`
- Revenue Share: `formatDisplayMoney`
- Finance Console 支付、收益分成、带 currency 的金额: `formatDisplayMoney`

### 仍存在但不是硬编码美元的命中

- API 路径模板字符串中的 `$`，例如 `` `/api/.../${id}` ``。
- `#${id}` 这类编号展示。
- `subscriptions/types.ts` 中 `currency` schema 的默认值 `USD`，属于接口数据默认值，不再决定商业金额最终展示符号。
- `commercial-display.ts` 中保留 `$` 作为系统配置确认为 USD 时的合法回退。
- `Intl.NumberFormat` 在 `commercial-display.ts` 中用于数字和系统货币格式化，不再绕过系统配置。

## 修复

### `commercial-display.ts`

`formatDisplayMoney` 改为先读取 `useSystemConfigStore.getState().config.currency`：

- 如果系统配置存在非默认 `customCurrencySymbol`，优先使用该符号。
- 如果 `quotaDisplayType = CNY`，显示 `¥`。
- 如果 `quotaDisplayType = CUSTOM`，显示自定义符号。
- 只有系统明确为 USD 或无自定义符号时才回退 `$` / `Intl.NumberFormat(... USD)`。

因此即使调用方传入 `formatDisplayMoney(123.45, 'USD')`，只要 `/api/status` 同步到系统配置中的 `customCurrencySymbol = ¥`，页面显示也会是：

```text
¥123.45
```

### `Finance Console`

`formatQuota(value, currency)` 中带 currency 的分支改为调用 `formatDisplayMoney`。只有真正的 `QUOTA` 值继续显示为额度单位。

## 修复文件

- `web/default/src/lib/commercial-display.ts`
- `web/default/src/features/finance-console/index.tsx`

## 测试结果

- `npm run typecheck`：通过。
- `npm run build`：通过。

## 已知边界

- 本轮不修改后端、不修改数据库、不修改 Billing Runtime / Quota Engine。
- 系统设置页中的支付配置表单、订阅计划编辑表单仍可保存接口要求的 currency 字段；这些是配置输入，不是最终金额展示。
