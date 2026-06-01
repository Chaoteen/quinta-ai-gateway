# Iteration 7.2 Organization Scope Foundation

## 设计目标

Iteration 7.2 引入组织级访问范围基础设施，但暂不在业务路由中启用新的 `organization_admin` 访问能力。

本阶段目标是为后续 `organization_admin` 路由迁移提供安全基础：

- 保持现有 tenant-only 行为兼容；
- 保留 root 的全局访问能力；
- 保留 tenant_admin 的租户级访问能力；
- 将 `organization_admin` 限定为组织级访问；
- 当 `organization_admin` 缺少 `organization_id` 时 fail closed；
- 支持显式 department 过滤，为后续部门级 scope 工作预留基础。

本阶段不修改 controller 业务逻辑、不修改计费逻辑、不修改 relay 逻辑、不修改 channel mutation 逻辑，也不修改 subscription mutation 逻辑。

## 实现内容

本阶段新增的是 common/model 层基础设施：

- RBAC helper 扩展；
- `model.AccessScope`；
- `AccessScopeFromContext(c)`；
- `ApplyOwnershipScope(db, tableAliasOrName, scope)`；
- `AllowsOwnership(scope, tenantId, organizationId, departmentId)`；
- 对 root、tenant_admin、organization_admin、department 显式过滤和 fail closed 的单元测试；
- 本文档。

## RBAC Helper

新增 helper：

- `IsOrganizationAdminRole(roleKey string)`
- `IsScopedAdminRole(roleKey string)`

兼容的 role key 保持不变：

- `root`
- `tenant_admin`
- `organization_admin`
- `finance`
- `ops`
- `auditor`
- `user`

`IsScopedAdminRole()` 当前只识别：

- `root`
- `tenant_admin`
- `organization_admin`

`finance`、`ops`、`auditor`、`user` 不被视为 scoped admin。

## AccessScope

新增 `model.AccessScope`：

| 字段 | 含义 |
|---|---|
| `TenantId` | 租户边界 |
| `OrganizationId` | 组织边界 |
| `DepartmentId` | 可选部门边界 |
| `RoleKey` | 标准化后的 role key |
| `IsRoot` | 是否 root 全局访问 |

新增 `AccessScopeFromContext(c)`：

| 角色 | Scope 行为 |
|---|---|
| `root` | 全局访问 |
| `tenant_admin` | tenant 级访问 |
| `organization_admin` | tenant + organization 级访问 |
| 其它角色 | 保持当前 tenant 级行为 |

当前实现保持基础设施性质，不主动替换现有 controller 逻辑。后续路由迁移时应显式使用 `AccessScope` 相关 helper。

## Ownership Scope Helper

新增 `ApplyOwnershipScope(db, tableAliasOrName, scope)`：

- root：不追加过滤条件；
- 非 root：按 `tenant_id` 过滤；
- organization scope：在 tenant 过滤基础上追加 `organization_id` 过滤；
- 显式 department scope：在 organization 过滤基础上追加 `department_id` 过滤；
- 无效的 `organization_admin` scope：追加 `1 = 0`，确保查不到任何数据。

新增 `AllowsOwnership(scope, tenantId, organizationId, departmentId)`：

- root 允许所有 ownership；
- tenant_admin 允许当前 tenant；
- organization_admin 只允许当前 organization；
- `organization_admin` 且 `organization_id = 0` 拒绝所有访问；
- 当 `scope.DepartmentId > 0` 时，只允许匹配的 department。

## Fail Closed 规则

`organization_admin` 必须具备有效的 `organization_id`：

```text
organization_id > 0
```

如果 `organization_admin` 的 `organization_id = 0`，则：

- `AllowsOwnership()` 返回 `false`；
- `ApplyOwnershipScope()` 追加 `1 = 0`；
- 不 fallback 到 tenant 级访问；
- 不自动放大为 tenant_admin；
- 不自动继承任何更高权限。

该规则用于避免组织管理员在 ownership 缺失时获得租户级可见性。

## 兼容性说明

- 现有 `TenantScope` 未修改。
- 现有 relay tenant isolation 未修改。
- 现有 `RoleAuth`、`RootAuth`、`AdminAuth`、`UserAuth` 行为未修改。
- 现有 controller 路由 ownership 检查未修改。
- 现有 billing、relay、channel mutation、subscription mutation 流程未修改。
- 本阶段没有给 `organization_admin` 开放新的业务路由。
- 本阶段只建设 common/model 层 scope foundation。

## 测试覆盖

新增单元测试覆盖：

1. root 可以访问全部 ownership 边界；
2. tenant_admin 可以访问本 tenant 资源；
3. organization_admin 可以访问本 organization 资源；
4. organization_admin 且 `organization_id = 0` 必须 fail closed；
5. 跨 organization 访问被拒绝；
6. 跨 tenant 访问被拒绝；
7. 显式 department 过滤存在时会按 `department_id` 过滤。

## 测试结果

原始验收命令：

```bash
go test ./common ./model ./controller ./service ./router ./middleware
```

在当前环境中，该命令因为默认 Go build cache 目录只读而在包执行前失败：

```text
open /home/boris/.cache/go-build/...: read-only file system
```

使用可写 cache 后测试通过：

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
