# Subscription RBAC Matrix

This document defines the Iteration 10.1A subscription permission model. It covers Subscription Plan Management, User Subscription Management, and Admin Console visibility only. It does not define quota engine, billing, payment, voucher, invoice, or revenue-share permissions.

## Current Backend Gates

- Plan read: `GET /api/subscription/admin/plans` uses the billing read role gate and is available to root, tenant_admin, organization_admin, finance, and auditor.
- Plan create/update/enable/disable: `POST /api/subscription/admin/plans`, `PUT /api/subscription/admin/plans/:id`, and `PATCH /api/subscription/admin/plans/:id` are root-only.
- User subscription read: list APIs are available to root, tenant_admin, organization_admin, finance, and auditor, with ownership scoping for non-root callers.
- User subscription writes: assign, renew, suspend, and cancel are available to root and tenant_admin. Non-root writes are tenant scoped.
- User role: no admin subscription access.
- Ownership field visibility: root-only in admin user subscription DTO responses.

## Formal Matrix

| Role | Admin Console visibility | Plan read | Plan create/update/enable/disable | User Subscription read | User Subscription assign | User Subscription renew | User Subscription suspend | User Subscription cancel | Ownership field visibility |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| root | Visible | All plans | Yes | All subscriptions | Yes | Yes | Yes | Yes | Visible |
| tenant_admin | Visible | Read-only | No | Own tenant | Own tenant | Own tenant | Own tenant | Own tenant | Hidden |
| organization_admin | Recommended hidden or read-only | Current code: read-only | No | Current code: own organization scope | No | No | No | No | Hidden |
| finance | Visible read-only | Read-only | No | Scoped read-only | No | No | No | No | Hidden |
| auditor | Visible read-only | Read-only | No | Scoped read-only | No | No | No | No | Hidden |
| user | Hidden | No | No | No | No | No | No | No | Hidden |

## Role Notes

### root

Root owns global Subscription Plan Management. Root can create, update, enable, and disable plans. Root can read and mutate user subscriptions across all ownership scopes. Root may see ownership fields in admin user subscription DTOs.

### tenant_admin

Tenant admins may manage user subscriptions inside their own tenant. They may read plans because assignment needs a plan selector, but they must not mutate plan definitions. Tenant admins do not receive ownership fields in DTO responses.

### organization_admin

Current code allows organization_admin read access to the subscription admin list APIs and Admin Console subscription page. The formal 10.1A recommendation is to keep organization_admin hidden or read-only, and not grant write permission in this phase.

If product policy chooses "hidden", a later implementation commit should remove Admin Console visibility and backend read access for organization_admin. If product policy chooses "read-only", the current backend direction is compatible.

### finance

Finance should be read-only. It may inspect plan and user subscription data within its access scope, but cannot create plans or mutate user subscription lifecycle.

### auditor

Auditor should be read-only. It may inspect plan and user subscription data within its access scope, but cannot create plans or mutate user subscription lifecycle.

### user

Regular users can use self and public subscription APIs, but must not enter backend subscription management or call admin subscription APIs.

## Ownership Scope

Non-root admin reads are ownership scoped. Current user subscription list APIs apply access scope filters to `user_subscriptions`, and user-specific reads validate that the target user is inside the caller's access scope.

Non-root writes are restricted to tenant_admin. Tenant_admin writes validate that the target user or subscription belongs to the tenant_admin's tenant scope.

## Button-Level Admin Console Policy

- Plan create button: root only.
- Plan edit button: root only.
- Plan enable/disable action: root only.
- User subscription assign action: root and tenant_admin only.
- User subscription renew action: root and tenant_admin only.
- User subscription suspend action: root and tenant_admin only.
- User subscription cancel action: root and tenant_admin only.
- Read-only roles should see table data without mutation buttons.
- user should not see the Admin Console subscription entry.

## Known Current-Code Differences From Strict Formal Policy

- Plan read is not root-only in the current backend; only plan mutation is root-only. This is intentional for tenant_admin assignment and read-only admin roles.
- organization_admin currently has read-only backend and frontend visibility for subscription management. The formal recommendation is "hidden or read-only"; if "hidden" is selected, code needs a follow-up tightening commit.
- Admin user subscription DTO ownership fields are root-only today. tenant_admin, organization_admin, finance, and auditor do not receive ownership fields.
