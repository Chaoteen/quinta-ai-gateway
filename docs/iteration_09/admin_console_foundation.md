# Iteration 9.1 Frontend RBAC & Admin Console Foundation

## 1. 当前目标

本轮基于 `platform_readiness_audit.md` 的结论，开始实现受控 Alpha 所需的后台操作基础能力。

核心原则：

- 前端读取并使用 `role_key`；
- Admin 菜单按 `role_key` 精准展示；
- 页面 guard 与后端 RoleAuth 边界对齐；
- 新增多租户结构只读页面；
- 不开放 Tenant / Organization / Department / DistributionChannel 的创建、编辑、删除；
- 不做 Wallet、Ledger、Invoice 或 Billing 重构。

## 2. 已完成能力

### role_key 读取

前端用户态新增支持字段：

- `role_key`
- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`

后端登录响应和 `/api/user/self` 响应补充返回上述只读字段。前端在 `role_key` 缺失时继续兼容 legacy 数值角色：

- `role=100` 映射为 `root`
- `role=10` 映射为 `tenant_admin`
- 其他映射为 `user`

### Admin 菜单矩阵

Admin 菜单已按 `role_key` 过滤：

| 角色 | 可见 Admin 菜单 |
| --- | --- |
| root | 全部 |
| tenant_admin | Users、Channels、Subscriptions、Logs |
| organization_admin | Users、Subscriptions、Logs |
| finance | TopUp、Redemption、Subscriptions、Statistics |
| ops | Channels、Logs |
| auditor | Logs、Statistics、Subscriptions |
| user | 不显示 Admin 菜单 |

### 页面 Guard

页面级 guard 已改为 `role_key` 判定：

- Users：`tenant_admin`、`organization_admin`、root
- Channels：`tenant_admin`、`ops`、root
- Subscriptions：`tenant_admin`、`organization_admin`、`finance`、`auditor`、root
- Redemption：`finance`、root
- Logs：`tenant_admin`、`organization_admin`、`ops`、`auditor`、root
- Models：root
- System Settings：root
- TopUp：`finance`、root
- Tenant / Organization / Department / DistributionChannel 只读页：root

### Users 页面增强

Users 表格新增只读展示列：

- `role_key`
- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`

不改变用户创建、编辑、删除、额度调整等现有写操作。

### 只读管理页面

新增 root-only 只读页面：

- Tenants
- Organizations
- Departments
- Distribution Channels

对应后端新增 root-only GET API：

- `GET /api/admin_console/tenants`
- `GET /api/admin_console/organizations`
- `GET /api/admin_console/departments`
- `GET /api/admin_console/distribution_channels`

这些接口只查询列表，不提供创建、编辑、删除。

### TopUp 只读入口

新增 `/topup` 页面，复用现有 `GET /api/user/topup` 查询充值记录。

该页面只展示充值记录，不暴露 `topup complete` 或任何 billing mutation。

## 3. 明确未做

本轮未实现：

- Tenant CRUD
- Organization CRUD
- Department CRUD
- DistributionChannel CRUD
- Wallet
- Ledger
- Invoice
- Billing 重构
- organization_admin 写操作
- finance mutation
- auditor mutation

## 4. Alpha 意义

本轮完成后，受控 Alpha 可以具备更清晰的后台入口：

- root 可进行全局后台测试；
- tenant_admin 可进行租户内用户、渠道、订阅、日志测试；
- organization_admin 可进行组织内用户、订阅、日志只读测试；
- finance 可进行财务只读和兑换码管理入口测试；
- ops 可进行渠道运维和日志测试；
- auditor 可进行审计只读测试。

前端菜单和页面 guard 不再只依赖 legacy 数值 role，降低“菜单可见但 API 拒绝”的概率。

## 5. 验收记录

已执行：

- `npm build`：npm 不支持该命令，提示应使用 `npm run build`。
- `npm run build`：通过。
- `npm run typecheck`：通过。
- `GOCACHE=/tmp/quinta-go-build-cache go test ./common ./model ./controller ./service ./router ./middleware`：待最终验收执行。

## 6. 后续建议

Iteration 9.2 建议继续：

1. 为 role_key 菜单矩阵补充前端单元或路由测试。
2. 将 Users 页面中的 role_key 从只读展示升级为受控编辑，但只允许 root 或明确授权角色修改。
3. 为 Tenant / Organization / Department / DistributionChannel 增加 scoped read，而不是只保留 root-only 全局读。
4. 为 TopUp、Redemption、Subscriptions 拆分 finance/auditor 只读视图和 root mutation 视图。
5. 补充审计日志页面，展示高危操作 actor、target、ownership 和 before/after。

