---
name: maas-billing
description: Use when designing or implementing MaaS billing, wallet, token packages, top-up, redemption, subscriptions, settlement, ledger, invoice, pricing, or usage accounting in Quinta AI Gateway.
---

You are working on Quinta AI Gateway MaaS billing and commercial foundation.

Business model:

- Token packages
- Model usage billing
- Skill products
- Agent products
- Top-up
- Redemption
- Subscription
- Wallet balance
- Usage logs
- Settlement
- Invoice and reconciliation foundation

Financial engineering principles:

- Do not treat financial records as ordinary CRUD.
- Prefer append-only ledger style records for balance-affecting operations.
- Every balance change must be traceable.
- Every commercial record must carry tenant ownership metadata.
- Avoid destructive updates to financial history.
- Avoid silent balance correction.
- Use explicit transaction boundaries where money, quota, or balance changes.
- Separate display balance from auditable ledger records when possible.
- Preserve enough metadata for reconciliation and later invoice support.

Expected workflow:

1. Identify the commercial object being changed.
2. Identify whether the change affects balance, quota, subscription state, or reconciliation.
3. Preserve tenant_id and ownership metadata.
4. Prefer auditable records over overwriting historical values.
5. Check concurrency and repeated-submit risks.
6. Run available tests.
7. Report billing impact, accounting assumptions, and risks.
