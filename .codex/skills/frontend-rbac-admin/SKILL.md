---
name: frontend-rbac-admin
description: Use when implementing or reviewing frontend RBAC, admin menus, route guards, page permissions, button permissions, role_key logic, or Admin Console access control in Quinta AI Gateway.
---

You are working on Quinta AI Gateway Frontend RBAC and Admin Console.

Supported role_key values:

- root
- tenant_admin
- organization_admin
- finance
- ops
- auditor
- user

RBAC principles:

- Menu visibility must be driven by role_key.
- Route access must not be weaker than menu visibility.
- Hidden menu items must not remain directly accessible through URL navigation.
- Action buttons must follow the same permission boundary as the page.
- Root may see all admin features.
- Non-root users must only see and access their allowed scope.
- Do not add temporary bypasses for convenience.
- Do not hardcode user IDs to simulate permissions.
- Do not weaken tenant or ownership checks while implementing frontend access control.

Expected workflow:

1. Inspect existing auth state and current user model.
2. Locate where role_key is loaded, stored, and consumed.
3. Locate admin menu config and route definitions.
4. Implement role_key based visibility.
5. Add or update tests when available.
6. Verify root, tenant_admin, organization_admin, finance, ops, auditor, and user behavior.
7. Report changed files, RBAC impact, and remaining risks.
