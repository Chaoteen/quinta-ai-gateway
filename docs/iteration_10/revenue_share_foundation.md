# Iteration 10.6 Revenue Share Foundation

## Goal

Iteration 10.6 establishes the Revenue Share Foundation for Quinta AI Gateway as a MaaS, Agent Marketplace, Skill Marketplace, and distributor platform.

This iteration creates auditable revenue split rules and revenue share records. It does not perform payouts, payment collection, invoice generation, wallet mutation, or settlement.

## Channel Levels

Revenue share supports a fixed three-level channel hierarchy:

| Level | Code | Meaning |
| --- | --- | --- |
| 0 | PLATFORM | Quinta AI platform |
| 1 | MASTER_DISTRIBUTOR | Master distributor |
| 2 | DISTRIBUTOR | Distributor |

Rules:

- Infinite channel trees are not supported.
- Revenue share supports at most platform, master distributor, and distributor.
- The platform always exists.
- A master distributor can have multiple distributors.
- A distributor cannot develop child distributors.
- Revenue share records must carry `tenant_id`.
- Revenue share records must be linkable to `BillingRecord` or future order/payment facts.

## Models

### RevenueShareRule

`RevenueShareRule` defines a tenant-scoped revenue split rule.

Fields:

- `id`
- `tenant_id`
- `distribution_channel_id`
- `rule_name`
- `rule_scope`
- `provider_name`
- `model_name`
- `product_type`
- `platform_share_rate`
- `master_distributor_share_rate`
- `distributor_share_rate`
- `effective_from`
- `effective_to`
- `enabled`
- `created_at`
- `updated_at`

Supported `rule_scope` values:

- `global`
- `provider`
- `model`
- `subscription`
- `agent`
- `skill`
- `video`

The three share rates must sum to `100`. If no master distributor or distributor participates, the corresponding rate should be `0`.

### RevenueShareRecord

`RevenueShareRecord` records how one revenue fact is split.

Fields:

- `id`
- `tenant_id`
- `billing_record_id`
- `source_type`
- `source_id`
- `distribution_channel_id`
- `master_distributor_id`
- `distributor_id`
- `gross_amount`
- `platform_amount`
- `master_distributor_amount`
- `distributor_amount`
- `currency`
- `share_rule_id`
- `status`
- `calculated_at`
- `settled_at`
- `created_at`
- `updated_at`

Supported `source_type` values:

- `billing`
- `payment`
- `order`
- `subscription`
- `manual`

Supported `status` values:

- `pending`
- `calculated`
- `locked`
- `settled`
- `cancelled`

## APIs

Revenue Share Rule management:

- `POST /api/revenue-share/rules`
- `GET /api/revenue-share/rules`
- `PUT /api/revenue-share/rules/:id`
- `POST /api/revenue-share/rules/:id/enable`
- `POST /api/revenue-share/rules/:id/disable`

Revenue Share Record query:

- `GET /api/revenue-share/records`

Record query filters:

- `tenant_id`
- `distribution_channel_id`
- `status`
- `source_type`
- `start_time`
- `end_time`
- `page`
- `limit`

## Calculation

`CalculateRevenueShare` takes:

- `tenant_id`
- `billing_record_id` or `source_id`
- `gross_amount`
- `distribution_channel_id`
- `provider_name`
- `model_name`
- `product_type`

It returns:

- `platform_amount`
- `master_distributor_amount`
- `distributor_amount`
- `matched_rule_id`

Rule matching priority:

1. `model`
2. `provider`
3. product type scopes such as `subscription`, `agent`, `skill`, or `video`
4. `global`

If no rule matches, the default split is:

- platform: `100%`
- master distributor: `0%`
- distributor: `0%`

For billing source records, `billing_record_id` is idempotent. Calling record creation repeatedly for the same billing record returns the existing `RevenueShareRecord`.

## Permissions

| Role | Rule Read | Rule Write | Record Read |
| --- | --- | --- | --- |
| root | All tenants | All tenants | All tenants |
| tenant_admin | Own tenant | Own tenant | Own tenant |
| finance | Own tenant | No | Own tenant |
| auditor | Own tenant | No | Own tenant |
| organization_admin | No | No | No |
| user | No | No | No |

Service-layer tenant checks remain in place in addition to route-level RBAC.

## Billing Runtime Relationship

Revenue Share Foundation reads `BillingRecord` as the current billing fact source. It can create `RevenueShareRecord` from a `BillingRecord`, but it does not:

- mutate `BillingRecord`
- call `BillingSession`
- call `FundingSource`
- update wallet balances
- update `User.Quota`
- update `Token.RemainQuota`
- update `UserSubscription.AmountUsed`

This keeps 10.6 in shadow/foundation mode.

## Future Reuse

Payment and order integrations can reuse the same `RevenueShareRecord` table by setting `source_type` and `source_id`.

Settlement can later lock calculated records, group them by tenant/channel/currency, and mark them `settled` after confirmed payout.

Finance Console can read `BillingRecord` for billing facts and `RevenueShareRecord` for split facts. Revenue Share should remain append-only for settled records in future iterations.
