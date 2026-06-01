# Iteration 7.6 Catalog & Subscription Read Adoption

审计/实施日期：2026-06-01

目标：只迁移低风险只读接口，不开放任何写操作。

## 1. 本次迁移范围

本次仅覆盖以下 GET 接口：

| 接口 | 迁移后权限 | 说明 |
| --- | --- | --- |
| `GET /api/models/` | `catalogReadAuth` | 模型 catalog 列表只读，允许 `tenant_admin`、`ops`、`auditor`，root 仍可全局读取 |
| `GET /api/models/search` | `catalogReadAuth` | 模型 catalog 搜索只读 |
| `GET /api/models/:id` | `catalogReadAuth` | 模型 catalog 详情只读 |
| `GET /api/channel/models` | `catalogReadAuth` | 渠道可选模型 catalog 只读 |
| `GET /api/subscription/admin/users/:id/subscriptions` | `subscriptionReadAuth`，新增 `organization_admin` | organization_admin 只能读取本组织用户订阅 |
| `GET /api/log/stat` | `operationalFinanceReadAuth` | organization_admin 只能读取本组织统计，无组织归属 fail closed |

## 2. 权限边界

本次保持以下边界不变：

- root：继续拥有全局只读能力。
- tenant_admin：可读取本 tenant 范围内的订阅和统计；catalog 类只读信息按现有 catalog read 边界开放。
- organization_admin：只可读取本 organization 用户的订阅和本 organization 的统计。
- organization_admin 且 `organization_id = 0`：必须 fail closed，不允许 fallback 到 tenant scope。
- finance/auditor/ops：保留既有只读角色边界；finance 未获得 catalog read，ops 未获得 subscription/finance read。

## 3. 实现要点

- 模型 catalog 和 `channel/models` 仅调整 router 权限到既有只读角色组合，没有修改模型、供应商或渠道写操作。
- `GET /api/subscription/admin/users/:id/subscriptions` 从 tenant-only 校验改为 `AccessScope` 校验：
  - 先用 `AllowsOwnership()` 校验目标用户是否在当前访问范围内；
  - 再用 `ApplyOwnershipScope()` 过滤 `user_subscriptions`。
- `GET /api/log/stat` 改为复用 `operationalReadAccessScope()` 和 `SumUsedQuotaByAccessScope()`：
  - root 全局；
  - tenant_admin/finance/auditor tenant scoped；
  - organization_admin organization scoped；
  - organization_admin 无组织归属 fail closed。

## 4. 未开放能力

本次未开放任何写操作，包括：

- 模型写操作：`POST/PUT/DELETE /api/models/**`
- 供应商写操作：`POST/PUT/DELETE /api/vendors/**`
- 渠道写操作：`POST/PUT/DELETE /api/channel/**` 以及渠道 test/balance/upstream mutation
- 订阅创建、绑定、失效、删除
- 充值、兑换码、计费 mutation
- Relay、OAuth、User Management

## 5. 测试覆盖

新增/调整的测试覆盖：

- catalog read：`/api/models/`、`/api/models/search`、`/api/models/:id` 对 `tenant_admin`、`ops`、`auditor`、root 放行，对 finance/user 拒绝。
- channel model catalog：`/api/channel/models` 按 catalog/channel read 边界放行。
- subscription read：organization_admin 可读取本组织用户订阅，拒绝其他组织用户，`organization_id = 0` 拒绝。
- log stat：organization_admin 统计只包含本组织 quota，`organization_id = 0` 拒绝。
- deferred/admin-only 测试同步移除本次已迁移的只读接口，继续保留未迁移接口。

## 6. 验收

验收命令：

```bash
GOCACHE=/tmp/quinta-go-build-cache go test ./common ./model ./controller ./service ./router ./middleware
```

本次迁移不需要数据库 migration，不改变表结构。
