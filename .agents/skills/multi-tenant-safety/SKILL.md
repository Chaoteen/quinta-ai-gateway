---
name: multi-tenant-safety
description: Use when modifying tenant, organization, department, user, token, channel, subscription, billing, log, or relay logic in Quinta AI Gateway.
---

You are working on Quinta AI Gateway multi-tenant safety.

Core entities:

- tenant_id
- organization_id
- department_id
- distribution_channel_id
- user_id
- role_key
- ownership scope

Safety rules:

- Never remove tenant isolation.
- Never bypass ownership validation.
- Never introduce tenant_id fallback behavior unless explicitly required and documented.
- Never allow non-root users to query or mutate data outside their tenant scope.
- Root-only behavior must remain explicitly guarded.
- Cache keys must be tenant-safe when data differs by tenant.
- Channel selection and relay paths must preserve tenant isolation.
- Logs, billing records, top-ups, redemptions, subscriptions, and tasks must retain ownership metadata.

Expected workflow:

1. Identify all query paths affected by the change.
2. Check whether tenant_id and ownership fields are preserved.
3. Check whether root bypass is explicit and limited.
4. Check whether cache keys include tenant scope where required.
5. Check whether create/update paths write ownership metadata.
6. Run available backend tests.
7. Report tenant-safety impact and any unsafe assumptions.
