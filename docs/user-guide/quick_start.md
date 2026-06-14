# 快速开始

本文面向首次使用 Quinta AI Gateway 的用户，帮助你完成从登录到第一次 API 请求的基础流程。

## 1. 登录平台

打开 Quinta AI Gateway 访问地址，使用管理员提供的账号登录。如果你的环境启用了邮箱、OAuth 或 Passkey，请按页面提示完成认证。

## 2. 查看控制台

登录后进入控制台，重点确认：

- 当前账号状态正常。
- 所属租户、组织或角色正确。
- 可用额度或订阅权益满足测试请求。
- 管理员已配置至少一个可用模型渠道。

## 3. 创建 API Key

进入 API Key 页面，创建用于应用调用的密钥。请妥善保存密钥明文，避免提交到代码仓库或公开日志。

## 4. 发起第一次请求

使用兼容 OpenAI 的客户端或 HTTP 工具，将 Base URL 指向 Quinta AI Gateway 的网关地址，并在请求头中携带 API Key。

## 5. 查看用量

请求完成后，可在使用日志、额度与用量、账单中心查看请求状态、Token 消耗和计费记录。

## 下一步

- 阅读 `docs/user-guide/api_access.md` 完成正式接入。
- 阅读 `docs/user-guide/quota_and_billing.md` 理解额度和计费。
