# Iteration 7.8 Tenant Admin Channel Read & Execute

实施日期：2026-06-01

目标：开放 `tenant_admin` / `ops` 对低风险单渠道 Channel Read & Execute 接口的访问能力。本轮不开放任何 channel 写配置、credential reveal、OAuth complete/refresh、multi_key、copy、Ollama pull/delete、upstream apply/apply_all 或全局能力。

## 1. 本次迁移接口

| 接口 | 迁移后权限 | 说明 |
| --- | --- | --- |
| `GET /api/channel/fetch_models/:id` | `tenant_admin`、`ops`、root | 单渠道 upstream models 拉取；不写 channel 配置，不返回 credential |
| `GET /api/channel/ollama/version/:id` | `tenant_admin`、`ops`、root | 单渠道 Ollama version 查询；不写本地配置，不返回 credential |
| `GET /api/channel/test/:id` | `tenant_admin`、`ops`、root | 单渠道测试；会调用上游，可能更新 response_time 或触发既有自动禁用/启用链路 |
| `GET /api/channel/update_balance/:id` | `tenant_admin`、`ops`、`finance`、root | 单渠道余额刷新；会调用上游并更新本地 balance |

## 2. 权限边界

- root：保持可访问所有本轮迁移接口，并可跨 tenant channel。
- tenant_admin：只可访问本 tenant channel。
- ops：只可访问本 tenant channel。
- finance：只允许访问 `GET /api/channel/update_balance/:id`，不允许 test、fetch_models、Ollama version。
- organization_admin：不允许访问本轮 channel ops 接口。
- user：不允许访问本轮 channel ops 接口。
- 跨 tenant channel：非 root 必须拒绝，不允许 fallback。

## 3. 实现要点

- router 新增两个只读执行类角色边界：
  - channel execute：`tenant_admin`、`ops`
  - channel balance execute：`tenant_admin`、`ops`、`finance`
- controller 继续复用现有单渠道 tenant scope 校验：
  - `requireChannelTenantAccess()`
  - `TenantScopeFromContext()`
- 未新增 credential 返回字段。
- 未新增 channel 配置写入入口。
- 未修改全局 batch/test/balance、global fetch_models、model sync、ratio sync。

## 4. 执行副作用

这些接口虽然是 GET，但不是纯读：

- `test/:id` 会向上游发送测试请求，可能消耗 provider 额度；成功/失败后会更新响应时间，错误场景可能触发既有自动禁用/启用逻辑。
- `update_balance/:id` 会请求 provider 余额接口，并更新本地 channel balance。
- `fetch_models/:id` 会请求上游 models 接口，但本轮保持为只返回结果，不应用到本地 channel 配置。
- `ollama/version/:id` 会请求远端 Ollama version；不执行 pull/delete。

## 5. 明确未开放

本轮未开放以下能力：

- channel create/update/delete
- channel copy
- channel multi_key
- channel key reveal
- Codex OAuth complete / refresh
- Ollama pull/delete
- upstream apply/apply_all
- global fetch_models
- model sync
- ratio sync
- billing
- relay
- user management

## 6. 测试覆盖

新增/调整测试覆盖：

- `tenant_admin`、`ops`、root 可到达 `fetch_models/:id`、`ollama/version/:id`、`test/:id` handler。
- `finance` 不可访问 `fetch_models/:id`、`ollama/version/:id`、`test/:id`。
- `finance` 可到达 `update_balance/:id` handler。
- `auditor`、`organization_admin`、普通 user 不可访问本轮 channel ops 接口。
- 非 root 访问其他 tenant channel 必须返回 tenant scope 拒绝。
- root 访问其他 tenant channel 保持可到达 handler。

## 7. 验收

验收命令：

```bash
GOCACHE=/tmp/quinta-go-build-cache go test ./common ./model ./controller ./service ./router ./middleware
```
