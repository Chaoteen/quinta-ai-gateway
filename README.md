# Quinta AI Gateway

Quinta AI Gateway 是基于 New API 二次开发的多租户 AI API 网关项目，面向企业级 AI API 中转、模型接入、渠道管理、计费结算和租户隔离场景。

## 项目定位

本项目目标是构建一套可用于商业化运营的 AI API Gateway / MaaS 基础平台，支持：

- 多租户数据归属与后台隔离
- 用户、组织、部门、分销渠道管理基础
- 模型渠道接入与路由管理
- Token 用量记录与计费基础
- 充值、兑换码、订阅等业务记录归属
- 管理后台租户级数据隔离
- 后续扩展分销结算、企业客户管理和 Agent 服务入口

## 当前改造重点

当前版本已在 New API 基础上完成第一阶段多租户改造：

- 新增 tenant / organization / department / distribution channel 基础模型
- 核心业务表增加 tenant_id、organization_id、department_id、distribution_channel_id
- 认证用户归属信息写入 Gin Context 和 RelayInfo
- logs、topups、redemptions、tasks、subscriptions、midjourney 等业务数据写入 ownership 快照
- 后台列表增加 tenant scope 过滤
- 后台详情、更新、删除、批量操作增加 tenant access check
- root 用户保留平台级全局管理视角
- 普通用户 self 接口保持 user_id 限制

## 后续计划

- 租户管理后台
- 分销渠道和卡券核销
- 租户级计费、结算、对账
- quota_data / rankings / perf_metrics 的租户统计口径设计
- 企业客户控制台
- Agent 服务入口和 API 产品化

## 说明

本项目基于开源项目 New API 进行二次开发。原项目版权和许可证信息请参考仓库中的 LICENSE、NOTICE 和 THIRD-PARTY-LICENSES.md。

