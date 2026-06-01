# Iteration 7.3 Organization Admin User Read Adoption

## 目标

本阶段只迁移用户读取类接口，让 `organization_admin` 可以读取本组织用户。

迁移范围：

- `GET /api/user/`
- `GET /api/user/search`
- `GET /api/user/:id`

本阶段不开放用户写操作，不修改 Billing、Relay、Channel、Subscription 逻辑。

## 实现内容

### 路由迁移

新增 user read auth：

```go
RoleAuth(tenant_admin, organization_admin)
```

迁移：

- `GET /api/user/`
- `GET /api/user/search`
- `GET /api/user/:id`

保留不变：

- `POST /api/user/`
- `PUT /api/user/`
- `DELETE /api/user/:id`
- `POST /api/user/manage`
- `POST /api/user/topup/complete`
- OAuth 绑定管理
- Passkey 管理
- 2FA 管理

这些接口仍由原有 `AdminAuth()` / `RootAuth()` 保护。

### Scope 使用

用户列表和搜索改为使用：

- `AccessScopeFromContext(c)`
- `GetAllUsersByAccessScope(...)`
- `SearchUsersByAccessScope(...)`
- `ApplyOwnershipScope(...)`

用户详情使用：

- `AccessScopeFromContext(c)`
- `AllowsOwnership(...)`

## 权限边界

### root

root 保持全局读取能力，可以读取全部 tenant / organization 用户。

### tenant_admin

tenant_admin 行为保持不变：

- 可以读取本 tenant 用户；
- 不能读取其它 tenant 用户；
- 不受 organization 限制。

### organization_admin

organization_admin 新增读取能力：

- 可以读取本 tenant、本 organization 用户；
- 不能读取同 tenant 其它 organization 用户；
- 不能读取其它 tenant 用户；
- `organization_id = 0` 时直接拒绝访问。

### user

普通 user 不获得管理端用户读取能力。

## Fail Closed

`organization_admin` 必须具备有效 organization ownership：

```text
organization_id > 0
```

如果 `organization_id = 0`：

- `GET /api/user/` 返回失败；
- `GET /api/user/search` 返回失败；
- `GET /api/user/:id` 返回失败；
- 不 fallback 到 tenant 级读取；
- 不继承 tenant_admin 权限。

## 未开放范围

本阶段明确不开放：

- 用户创建；
- 用户更新；
- 用户删除；
- 用户角色管理；
- 用户状态管理；
- OAuth 绑定管理；
- Passkey 管理；
- 2FA 管理；
- TopUp complete；
- Billing；
- Relay；
- Channel；
- Subscription。

## 测试覆盖

新增/扩展 router 层测试覆盖：

- organization_admin 可以读取本 organization 用户列表；
- organization_admin 搜索结果只包含本 organization 用户；
- organization_admin 可以读取本 organization 用户详情；
- organization_admin 不能读取同 tenant 其它 organization 用户；
- organization_admin 不能读取其它 tenant 用户；
- organization_admin 且 `organization_id = 0` 时列表、搜索、详情全部拒绝；
- tenant_admin 仍可读取本 tenant 内不同 organization 用户；
- tenant_admin 仍不能读取其它 tenant 用户；
- root 仍可读取其它 tenant 用户；
- 普通 user 不能访问管理端用户读取接口；
- organization_admin 不能访问用户写操作和 OAuth / Passkey / 2FA 管理接口。

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
