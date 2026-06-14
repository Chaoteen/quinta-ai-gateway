# Iteration 12.2 Landing Page Productization

## 重构目标

将首页从“开发者 API 网关首页”升级为“企业级 AI Gateway + MaaS 平台首页”，突出 Quinta AI Gateway 在统一模型接入、额度、计费、订阅、卡券、发票、多租户和未来 Agent Marketplace 方向的商业价值。

本轮仅重构首页与公开营销页面，不修改后端、数据库、支付逻辑、Billing Runtime 或 Quota Engine。

## 新页面结构

首页默认结构调整为：

1. Hero
2. Platform Architecture
3. Core Capabilities
4. Metrics
5. Product Roadmap
6. Pricing Preview
7. Footer

Hero 保留主标题：

- 统一 API 网关，服务于所有 AI 模型

Hero 副标题调整为：

- 企业级 AI Gateway 与 MaaS 平台，统一接入模型、管理额度、计费与租户权限。

Hero 按钮调整为：

- 主按钮：开始使用
- 次按钮：查看产品

## 删除内容

已从首页完全移除旧 API Demo Window：

- `Responses API`
- `POST /v1/responses`
- `curl`
- API Request / API Response 模拟窗口
- Chat / Claude / Gemini Tab

同时删除不再引用的旧首页 demo 和旧开发者叙事 section：

- `web/default/src/features/home/components/hero-terminal-demo.tsx`
- `web/default/src/features/home/components/sections/how-it-works.tsx`
- `web/default/src/features/home/components/sections/cta.tsx`

## 新增模块

### Platform Architecture

新增平台架构图区块：

- OpenAI / Claude / Gemini / DeepSeek / Qwen / Llama
- Quinta AI Gateway
- Quota Engine / Billing / Subscription / Voucher / Invoice
- Enterprise AI Apps / Agent Marketplace / Knowledge Base / Workflow

### Core Capabilities

新增五个核心能力模块：

- 统一模型接入：一次接入，多模型切换。
- 额度与计费：Quota Engine、Usage Metering、Billing Runtime。
- 企业多租户：Tenant、Organization、Department。
- 支付与财务：Payment、Voucher、Invoice、Revenue Share。
- Agent Marketplace：未来支持 Skill、Agent、SaaS Marketplace。

### Metrics

新增商业指标区块，不伪造客户数量：

- 支持模型：20+
- 统一网关：1
- 企业租户：∞
- 支持计费能力：100%

### Product Roadmap

新增产品路线图区块：

- Phase 1：AI Gateway
- Phase 2：MaaS Platform
- Phase 3：Agent Marketplace
- Phase 4：Enterprise AI OS

### Pricing Preview

新增定价预览：

- Token 包
- Subscription
- Voucher
- Enterprise Plan

“查看定价”按钮跳转 `/pricing`。

## 品牌升级说明

首页定位已从 OpenAI Proxy / OneAPI / NewAPI 时代的开发者 API 示例，升级为 Quinta AI Gateway 企业级 AI Gateway 与 MaaS 平台。

`/about` 默认空状态补充 Quinta AI Gateway 产品定位说明，避免未配置 About 内容时只显示项目占位信息。

源码范围审计：

```bash
rg -n "New API|NewAPI|OneAPI|One API|Responses API|POST /v1/responses|curl|API Request|API Response" web/default/src/features/home web/default/src/features/about
```

结果：无命中。

说明：`web/default/src/i18n/locales/zh.json` 中仍存在后台设置、API 调试或系统能力相关的 `Responses API` / `curl` 翻译 key，不属于本轮首页用户可见范围。

## 移动端适配

本轮首页模块使用响应式布局：

- Hero：`clamp()` 标题、移动端纵向按钮、无代码窗口溢出风险。
- 架构图：Provider 为 2 列移动布局，商业能力和企业应用在小屏下单列/双列堆叠。
- Feature 卡片：移动端单列，768px 以上双列，桌面三列。
- Metrics：移动端双列，桌面四列。
- Roadmap：移动端单列，桌面四列。
- Pricing Preview：移动端单列，桌面四列。

当前环境未安装浏览器截图/E2E 工具，本轮完成静态响应式检查和生产构建验证。

## 测试结果

- `cd web/default && npm run i18n:sync`：通过。
- `cd web/default && npm run typecheck`：通过。
- `cd web/default && npm run build`：通过。
- `git diff --check`：通过。
