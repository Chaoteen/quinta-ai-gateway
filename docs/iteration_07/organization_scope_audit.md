# Iteration 7.1 Organization Scope Audit

## 1. Audit Conclusion

Current code already has the structural foundation for organization and department ownership:

- `Organization` and `Department` models exist.
- `User`, `Channel`, `Ability`, `Token`, `Log`, `TopUp`, `Redemption`, `Task`, `Midjourney`, `SubscriptionOrder`, `UserSubscription`, and `SubscriptionPreConsumeRecord` carry `tenant_id`, `organization_id`, `department_id`, and `distribution_channel_id`.
- `UserBase` cache writes organization and department context into Gin context.
- `OwnershipSnapshot` can copy ownership from context, user, channel, subscription order, or subscription.
- root-created user ownership is validated by `ValidateOwnershipHierarchy`.

However, authorization scope is still tenant-only. `TenantScope` only contains `TenantId` and `IsRoot`, and `TenantScope.Apply()` only filters `tenant_id`. There is no effective `OrganizationScope`, no department tree scope, and no `organization_admin` boundary enforcement.

Conclusion: do not migrate additional AdminAuth routes to `organization_admin` until Iteration 7.2 introduces organization-aware scope helpers and route tests. The next implementation should build scope primitives first, then migrate only tenant-scoped read routes whose model queries can be proven organization-scoped.

## 2. Current Code Status

### Organization Model

`model/organization.go` defines:

| Field | Status |
|---|---|
| `id` | primary identifier |
| `tenant_id` | indexed, default `1` |
| `name` | indexed |
| `status` | indexed |
| `distribution_channel_id` | indexed |
| `remark` | metadata |
| timestamps / soft delete | present |

Gaps:

- No organization CRUD controller/router found in current audit target.
- No uniqueness rule for `(tenant_id, name)`.
- No organization-specific scope helper.
- No explicit `organization_admin` membership/assignment model beyond `users.role_key`.

### Department Model

`model/department.go` defines:

| Field | Status |
|---|---|
| `id` | primary identifier |
| `tenant_id` | indexed, default `1` |
| `organization_id` | indexed |
| `parent_id` | indexed |
| `name` | indexed |
| `status` | indexed |
| `distribution_channel_id` | indexed |
| `remark` | metadata |
| timestamps / soft delete | present |

Gaps:

- No department CRUD controller/router found in current audit target.
- No helper to expand a department subtree.
- No validation that `parent_id` belongs to the same tenant and organization.
- No department scope semantics for department admins or inherited visibility.

### User Visibility

Current user ownership fields:

- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`
- `role`
- `role_key`

Current user visibility checks:

- `GetAllUsers` and `SearchUsers` call `model.TenantScopeFromContext(c)`.
- `GetUser`, `UpdateUser`, `DeleteUser`, `ManageUser`, binding cleanup, passkey/2FA admin flows use `requireUserTenantAccess`.
- `requireUserTenantAccess` only validates `tenant_id`.
- legacy role int comparisons still govern same-level/higher-level user management.

Implication: an `organization_admin` would currently see or mutate all tenant users if simply admitted through `RoleAuth`, because the model and controller checks do not filter by organization.

### Resource Ownership

Ownership propagation exists for resource creation:

- `ApplyOwnershipFromContext`
- `ApplyOwnershipFromUser`
- `OwnershipFromChannel`
- `ownershipFromSubscriptionOrder`
- `ownershipFromUserSubscription`

Resources with organization/department fields include:

- users
- tokens
- channels
- abilities
- logs
- topups
- redemptions
- tasks
- midjourney
- subscription orders
- user subscriptions
- subscription pre-consume records

Current read/write scope enforcement remains tenant-only through:

- `TenantScopeFromContext`
- `RelayTenantScopeFromContext`
- `TenantScope.Apply`
- `AllowsTenant`
- `require*TenantAccess`

### organization_admin Boundary

`common.RoleKeyOrganizationAdmin` exists and is normalized by RBAC helpers.

Missing pieces:

- No `IsOrganizationAdminRole` helper.
- No `OrganizationScope` or expanded `AccessScope`.
- No `RoleAuth` route currently grants `organization_admin`.
- No scope check that binds `organization_admin` to `context.organization_id`.
- No fail-closed rule for `organization_admin` with `organization_id = 0`.
- No tests proving tenant admin can see tenant-wide data while organization admin only sees organization data.

### Department Scope

Department IDs are stored and propagated, but no query-level department visibility exists.

Open design question:

- Should department scope mean exact department only?
- Should it include child departments through `parent_id`?
- Should organization_admin see all departments under the organization?
- Should department-level role be introduced later, or should department scope only restrict ordinary users/resources?

Recommendation: Iteration 7.2 should model department scope as data structure only, but avoid route adoption until subtree behavior is explicit.

## 3. Risk Points

| Risk | Severity | Detail |
|---|---:|---|
| Tenant-only scope | High | Current `TenantScope` would overexpose tenant-wide data to `organization_admin`. |
| User management relies on role int | High | User admin logic still compares legacy `role` integers, not `role_key` hierarchy. |
| No organization scope fail-closed | High | A user with `role_key = organization_admin` and `organization_id = 0` has no safe boundary semantics. |
| Department tree undefined | Medium | `parent_id` exists, but no subtree query or validation helper exists. |
| Global catalog/config read ambiguity | Medium | models/group/prefill_group expose global config not tied to organization. |
| Billing mutation boundary | High | subscription/topup/redemption writes need billing operation policy before role expansion. |
| External channel operations | High | channel test, balance, upstream fetch, Codex/Ollama operations may call external systems or mutate state. |
| Mixed role systems | Medium | RBAC `role_key` and legacy `role int` coexist; controllers still use `role int` checks. |

## 4. Route Classification Table

### Candidate After Organization Scope

These routes may be considered for `organization_admin` only after model queries and controller checks support organization filtering.

| Route | Current Auth | Future Role | Required Scope Work | Notes |
|---|---|---|---|---|
| `GET /api/user/` | `AdminAuth` | `organization_admin` | users filtered by organization | Read-only but sensitive. |
| `GET /api/user/search` | `AdminAuth` | `organization_admin` | users filtered by organization | Must prevent tenant-wide search. |
| `GET /api/user/:id` | `AdminAuth` | `organization_admin` | exact user organization access | Existing check is tenant-only. |
| `GET /api/user/:id/oauth/bindings` | `AdminAuth` | maybe `organization_admin` | target user organization access | Sensitive identity bindings; consider auditor-only read later. |
| `GET /api/log/` | `RoleAuth(tenant_admin, finance, auditor)` | maybe `organization_admin`, `auditor` | logs filtered by organization | Already tenant scoped; extend before granting. |
| `GET /api/log/stat` | `RoleAuth(tenant_admin, finance, auditor)` | maybe `organization_admin`, `auditor` | stats filtered by organization | Needs aggregate scope tests. |
| `GET /api/user/topup` | `RoleAuth(tenant_admin, finance, auditor)` | maybe `organization_admin`, `finance`, `auditor` | topups filtered by organization | Billing visibility policy needed. |
| `GET /api/redemption/` | `RoleAuth(tenant_admin, finance, auditor)` | maybe `organization_admin`, `finance`, `auditor` | redemptions filtered by organization | Read-only; creation/mutation stays admin. |
| `GET /api/redemption/search` | `RoleAuth(tenant_admin, finance, auditor)` | maybe `organization_admin`, `finance`, `auditor` | redemptions filtered by organization | Same as above. |
| `GET /api/redemption/:id` | `RoleAuth(tenant_admin, finance, auditor)` | maybe `organization_admin`, `finance`, `auditor` | exact redemption organization access | Existing check is tenant-only. |
| `GET /api/task/` | `RoleAuth(tenant_admin, ops, auditor)` | maybe `organization_admin`, `ops`, `auditor` | tasks filtered by organization | Async task ownership exists. |
| `GET /api/mj/` | `RoleAuth(tenant_admin, ops, auditor)` | maybe `organization_admin`, `ops`, `auditor` | midjourney filtered by organization | Ownership fields exist. |
| `GET /api/channel/` | `RoleAuth(tenant_admin, ops, auditor)` | maybe `organization_admin`, `ops`, `auditor` | channels filtered by organization | Must include ability/channel joins. |
| `GET /api/channel/search` | `RoleAuth(tenant_admin, ops, auditor)` | maybe `organization_admin`, `ops`, `auditor` | channels filtered by organization | Must handle tag mode. |
| `GET /api/channel/:id` | `RoleAuth(tenant_admin, ops, auditor)` | maybe `organization_admin`, `ops`, `auditor` | exact channel organization access | Existing check is tenant-only. |
| `GET /api/channel/models_enabled` | `RoleAuth(tenant_admin, ops, auditor)` | maybe `organization_admin`, `ops`, `auditor` | ability/model filtered by organization | Uses enabled models by scope. |
| `GET /api/channel/tag/models` | `RoleAuth(tenant_admin, ops, auditor)` | maybe `organization_admin`, `ops`, `auditor` | channels by tag filtered by organization | Existing scope only tenant. |
| `GET /api/subscription/admin/users/:id/subscriptions` | `RoleAuth(tenant_admin, finance, auditor)` | maybe `organization_admin`, `finance`, `auditor` | target user and subscription org access | Existing ensure only tenant. |

### Keep Tenant Admin / Admin Until Later

| Route | Current Auth | Reason |
|---|---|---|
| `GET /api/models/` | `AdminAuth` | Global catalog enriched with bound channels/groups/quota types. |
| `GET /api/models/search` | `AdminAuth` | Same as full model read. |
| `GET /api/models/:id` | `AdminAuth` | Same as full model read. |
| `GET /api/group/` | `AdminAuth` | Global group config view. |
| `GET /api/prefill_group/` | `AdminAuth` | Global template/config content. |
| `GET /api/channel/models` | `AdminAuth` | Global channel model view; scope semantics unclear. |
| `GET /api/log/search` | `AdminAuth` | Deprecated endpoint returning failure; no migration value. |

### Keep Admin/Root Due To Writes Or Sensitive Effects

| Route Family | Current Auth | Reason |
|---|---|---|
| user create/update/delete/manage | `AdminAuth` | Role assignment, status changes, destructive actions. |
| user binding/passkey/2FA admin deletes | `AdminAuth` / nested `RootAuth` | Sensitive identity/security operations. |
| subscription bind/create/invalidate/delete | `AdminAuth` | Billing mutation. |
| subscription plan POST/PUT/PATCH | `RootAuth` | Billing config mutation. |
| redemption POST/PUT/DELETE | `AdminAuth` | Billing/redeemable value mutation. |
| channel create/update/delete/tag/copy/multi_key | `AdminAuth` | Channel config mutation and credential adjacency. |
| channel key reveal | `RootAuth` | Credential access. |
| channel test/balance/fetch/upstream/Codex/Ollama | `AdminAuth` | External requests and/or state mutation. |
| model/vendor writes | `RootAuth` | Global catalog mutation. |
| sync upstream | `AdminAuth` preview / `RootAuth` write | External request and global mutation. |
| option/performance/custom OAuth/deployments/data root routes | `RootAuth` | System/global operations. |

## 5. Iteration 7.2 Recommended Development List

1. Define scope model.
   - Add an access scope type that includes tenant, organization, department, root flag, and role key.
   - Keep `TenantScope` stable for existing tenant-only code, or extend with compatibility wrappers.

2. Add role helper.
   - `IsOrganizationAdminRole(roleKey string)`.
   - `IsScopedAdminRole(roleKey string)` if useful.

3. Implement organization-aware scope extraction.
   - `AccessScopeFromContext(c)`.
   - Fail closed when `organization_admin` has no `organization_id`.
   - root remains unrestricted.
   - tenant_admin remains tenant-wide.
   - organization_admin is restricted to one tenant and one organization.

4. Add query helpers.
   - `ApplyOwnershipScope(db, tableAlias)` filtering by `tenant_id`, then `organization_id`, optionally `department_id`.
   - `AllowsOwnership(tenantId, organizationId, departmentId int)`.
   - Resource-specific helpers for users, channels, logs, topups, redemptions, subscriptions, tasks, and midjourney.

5. Add hierarchy validation.
   - Validate department `parent_id` belongs to same tenant and organization.
   - Add helper to validate organization/department active status if needed.

6. Update read models incrementally.
   - Start with users list/search/detail under organization scope.
   - Then logs/topups/redemptions/task/mj.
   - Then channel read and subscription user read.

7. Keep mutations out of Iteration 7.2.
   - Do not migrate billing writes, user role changes, channel writes, external upstream calls, or key access.

8. Document department semantics before route adoption.
   - Exact department vs subtree visibility must be decided before any department-scoped admin routes.

## 6. Test Recommendations

### Scope Helper Tests

- root can access all tenants and organizations.
- tenant_admin can access all resources in own tenant but not another tenant.
- organization_admin can access own organization only.
- organization_admin with `organization_id = 0` is denied.
- organization_admin cannot access another organization in same tenant.
- organization_admin cannot access another tenant.
- ordinary user is not treated as admin scope.

### Model Query Tests

For each scoped model helper:

- tenant 1 / organization 1 data is returned to org 1 admin.
- tenant 1 / organization 2 data is hidden from org 1 admin.
- tenant 2 data is hidden from tenant 1 org admin.
- tenant_admin sees both organizations within tenant 1.
- root sees all.

Priority model areas:

- `User`
- `Log`
- `TopUp`
- `Redemption`
- `Task`
- `Midjourney`
- `Channel`
- `UserSubscription`

### Router Tests

Add router tests only after model scope is implemented:

- organization_admin can access migrated read routes within own org.
- organization_admin receives failure for same-tenant different-org ids.
- organization_admin receives failure for tenant_admin-only routes.
- tenant_admin behavior remains unchanged.
- root behavior remains unchanged.
- finance/ops/auditor behavior remains unchanged for routes migrated in Iteration 6.

### Regression Tests

- RootAuth routes still reject tenant_admin, organization_admin, finance, ops, auditor.
- AdminAuth legacy behavior remains unchanged until route migration.
- Existing RoleAuth routes do not accidentally admit organization_admin before scope support.
- Relay scope remains tenant-safe and does not silently become organization-scoped without design.
