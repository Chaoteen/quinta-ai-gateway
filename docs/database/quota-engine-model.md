# Quota Engine Database Model

This document proposes the Iteration 10.2 Quota Engine Foundation database model. It is a design document, not an implemented migration.

10.2 does not introduce real billing, payment, invoice, voucher, revenue-share, wallet deduction, relay integration, or `subscription_orders` expansion.

## Authoritative Entitlement Sources

Quota Engine must use Alpha subscription snapshot fields as the authoritative entitlement sources:

- `user_subscriptions.token_quota_snapshot` is the Alpha token entitlement source.
- `user_subscriptions.request_quota_snapshot` is the Alpha request entitlement source.
- `user_subscriptions.model_quota_snapshot` is the Alpha model entitlement source.

Legacy fields remain compatibility-only:

- `user_subscriptions.amount_total`
- `user_subscriptions.amount_used`

These legacy fields must not become the formal Quota Engine contract.

## Proposed Table: quota_usage_records

`quota_usage_records` is an append-oriented audit table for quota usage state transitions. It is not a payment table and does not deduct wallet balance.

### Field Design

Identity:

- `id`

Ownership fields:

- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`

References:

- `user_id`
- `user_subscription_id`
- `request_id`
- `reservation_id`

Usage context:

- `model_name`
- `quota_dimension`
- `token_delta`
- `request_delta`

Status:

- `status`

Timestamps:

- `occurred_at`
- `created_at`
- `updated_at`

Metadata:

- `metadata`

### Suggested Types

- `id`: integer primary key.
- ownership fields: integer, indexed.
- `user_id`: integer, indexed.
- `user_subscription_id`: integer, indexed.
- `request_id`: varchar, indexed.
- `reservation_id`: varchar, indexed.
- `model_name`: varchar or text, indexed if supported by the database.
- `quota_dimension`: varchar, examples: `token`, `request`, `model`, `reset`.
- `token_delta`: bigint, default `0`.
- `request_delta`: bigint, default `0`.
- `status`: varchar, examples: `reserved`, `committed`, `rolled_back`, `reset`.
- `occurred_at`: bigint.
- `created_at`: bigint.
- `updated_at`: bigint.
- `metadata`: text containing JSON.

Use `TEXT` for JSON-like metadata for SQLite, MySQL, and PostgreSQL compatibility.

### Status Semantics

- `reserved`: quota capacity was reserved.
- `committed`: actual usage was committed.
- `rolled_back`: reserved quota was released or negated.
- `reset`: quota window reset marker.

### Reset Record Design

Reset should be auditable and idempotent.

Recommended reset record fields:

- `quota_dimension = "reset"`
- `status = "reset"`
- `user_subscription_id`
- `reservation_id = ""`
- `request_id = "reset:<subscription_id>:<reset_window>"`
- `token_delta = 0`
- `request_delta = 0`
- `metadata` includes previous used values, reset timestamp, and next reset timestamp.

One reset window should produce at most one reset usage record per subscription.

## Proposed Table: quota_reservations

`quota_reservations` stores idempotent reservation state before usage is committed or rolled back.

### Field Design

Identity:

- `id`
- `reservation_id`

Ownership fields:

- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`

References:

- `user_id`
- `user_subscription_id`
- `request_id`

Reservation context:

- `model_name`
- `token_reserved`
- `request_reserved`

Status:

- `status`

Timestamps:

- `expires_at`
- `committed_at`
- `rolled_back_at`
- `created_at`
- `updated_at`

Metadata:

- `metadata`

### Suggested Types

- `id`: integer primary key.
- `reservation_id`: varchar, unique.
- ownership fields: integer, indexed.
- `user_id`: integer, indexed.
- `user_subscription_id`: integer, indexed.
- `request_id`: varchar, unique or indexed depending on retry semantics.
- `model_name`: varchar or text.
- `token_reserved`: bigint, default `0`.
- `request_reserved`: bigint, default `0`.
- `status`: varchar, examples: `active`, `committed`, `rolled_back`, `expired`.
- `expires_at`: bigint, indexed.
- `committed_at`: bigint.
- `rolled_back_at`: bigint.
- `created_at`: bigint.
- `updated_at`: bigint.
- `metadata`: text containing JSON.

### Reservation Status Semantics

- `active`: reservation can be committed or rolled back.
- `committed`: usage has been finalized.
- `rolled_back`: reservation has been released.
- `expired`: reservation timed out without commit.

## Relationship With user_subscriptions

`user_subscriptions` remains the entitlement and lifecycle source.

Quota Engine should select active subscriptions with:

- matching `user_id`.
- active lifecycle.
- valid `start_time` / `end_time`.
- matching ownership scope.

Quota Engine reads entitlement from:

- `token_quota_snapshot`
- `request_quota_snapshot`
- `model_quota_snapshot`

Quota Engine reads reset schedule from:

- `last_reset_time`
- `next_reset_time`

Quota Engine should copy ownership fields from the selected `user_subscriptions` row into both proposed tables.

`amount_total` and `amount_used` are legacy compatibility fields. If a fallback is needed for old data, it must be explicit and tested. New Alpha quota decisions should not use those fields as the primary source.

## Relationship With SubscriptionPreConsumeRecord

`SubscriptionPreConsumeRecord` is the current legacy idempotency table for subscription pre-consume behavior. It is tied to `PreConsumeUserSubscription`, `RefundSubscriptionPreConsume`, and `amount_used` mutation.

10.2 Quota Engine Foundation should not delete or replace it.

Recommended boundary:

- keep `SubscriptionPreConsumeRecord` for existing legacy billing/session compatibility.
- introduce `quota_reservations` for formal Quota Engine reservation state.
- introduce `quota_usage_records` for formal Quota Engine usage audit state.
- do not mix statuses between the two systems.

Later iterations may adapt `SubscriptionPreConsumeRecord` behavior behind Quota Engine interfaces, but 10.2 should avoid a behavioral migration.

## Relationship With SubscriptionOrder

`subscription_orders` remains a payment/order compatibility table.

Quota Engine must not:

- add fields to `subscription_orders`.
- read payment provider state from `subscription_orders`.
- write order callback state.
- infer quota usage from payment order state.

Subscription orders may create subscriptions through existing flows, but Quota Engine starts from `user_subscriptions`, not from orders.

## Idempotency Field Design

Reservations:

- `quota_reservations.reservation_id` should be unique.
- `quota_reservations.request_id` should be unique if one request maps to one reservation.
- duplicate `ReserveQuota` should return the existing reservation.

Usage records:

- committed usage should be idempotent by `reservation_id`.
- rollback usage should be idempotent by `reservation_id`.
- reset usage should be idempotent by generated reset `request_id`.

Recommended uniqueness:

- unique index on `quota_reservations.reservation_id`.
- unique index on `quota_reservations.request_id` if request-level uniqueness is required.
- composite unique index on `quota_usage_records(reservation_id, status)` for `committed` and `rolled_back` semantics may need database-specific care; if partial indexes are avoided, enforce this in service logic and tests.
- unique reset `request_id` for reset records.

## Index Recommendations

For `quota_reservations`:

- unique: `reservation_id`.
- optional unique or non-unique: `request_id`.
- index: `user_id`.
- index: `user_subscription_id`.
- index: `tenant_id`, `organization_id`, `department_id`, `distribution_channel_id`.
- index: `status`.
- index: `expires_at`.

For `quota_usage_records`:

- index: `request_id`.
- index: `reservation_id`.
- index: `user_id`.
- index: `user_subscription_id`.
- index: `model_name`.
- index: `quota_dimension`.
- index: `status`.
- index: `occurred_at`.
- index: `tenant_id`, `organization_id`, `department_id`, `distribution_channel_id`.

Avoid database-specific partial indexes in the initial migration. Use GORM-compatible indexes and service-level idempotency checks for SQLite, MySQL, and PostgreSQL compatibility.

## Migration Risks

- `amount_total` and `amount_used` already have production meaning in legacy subscription deduction. Do not reinterpret them silently.
- `SubscriptionPreConsumeRecord` already has idempotency behavior. Do not migrate existing records into new tables without a dedicated migration plan.
- `model_quota_snapshot` currently stores text. Parser validation must be explicit and fail safely.
- Metadata should be `TEXT`, not database-specific JSON column types.
- Cross-database behavior for `FOR UPDATE` differs. Service tests should validate idempotency and final state, not rely on lock syntax alone.
- Reset behavior currently updates `amount_used`. Formal reset records should be added without breaking the existing reset task.
- Future relay integration must be feature-gated or staged so existing traffic is not suddenly denied or double-counted.

## 10.2 No Real Billing Constraint

The proposed tables are quota foundation tables only.

They must not:

- change `users.quota`.
- create top-up records.
- create payment records.
- create invoice records.
- create revenue-share records.
- mutate `subscription_orders`.
- trigger wallet deduction.

They may later serve as inputs to Usage Metering and Billing, but in 10.2 they are not billing records.
