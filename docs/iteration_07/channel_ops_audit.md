# Iteration 7.7 Tenant Admin Channel Ops Audit

审计日期：2026-06-01

目标：审计 Channel / Upstream / Codex OAuth / Ollama / Balance / Test 相关管理路由。本轮只做审计文档，不修改业务代码、router、测试或功能。

## 1. 当前状态

当前 `router/api-router.go` 中 Channel 管理面的权限状态：

- 渠道列表、搜索、详情、enabled models、tag models 已经使用只读角色边界。
- `GET /api/channel/models` 已迁移到 catalog read。
- channel ops 相关执行类接口仍主要使用 `AdminAuth()`。
- channel key reveal、全局 `fix`、全局 `fetch_models` 使用 `RootAuth()`。
- 相关 controller 已经在多数单渠道/批量渠道路径中使用 `requireChannelTenantAccess()`、`requireChannelsTenantAccess()` 或 `TenantScopeFromContext()`，具备迁移为 tenant scoped 操作的基础。

本审计按四类风险标记：

- 消耗上游额度：会发起测试请求、余额查询、usage 查询、模型探测、OAuth/token 交换、Ollama 操作等。
- 写本地数据库：会更新 channel、abilities、key、balance、response_time、状态、tag、settings、缓存等。
- 写远端服务：会对远端服务产生状态变化，例如 Ollama pull/delete。
- 暴露 credential：会返回、生成、刷新、保存或可间接接触 API key / OAuth token / key preview。

## 2. 分类总览

### A. 适合 tenant_admin/ops 的只读或低风险执行类接口

这些接口可作为 Iteration 7.8 的优先迁移候选，但仍建议加审计日志和限频，因为它们不是纯 DB read。

| 接口 | 当前权限 | 消耗上游额度 | 写本地数据库 | 写远端服务 | 暴露 credential | 审计结论 |
| --- | --- | --- | --- | --- | --- | --- |
| `GET /api/channel/test/:id` | `AdminAuth` | 是，发起单渠道测试请求，可能计入 provider 用量 | 是，异步更新 response_time；错误时可能触发自动禁用/启用 | 否 | 否 | 适合 `tenant_admin/ops`，但应视为执行类接口，需限频 |
| `GET /api/channel/test` | `AdminAuth` | 是，批量测试本租户可见渠道 | 是，更新 response_time，可能自动禁用/启用渠道 | 否 | 否 | 适合 `tenant_admin/ops`，但应异步、限频、互斥执行 |
| `GET /api/channel/update_balance/:id` | `AdminAuth` | 是，调用 provider 余额接口 | 是，更新 channel balance；余额不足路径可能禁用渠道 | 否 | 否 | 适合 `tenant_admin/ops/finance`，需限频 |
| `GET /api/channel/update_balance` | `AdminAuth` | 是，批量调用 provider 余额接口 | 是，更新 balance，可能禁用余额不足渠道 | 否 | 否 | 适合 `tenant_admin/ops/finance`，需限频和批量任务保护 |
| `POST /api/channel/upstream_updates/detect` | `AdminAuth` | 是，拉取单渠道 upstream models | 是，持久化检测结果、last_check_time；auto-add 时可能更新 channel models/abilities | 否 | 否 | 适合 `tenant_admin/ops`，但需明确这是检测+本地写入，不是纯 read |
| `POST /api/channel/upstream_updates/detect_all` | `AdminAuth` | 是，批量拉取 upstream models | 是，持久化检测结果；auto-add 时可能更新 models/abilities | 否 | 否 | 适合 `tenant_admin/ops`，需批量限频和任务锁 |
| `GET /api/channel/fetch_models/:id` | `AdminAuth` | 是，按单渠道拉取 upstream models | 否，仅返回模型列表 | 否 | 否 | 适合 `tenant_admin/ops`，低风险优先候选 |
| `GET /api/channel/:id/codex/usage` | `AdminAuth` | 是，调用 Codex usage API；401/403 时会尝试 refresh token | 是，refresh 成功会更新 channel key、重置缓存 | 否 | 间接使用 OAuth token，不返回 token | 可考虑给 `tenant_admin/ops/finance`，但因可刷新 credential，建议放入 B 而非纯 A |
| `GET /api/channel/ollama/version/:id` | `AdminAuth` | 是，调用远端 Ollama version | 否 | 否 | 使用 key，但不返回 | 适合 `tenant_admin/ops`，需记录调用失败和限频 |

说明：

- `test` 和 `balance` 使用 GET，但实际有副作用，应在权限设计里按执行类 mutation 对待。
- `detect` 名义是检测，但当前会写入 channel other settings，且配置允许时可能 auto-add model，因此也不是纯 read。

### B. 适合 tenant_admin 但需要二次验证/审计日志的接口

这些接口可以面向 tenant_admin 规划，但不建议直接给普通 ops；迁移时应增加 Secure Verification、审计日志、速率限制和明确的租户边界测试。

| 接口 | 当前权限 | 消耗上游额度 | 写本地数据库 | 写远端服务 | 暴露 credential | 审计结论 |
| --- | --- | --- | --- | --- | --- | --- |
| `POST /api/channel/` | `AdminAuth` | 否 | 是，创建 channel，保存 key | 否 | 请求体包含 key | 适合 `tenant_admin`，需要二次验证和审计 |
| `PUT /api/channel/` | `AdminAuth` | 否 | 是，更新 channel/key/settings/abilities/cache | 否 | 请求体可能包含 key；multi-key append/overwrite | 适合 `tenant_admin`，需要字段级校验、二次验证、审计 |
| `DELETE /api/channel/:id` | `AdminAuth` | 否 | 是，删除 channel、刷新 cache | 否 | 否 | 适合 `tenant_admin`，需要审计和确认 |
| `DELETE /api/channel/disabled` | `AdminAuth` | 否 | 是，批量删除 disabled channel | 否 | 否 | 适合 `tenant_admin`，需审计和 dry-run/确认 |
| `POST /api/channel/batch` | `AdminAuth` | 否 | 是，批量删除 channel | 否 | 否 | 适合 `tenant_admin`，高风险批量删除，需二次验证 |
| `POST /api/channel/tag/disabled` | `AdminAuth` | 否 | 是，按 tag 批量禁用 channel/abilities | 否 | 否 | 适合 `tenant_admin/ops`，但需要审计 |
| `POST /api/channel/tag/enabled` | `AdminAuth` | 否 | 是，按 tag 批量启用 channel/abilities | 否 | 否 | 适合 `tenant_admin/ops`，但需要审计 |
| `PUT /api/channel/tag` | `AdminAuth` | 否 | 是，批量改 tag/models/group/priority/weight/override | 否 | header/param override 可能包含敏感配置 | 适合 `tenant_admin`，需要二次验证和字段审计 |
| `POST /api/channel/batch/tag` | `AdminAuth` | 否 | 是，批量设置 channel tag | 否 | 否 | 适合 `tenant_admin/ops`，需要审计 |
| `POST /api/channel/copy/:id` | `AdminAuth` | 否 | 是，复制 channel，包含原 key | 否 | 间接复制 credential | 适合 `tenant_admin`，必须审计；默认复制 key 风险较高 |
| `POST /api/channel/multi_key/manage` | `AdminAuth` | 否 | `get_key_status` 否；disable/enable/delete 会写 channel key info | 否 | key preview 暴露部分 key，mutation 接触 key 索引 | 适合 `tenant_admin`，需二次验证；`get_key_status` 可考虑拆只读 |
| `POST /api/channel/upstream_updates/apply` | `AdminAuth` | 否，使用已检测的 pending 结果 | 是，更新 channel models/settings/abilities/cache | 否 | 否 | 适合 `tenant_admin/ops`，但会改变可用模型，需审计 |
| `POST /api/channel/upstream_updates/apply_all` | `AdminAuth` | 否，使用已检测 pending 结果 | 是，批量更新 models/settings/abilities/cache | 否 | 否 | 适合 `tenant_admin`，批量高风险，需二次验证 |
| `POST /api/channel/codex/oauth/start` | `AdminAuth` | 否，生成授权 URL | 是，写 session | 否 | 返回 authorize_url，不返回 token | 适合 `tenant_admin`，需要审计 OAuth flow 启动 |
| `POST /api/channel/codex/oauth/complete` | `AdminAuth` | 是，交换 authorization code | 否，未绑定 channel 时返回生成的 key | 否 | 是，返回完整 Codex OAuth key | 暂不建议给 ops；tenant_admin 也需二次验证，见 D |
| `POST /api/channel/:id/codex/oauth/start` | `AdminAuth` | 否 | 是，写 session | 否 | 不返回 token | 适合 `tenant_admin`，需审计 |
| `POST /api/channel/:id/codex/oauth/complete` | `AdminAuth` | 是，交换 authorization code | 是，保存 channel key，重置缓存 | 否 | 保存 OAuth credential，不返回 token | 适合 `tenant_admin`，必须二次验证和审计 |
| `POST /api/channel/:id/codex/refresh` | `AdminAuth` | 是，调用 token refresh | 是，更新 channel key，重置缓存 | 否 | 不返回 token，但刷新 credential | 适合 `tenant_admin`，必须二次验证和审计 |
| `GET /api/channel/:id/codex/usage` | `AdminAuth` | 是，调用 usage；可能 refresh token | 是，可能更新 channel key | 否 | 不返回 token | 适合 `tenant_admin/finance`，需要审计和限频 |
| `POST /api/channel/ollama/pull` | `AdminAuth` | 是，占用远端带宽/磁盘/计算 | 否 | 是，远端下载模型 | 使用 key，不返回 | 适合 `tenant_admin/ops` 但需二次验证和资源配额 |
| `POST /api/channel/ollama/pull/stream` | `AdminAuth` | 是，占用远端资源 | 否 | 是，远端下载模型 | 使用 key，不返回 | 适合 `tenant_admin/ops`，需二次验证、长连接保护 |
| `DELETE /api/channel/ollama/delete` | `AdminAuth` | 否 | 否 | 是，远端删除模型 | 使用 key，不返回 | 适合 `tenant_admin`，必须二次验证和审计 |

### C. 必须保持 root 的接口

这些接口当前应继续保持 root，不建议在 7.8 中下放。

| 接口 | 当前权限 | 消耗上游额度 | 写本地数据库 | 写远端服务 | 暴露 credential | 必须 root 原因 |
| --- | --- | --- | --- | --- | --- | --- |
| `POST /api/channel/:id/key` | `RootAuth` + `SecureVerificationRequired` | 否 | 否，仅 RecordLog | 否 | 是，返回完整 channel key | 明文 credential reveal，必须 root |
| `POST /api/channel/fix` | `RootAuth` | 否 | 是，调用 `model.FixAbility()` 全局修复 abilities | 否 | 否 | 全局能力修复，可能跨租户改写 |
| `POST /api/channel/fetch_models` | `RootAuth` | 是，按外部 channel type/baseURL/key 拉取模型 | 否 | 否 | 请求体包含 key | 全局工具型 fetch，绕过 channel 归属，必须 root |
| `POST /api/models/sync_upstream` | `RootAuth` | 是，读取 upstream/source | 是，写模型 catalog | 否 | 取决于同步源配置 | 模型目录全局写能力 |
| `GET /api/models/sync_upstream/preview` | `AdminAuth` | 可能，取决于实现 | 否或临时读取 | 否 | 否 | 虽然是 preview，但属于模型目录同步链路；7.8 前建议保持现状 |
| `GET /api/ratio_sync/channels` | `RootAuth` | 否 | 否 | 否 | 可能暴露可同步渠道元信息 | pricing/ratio 全局能力 |
| `POST /api/ratio_sync/fetch` | `RootAuth` | 是，调用上游价格接口 | 是，更新 ratio/pricing 配置 | 否 | 可能使用 channel key | 影响全局计费基础 |

### D. 暂不应开放的接口

这些接口即使未来可能开放，也不应在下一轮直接迁移，原因是 credential、远端状态、批量高风险或语义尚不清晰。

| 接口 | 原因 |
| --- | --- |
| `POST /api/channel/codex/oauth/complete`（未绑定 channel） | 成功后直接返回完整 OAuth key，等同 credential reveal；不应给 tenant_admin/ops 常规权限 |
| `POST /api/channel/multi_key/manage` 的 key mutation | 直接改变 key 可用性，且 key preview 暴露部分 credential；建议拆分只读状态和 mutation 后再迁移 |
| `POST /api/channel/copy/:id` | 默认复制完整 key 到新 channel，credential 扩散风险高 |
| `DELETE /api/channel/disabled`、`POST /api/channel/batch` | 批量删除，恢复成本高；需要 dry-run、二次确认、审计日志 |
| `POST /api/channel/upstream_updates/apply_all` | 批量改变模型可用性和 abilities；需要任务预览、确认和回滚策略 |
| `POST /api/channel/ollama/pull/stream` | 长连接、远端资源占用、前端断连处理和任务并发保护不足时风险高 |
| `DELETE /api/channel/ollama/delete` | 直接删除远端模型，属于远端 destructive operation |
| `POST /api/channel/:id/key` | 完整 credential reveal，必须继续 root |
| `POST /api/channel/fix` | 全局 DB 修复，必须 root |
| `POST /api/channel/fetch_models` | 请求体 key + 全局工具能力，必须 root |

## 3. 重点接口副作用说明

### Test

- `GET /api/channel/test/:id`：调用真实 channel 测试请求，可能产生 provider 计费；更新 `response_time`；失败时相关链路可能自动禁用/启用 channel。
- `GET /api/channel/test`：后台批量测试 scoped channels；同样可能产生 provider 计费和本地状态变更；当前有全局互斥锁。

### Balance

- `GET /api/channel/update_balance/:id`：调用 provider 余额接口，更新本地 balance。
- `GET /api/channel/update_balance`：批量余额查询，跳过 disabled/multi-key；余额小于等于 0 时可能禁用 channel。

### Upstream Updates

- `detect`/`detect_all`：调用 upstream 模型列表，写入检测结果、last check time；如果配置启用 auto-add，可能直接更新 channel models 和 abilities。
- `apply`/`apply_all`：不一定再次调用 upstream，但会应用 pending add/remove，写 channel models/settings/abilities/cache。

### Codex OAuth

- `start`：写 session，生成 authorize URL。
- `complete`：交换 OAuth code，产生 access/refresh token；未绑定 channel 时返回完整 key；绑定 channel 时写入 channel key。
- `refresh`/`usage`：使用 refresh token 或 access token；可能更新 channel key。

### Ollama

- `version`：读远端版本。
- `pull`/`pull/stream`：写远端服务，下载模型，消耗带宽、磁盘和计算资源。
- `delete`：写远端服务，删除模型，属于破坏性操作。

### Channel CRUD / Batch / Tag / Copy / Multi-key

- create/update/delete/batch/tag 都写本地数据库并刷新 cache。
- update/copy/multi-key 会处理 credential 或 key 状态。
- copy 默认复制 key，应视为 credential 扩散。

## 4. Iteration 7.8 推荐开发清单

建议 7.8 只迁移 tenant scoped、已有 controller 校验基础较好的接口，并同步补测试。推荐顺序：

1. 迁移低风险单渠道 read/execute：
   - `GET /api/channel/fetch_models/:id` -> `tenant_admin/ops`
   - `GET /api/channel/ollama/version/:id` -> `tenant_admin/ops`
2. 迁移 test/balance，并加测试确认 tenant boundary：
   - `GET /api/channel/test/:id` -> `tenant_admin/ops`
   - `GET /api/channel/test` -> `tenant_admin/ops`
   - `GET /api/channel/update_balance/:id` -> `tenant_admin/ops/finance`
   - `GET /api/channel/update_balance` -> `tenant_admin/ops/finance`
3. 迁移 upstream detect：
   - `POST /api/channel/upstream_updates/detect` -> `tenant_admin/ops`
   - `POST /api/channel/upstream_updates/detect_all` -> `tenant_admin/ops`
   - 明确文案：detect 会写本地检测状态，不是纯 read。
4. 暂缓 apply/apply_all 到后续轮次：
   - 先补 preview/dry-run 和审计日志，再开放。
5. 暂缓 Codex OAuth、multi-key、copy、Ollama pull/delete：
   - 先实现二次验证、审计日志、限频、任务锁、credential 操作记录。
6. 保持 root：
   - `POST /api/channel/:id/key`
   - `POST /api/channel/fix`
   - `POST /api/channel/fetch_models`
   - ratio sync、模型全局 sync。

## 5. 7.8 测试建议

至少补充以下测试：

- tenant_admin/ops 可访问本 tenant 的 `test/:id`、`fetch_models/:id`、`ollama/version/:id`。
- tenant_admin/ops 拒绝访问其他 tenant channel。
- finance 只允许 balance，不允许 test/upstream detect。
- user、organization_admin 拒绝所有 channel ops 执行类接口。
- root 继续可访问全局 root-only 能力。
- `detect_all`、`test`、`update_balance` 的批量路径只处理当前 tenant channel。

## 6. 当前验收命令

本轮只新增文档，预期验收命令：

```bash
go test ./common ./model ./controller ./service ./router ./middleware
```

如果默认 Go build cache 只读失败，使用：

```bash
GOCACHE=/tmp/quinta-go-build-cache go test ./common ./model ./controller ./service ./router ./middleware
```
