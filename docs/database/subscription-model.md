# Subscription Database Model

This document records the Iteration 10.1A subscription database model. It documents current model shape and compatibility boundaries only. It does not introduce quota engine, billing, payment, voucher, invoice, or revenue-share semantics.

## subscription_plans

`subscription_plans` is the global subscription catalog. It has no tenant ownership fields in the current model.

### Field Groups

- Identity:
  - `id`
  - `code`
- Alpha display and admin fields:
  - `name`
  - `description`
  - `monthly_price`
  - `yearly_price`
  - `token_quota`
  - `request_quota`
  - `model_quota`
  - `status`
  - `created_at`
  - `updated_at`
- Legacy display and purchase compatibility fields:
  - `title`
  - `subtitle`
  - `price_amount`
  - `currency`
  - `duration_unit`
  - `duration_value`
  - `custom_seconds`
  - `enabled`
  - `sort_order`
  - `max_purchase_per_user`
  - `upgrade_group`
  - `total_amount`
  - `quota_reset_period`
  - `quota_reset_custom_seconds`
- Provider compatibility fields:
  - `stripe_price_id`
  - `creem_product_id`

### Alpha Authoritative Fields

The Alpha admin contract treats these fields as authoritative:

- `code`
- `name`
- `description`
- `monthly_price`
- `yearly_price`
- `token_quota`
- `request_quota`
- `model_quota`
- `status`

`created_at` and `updated_at` are returned by the admin DTO but are system-managed timestamps.

### Legacy Compatibility Fields

The model still keeps legacy fields for public purchase compatibility and old internal flows:

- `title` / `subtitle`
- `price_amount` / `currency`
- `duration_unit` / `duration_value` / `custom_seconds`
- `enabled`
- `sort_order`
- `max_purchase_per_user`
- `upgrade_group`
- `total_amount`
- `quota_reset_period` / `quota_reset_custom_seconds`

The admin Alpha DTO does not expose these fields, but the public purchase DTO still does.

### Runtime Lifecycle Fields

Plan availability currently has two related fields:

- `status`: Alpha admin status, normalized to `enabled` or `disabled`.
- `enabled`: legacy public purchase availability flag.

`NormalizeAlphaFields` keeps `status` and `enabled` aligned. Admin status patch updates both fields.

### Payment And Order Fields

`subscription_plans` does not store orders or callbacks. It does keep provider catalog identifiers for public purchase compatibility:

- `stripe_price_id`
- `creem_product_id`

These fields are not part of the admin Alpha plan DTO.

### Ownership Fields

None. Plans are currently global.

### Quota Snapshot Fields

None on the plan table. Plan quota source fields are copied into `user_subscriptions` when a subscription is created:

- `token_quota`
- `request_quota`
- `model_quota`

### Migration Risks

- `monthly_price` vs `price_amount`: Alpha admin uses `monthly_price`; legacy public purchase uses `price_amount`. Normalization copies values when one side is empty.
- `token_quota` vs `total_amount`: Alpha quota is `token_quota`; legacy purchase/amount compatibility is `total_amount`. Normalization copies values when one side is empty.
- `status` vs `enabled`: Alpha status and legacy availability must stay aligned.
- Duration and reset fields are hidden from admin DTO but are still runtime inputs for subscription creation, renewal, end-time calculation, and reset-time calculation.
- Provider IDs remain stored for purchase compatibility; they must not leak into admin plan DTOs.

## subscription_orders

`subscription_orders` is the current payment/order compatibility table. Iteration 10.1A does not expand it.

### Field Groups

- Identity:
  - `id`
- Ownership:
  - `tenant_id`
  - `organization_id`
  - `department_id`
  - `distribution_channel_id`
- User and plan references:
  - `user_id`
  - `plan_id`
- Order and payment fields:
  - `money`
  - `trade_no`
  - `payment_method`
  - `payment_provider`
  - `status`
  - `create_time`
  - `complete_time`
  - `provider_payload`

### Alpha Authoritative Fields

There are no Alpha admin-authoritative management fields in `subscription_orders`. It is not used as an admin subscription management DTO source in the Alpha boundary.

### Legacy Compatibility Fields

All order/payment fields are compatibility fields for existing purchase and callback flows:

- `money`
- `trade_no`
- `payment_method`
- `payment_provider`
- `status`
- `create_time`
- `complete_time`
- `provider_payload`

### Runtime Lifecycle Fields

`status` tracks order state, not user subscription lifecycle. It must not be confused with `subscription_plans.status` or `user_subscriptions.lifecycle_status`.

### Payment And Order Fields

The whole table is payment/order oriented. Current callback completion creates a `user_subscriptions` snapshot from the referenced plan after validating order state.

### Ownership Fields

Ownership fields are present and are copied from the user or from the completed order into the created user subscription:

- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`

### Quota Snapshot Fields

None. Quota snapshots are created in `user_subscriptions` from the selected plan during completion.

### Migration Risks

- The table is still required by existing purchase and callback compatibility paths.
- Do not reuse order `status` as subscription lifecycle status.
- 10.1A should not add new billing, payment, invoice, voucher, or revenue-share meaning to this table.

## user_subscriptions

`user_subscriptions` stores concrete subscription instances assigned to users.

### Field Groups

- Identity:
  - `id`
- Ownership:
  - `tenant_id`
  - `organization_id`
  - `department_id`
  - `distribution_channel_id`
- User and plan references:
  - `user_id`
  - `plan_id`
- Legacy quota usage fields:
  - `amount_total`
  - `amount_used`
- Runtime lifecycle fields:
  - `start_time`
  - `end_time`
  - `status`
  - `lifecycle_status`
  - `source`
  - `last_reset_time`
  - `next_reset_time`
  - `created_at`
  - `updated_at`
- Quota snapshot fields:
  - `token_quota_snapshot`
  - `request_quota_snapshot`
  - `model_quota_snapshot`
- Legacy group fields:
  - `upgrade_group`
  - `prev_user_group`

### Alpha Authoritative Fields

The current Alpha subscription lifecycle contract is centered on:

- `user_id`
- `plan_id`
- `lifecycle_status`
- `start_time`
- `end_time`
- `token_quota_snapshot`
- `request_quota_snapshot`
- `model_quota_snapshot`
- `next_reset_time`

Admin DTOs add plan display fields (`plan_code`, `plan_name`) by joining back to `subscription_plans`.

### Legacy Compatibility Fields

The following fields remain in the model for existing compatibility but are not exposed by admin user subscription DTOs:

- `amount_total`
- `amount_used`
- `source`
- `upgrade_group`
- `prev_user_group`

Self subscription DTO may expose user-facing derived token quota values:

- `token_quota`
- `token_used`
- `token_remaining`

These are DTO-level display fields and are not a full model exposure.

### Runtime Lifecycle Fields

- `lifecycle_status`: authoritative Alpha lifecycle state.
- `status`: legacy lifecycle state.
- `start_time`: subscription start timestamp.
- `end_time`: subscription end timestamp.
- `last_reset_time`: last quota reset timestamp.
- `next_reset_time`: next quota reset timestamp.

`NormalizeLifecycle` keeps `status` synchronized with `lifecycle_status`.

### Payment And Order Fields

`user_subscriptions` does not store payment callback payloads or provider order identifiers. `source` may indicate legacy creation source such as `order` or `admin`, but it is not exposed by the Alpha admin DTO.

### Ownership Fields

The table stores ownership:

- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`

Root admin DTO responses may include these fields. Non-root admin DTO responses currently omit ownership fields.

### Quota Snapshot Fields

Snapshots are copied from `subscription_plans` when the subscription is created or renewed:

- `token_quota_snapshot` from `subscription_plans.token_quota`
- `request_quota_snapshot` from `subscription_plans.request_quota`
- `model_quota_snapshot` from `subscription_plans.model_quota`

Snapshots preserve the subscribed entitlement even if the plan changes later.

### Migration Risks

- `status` vs `lifecycle_status`: `lifecycle_status` is the Alpha authoritative lifecycle field; `status` is synchronized for compatibility.
- `amount_total` vs `token_quota_snapshot`: `amount_total` is legacy quota compatibility; Alpha user-facing quota should come from token snapshot fields.
- `source` is useful internally but should not be exposed in admin or self DTOs.
- `duration_unit`, `duration_value`, `custom_seconds`, `quota_reset_period`, and `quota_reset_custom_seconds` are hidden from the admin Plan DTO but still used by create/renew and reset-time calculations through the full plan model.
