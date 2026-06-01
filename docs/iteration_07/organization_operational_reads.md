# Iteration 7.4 Organization Scoped Operational Reads

## 目标

本阶段开放 `organization_admin` 对运营类只读数据的访问能力，并确保所有读取都被限制在本组织内。

迁移范围：

- `GET /api/log/`
- `GET /api/task/`
- `GET /api/mj/`
- `GET /api/user/topup`
- `GET /api/redemption/`

说明：当前仓库中管理端 topup 只读路由为 `GET /api/user/topup`，不存在 `GET /api/topup/` 路由，因此本阶段按现有路由迁移。

## 实现内容

本阶段将上述运营只读接口接入：

- `AccessScopeFromContext()`
- `ApplyOwnershipScope()`
- `AllowsOwnership()`

新增或使用的 model 查询能力：

- `GetAllLogsByAccessScope(...)`
- `TaskGetAllTasksByAccessScope(...)`
- `TaskCountAllTasksByAccessScope(...)`
- `GetAllTasksByAccessScope(...)`
- `CountAllTasksByAccessScope(...)`
- `GetAllTopUpsByAccessScope(...)`
- `SearchAllTopUpsByAccessScope(...)`
- `GetAllRedemptionsByAccessScope(...)`

旧的 tenant scope 查询函数保留兼容。

## 权限边界

### root

root 可查看全部 tenant、全部 organization 的运营只读数据。

### tenant_admin

tenant_admin 行为保持不变：

- 可查看本 tenant 的运营只读数据；
- 不可查看其它 tenant 数据；
- 不受 organization 限制。

### organization_admin

organization_admin 新增运营只读能力：

- 只能查看本 tenant、本 organization 数据；
- 不能查看同 tenant 其它 organization 数据；
- 不能查看其它 tenant 数据；
- `organization_id = 0` 时必须 fail closed。

## Fail Closed

`organization_admin` 必须满足：

```text
organization_id > 0
```

如果 `organization_id = 0`：

- `GET /api/log/` 拒绝；
- `GET /api/task/` 拒绝；
- `GET /api/mj/` 拒绝；
- `GET /api/user/topup` 拒绝；
- `GET /api/redemption/` 拒绝；
- 不 fallback 到 tenant；
- 不继承 tenant_admin 权限。

## 未开放范围

本阶段不开放任何 mutation：

- Billing Mutation；
- Topup Complete；
- Subscription Create；
- Subscription Bind；
- Subscription Invalidate；
- Redemption Mutation；
- Channel Mutation；
- Relay Logic；
- OAuth；
- User Management。

## 测试覆盖

新增/扩展 router 层测试覆盖：

1. root 可查看全部运营只读数据；
2. tenant_admin 可查看本 tenant 数据；
3. organization_admin 只能查看本 organization 数据；
4. organization_admin 不能查看同 tenant 其它 organization 数据；
5. organization_admin 不能查看其它 tenant 数据；
6. organization_admin 且 `organization_id = 0` 时拒绝访问；
7. topup complete、redemption POST/PUT/DELETE 等 mutation routes 不向 organization_admin 开放。

## 测试结果

验收命令：

```bash
GOCACHE=/tmp/quinta-go-build-cache go test ./common ./model ./controller ./service ./router ./middleware
```

结果：

```text
ok github.com/Chaoteen/quinta-ai-gateway/common
ok github.com/Chaoteen/quinta-ai-gateway/model
ok github.com/Chaoteen/quinta-ai-gateway/controller
ok github.com/Chaoteen/quinta-ai-gateway/service
ok github.com/Chaoteen/quinta-ai-gateway/router
ok github.com/Chaoteen/quinta-ai-gateway/middleware
```
