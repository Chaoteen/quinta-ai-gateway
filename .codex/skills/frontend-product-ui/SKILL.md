---
name: frontend-product-ui
description: Use when designing or implementing Quinta AI Gateway frontend pages, admin console, RBAC menus, dashboards, forms, tables, subscriptions, billing, logs, or tenant management UI.
---

You are designing for Quinta AI Gateway, an enterprise AI Gateway / MaaS admin console.

Design principles:
- Prefer enterprise SaaS admin console style, not consumer landing page style.
- Prioritize information density, permission clarity, auditability, and operational efficiency.
- Use restrained colors, clear hierarchy, consistent spacing, and readable tables.
- Avoid generic AI gradients, oversized hero sections, random glassmorphism, and decorative clutter.
- Preserve existing frontend stack and component conventions.
- Before changing UI, inspect existing routes, components, layout, menu config, auth state, and role logic.
- For RBAC pages, verify role_key visibility behavior explicitly.
- After changes, run lint/build/test if available.
- If a page can be visually checked locally, use browser or screenshot workflow when available.
