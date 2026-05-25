# Iteration 2: Ownership Closure

## 本轮修复内容

本轮仅关闭新建对象的 ownership 写入缺口，不调整已有 tenant scope 查询规则、relay channel selection、权限模型、计费、缓存命名空间或迁移框架。

### 1. 管理员创建 User

- 在 `controller/user.go` 的 `CreateUser` 中，普通 admin 创建用户时调用 `model.ApplyOwnershipFromContext`，以当前请求 Context 的 `tenant_id`、`organization_id`、`department_id`、`distribution_channel_id` 覆盖新用户归属。原因是此前 `cleanUser` 只复制基础字段，请求发起者所在 tenant 未被写入，新用户会依赖数据库默认值落入 tenant 1。
- 在 `model/ownership.go` 中为 `OwnershipSnapshot.ApplyTo` 增加 `*User` 支持，使 User 使用与现有 Log、TopUp 等对象相同的归属写入路径，避免在 controller 中重复赋值。
- `controller/user.go` 中 root admin 创建用户时可从请求体指定 ownership，并通过 `OwnershipSnapshot.ApplyTo` 写入；`tenant_id=0` 仍会按 `model/ownership.go` 的既有 `NormalizeOwnership` 规则兼容为默认 tenant 1。

### 2. 创建 Token

- 在 `model/token.go` 的 `Token.Insert` 中，写库前必须通过 `model.RequiredOwnershipByUserId` 获取所属 User 的 ownership，并覆盖 Token 自身字段。原因是 Token 的归属应由 owner User 决定，不能依赖客户端提交字段，也不能依赖数据库默认 tenant。
- 在 `model/ownership.go` 中新增 `RequiredOwnershipByUserId`，当 user id 非法、用户读取失败或用户不存在时返回错误，而不是静默生成 tenant 1 归属；同时为 `OwnershipSnapshot.ApplyTo` 增加 `*Token` 支持。
- `controller/token.go` 的普通 token 创建最终调用 `Token.Insert`；`controller/user.go` 的注册默认 token 创建也调用同一方法。因此两个创建入口都会继承所属 User 的 ownership，无需在各 controller 分别维护归属赋值。

### 3. 创建 Channel

- 在 `controller/channel.go` 的 `AddChannel` 中，普通 admin 完成参数校验后、生成待插入 Channel 列表前，调用 `model.ApplyOwnershipFromContext` 覆盖请求体中的 ownership。原因是 `batch`、`single`、`multi_to_single` 三种创建模式都由该模板 Channel 派生，入口处覆盖可以确保所有新增记录归属于当前 admin tenant。
- 在 `model/ownership.go` 中为 `OwnershipSnapshot.ApplyTo` 增加 `*Channel` 支持。
- root admin 不执行上述覆盖，保留现有的平台级显式指定 ownership 能力。

### 4. Ability 继承 Channel Ownership

- 在 `model/ability.go` 的 `Channel.AddAbilities` 与 `Channel.UpdateAbilities` 中，每条新建 `Ability` 均调用 `OwnershipFromChannel(channel).ApplyTo(&ability)`。
- 在 `model/ownership.go` 中新增 `OwnershipFromChannel`，并为 `OwnershipSnapshot.ApplyTo` 增加 `*Ability` 支持。
- 这样处理的原因是 Ability 是 Channel 可用性索引的一部分，若它没有复制 Channel ownership，则即使 Channel 写入正确，后续按 Ability 检索时仍可能出现归属不一致的数据。

### 5. Task 继承 RelayInfo Ownership

- 在 `relay/common/relay_info.go` 中新增 `OwnershipResolved`，由 `genBaseRelayInfo` 根据 Gin Context 中是否实际携带非零 `tenant_id` 设置。既有兼容行为仍保留：`model/user_cache.go` 的 `UserBase.WriteContext` 会先将旧用户数据中的 `tenant_id=0` 写为 tenant 1；现在 Task 链路能够区分正常认证后的归属与未取得 tenant Context 的异常情况。
- 在 `model/task.go` 中，`InitTask` 改为返回错误，并将 `RelayInfo` 中的四个 ownership 字段复制到新 Task；当 `RelayInfo` 缺失或 `OwnershipResolved` 为 false 时直接返回错误。`Task.Insert` 另行拒绝写入 `tenant_id=0` 的新任务，形成写库前防线。
- 在 `controller/relay.go` 中，Task 初始化失败会通过 `common.SysError` 明确记录错误并跳过 Task 插入。原因是任务请求在该位置已经完成上游提交及现有结算流程，本轮只处理归属写入，未改变计费或响应行为。

## 修改文件

| 文件路径 | 修改目的 |
| --- | --- |
| `model/ownership.go` | 扩展统一 ownership 应用对象，新增 User ownership 强校验和 Channel ownership 来源 helper |
| `controller/user.go` | 管理员新建 User 写入 Context ownership，root 保留指定能力 |
| `model/token.go` | Token 插入前强制继承所属 User ownership |
| `controller/channel.go` | 普通 admin 新建 Channel 强制继承 Context ownership |
| `model/ability.go` | AddAbilities、UpdateAbilities 创建 Ability 时继承 Channel ownership |
| `relay/common/relay_info.go` | 标识 relay ownership 是否真实来自 Gin Context |
| `model/task.go` | Task 初始化复制 RelayInfo ownership，缺失归属时拒绝创建/插入 |
| `controller/relay.go` | 对 Task ownership 初始化失败记录明确错误 |
| `docs/iteration_02_ownership_closure.md` | 记录本轮范围、写入链路和遗留风险 |

## Ownership 写入链路

| 对象 | 来源 | 写入路径 | 结果 |
| --- | --- | --- | --- |
| `User` | 普通 admin 的 Gin Context | `controller/user.go` -> `model.ApplyOwnershipFromContext` -> `model/ownership.go` | 请求体无法覆盖普通 admin 所属 tenant |
| `User` | root 请求体 | `controller/user.go` -> `model.OwnershipSnapshot.ApplyTo` | root 可创建指定归属用户 |
| `Token` | 所属 `User` | `controller/token.go` 或 `controller/user.go` -> `model/token.go: Token.Insert` -> `model/ownership.go: RequiredOwnershipByUserId` | token 与 owner user 四个 ownership 字段一致 |
| `Channel` | 普通 admin 的 Gin Context | `controller/channel.go` -> `model.ApplyOwnershipFromContext` -> `model.BatchInsertChannels` | 批量和单条新建均继承当前 tenant |
| `Ability` | 所属 `Channel` | `model/channel.go` -> `model/ability.go: AddAbilities/UpdateAbilities` -> `model.OwnershipFromChannel` | ability 与 channel ownership 一致 |
| `Task` | relay Gin Context 生成的 `RelayInfo` | `relay/common/relay_info.go` -> `controller/relay.go` -> `model/task.go: InitTask/Insert` | ownership 缺失时不插入任务，并记录错误 |

## Root Admin 保留能力

- `controller/user.go`：root admin 创建 User 时允许由请求体提供 ownership，普通 admin 的同类字段会被 Context ownership 覆盖。
- `controller/channel.go`：root admin 创建 Channel 时保持原有请求体归属行为；普通 admin 的同类字段会被 Context ownership 覆盖。
- `model/token.go`：Token 始终绑定 owner User ownership，不为 root 提供脱离 owner 的覆盖路径，因为 Token 的归属语义是用户凭证归属，不是平台资源分配。
- `model/tenant_scope.go` 未修改；本轮不改变 root 跨 tenant 查询能力或现有 scope 行为。

## 仍未解决的问题

- `controller/relay.go`、relay 下游选择链路：本轮仅保证新 Task 的归属写入，未验证或收紧 relay 选取 Channel/Ability 时是否始终按 tenant 隔离。这属于下一轮风险重点。
- `relay/common/relay_info.go`：当前仅为 Task 插入增加 `OwnershipResolved` 防线；所有 relay 请求是否都已由中间件稳定写入 tenant Context，需要人工复核。
- `controller/user.go` 的公开注册沿用既有默认 tenant 策略；`controller/oauth.go` 等非后台管理员用户创建路径的完整 tenant 分配策略需要人工复核。它们是否需要按邀请关系或企业入口分配 tenant，需要产品规则后再处理。
- `model/ownership.go` 中 `OwnershipByUserId` 仍保留读取失败时回退默认 tenant 的既有兼容行为，供本轮范围外的旧写入路径使用；这些路径是否需要逐一改为强校验，需要人工复核。
- 旧数据中已经形成的 ownership 不一致记录未在本轮修复或回填；本轮只阻止相关新建路径继续产生同类错误。

## 下一轮 Relay Tenant Isolation 风险

- 应审计 relay channel selection 是否在从 `Ability`、`Channel` 查询候选和重试候选时始终使用请求的 tenant scope，重点路径包括 `model/ability.go`、`model/channel.go`、`middleware` 与 `relay` 入口。
- 应检查 token 消耗、quota 更新与日志写入是否均以请求 user/token 的 ownership 为来源，并对跨 tenant 的 channel 使用场景定义平台 root 的允许边界；相关改动涉及计费时必须独立评估，本轮未触碰。
- 应为 relay 入口缺少 tenant Context 的情况建立可观测告警和请求级拒绝策略；当前 `model/task.go` 仅保护 Task 落库，不等价于整个 relay 主链路已隔离。

## 验证

- 已执行 `go test ./model ./controller ./service`，测试通过。
