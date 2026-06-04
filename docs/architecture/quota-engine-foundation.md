# Quota Engine Foundation

This document defines the Iteration 10.2 Quota Engine Foundation design. It is an architecture contract for quota decision, reservation, usage recording, reset, and future integration points.

Iteration 10.2 does not connect the quota engine to relay traffic and does not perform real billing deduction.

## 10.2 Goals

- Define a small quota engine boundary that can be tested independently from relay and payment flows.
- Establish tenant-safe quota decision and reservation semantics for subscriptions.
- Make Alpha subscription entitlement fields authoritative for quota decisions:
  - `token_quota_snapshot` is the Alpha token entitlement source.
  - `request_quota_snapshot` is the Alpha request entitlement source.
  - `model_quota_snapshot` is the Alpha model entitlement source.
- Preserve compatibility with existing subscription deduction code without treating it as the formal quota engine.
- Prepare a clean foundation for later Usage Metering, Billing, and Revenue Share work.
- Keep all operations idempotent by request or reservation identifiers.

## 10.2 Non-Goals

- Do not connect quota checks to relay.
- Do not deduct wallet quota.
- Do not modify `users.quota`.
- Do not expand `subscription_orders`.
- Do not implement Billing Deduction.
- Do not implement Payment.
- Do not implement Invoice.
- Do not implement Voucher.
- Do not implement Revenue Share.
- Do not add Admin Console pages.
- Do not replace existing `BillingSession` or subscription deduction paths in this iteration.

## Current Legacy Subscription Deduction

The current system already has subscription-related deduction primitives:

- `BillingSession`
- `FundingSource`
- `SubscriptionFunding`
- `SubscriptionPreConsumeRecord`
- `PreConsumeUserSubscription`
- `PostConsumeUserSubscriptionDelta`
- `user_subscriptions.amount_total`
- `user_subscriptions.amount_used`

These are useful compatibility assets, but they are not the formal Quota Engine. They are coupled to existing billing and relay settlement behavior and primarily operate on legacy `amount_total` / `amount_used` quota units.

The Quota Engine Foundation should first model and test quota decisions separately. Later iterations may adapt the existing deduction path behind the quota engine boundary after relay and billing integration rules are explicit.

## Boundary With Subscription

Subscription owns entitlement creation and lifecycle:

- Plan assignment.
- Plan renewal.
- Lifecycle state: active, expired, suspended, cancelled.
- Snapshot creation:
  - `token_quota_snapshot`
  - `request_quota_snapshot`
  - `model_quota_snapshot`
- Reset schedule:
  - `last_reset_time`
  - `next_reset_time`

Quota Engine owns quota interpretation and state transitions:

- Whether a subscription can satisfy a quota request.
- Reservation of token/request/model quota capacity.
- Commit or rollback of reserved usage.
- Reset usage state for quota windows.
- Usage records for later metering and audit.

Subscription remains the source of entitlement. Quota Engine should not create, renew, cancel, suspend, or expire subscriptions.

## Boundary With Usage Metering

Usage Metering will eventually provide actual usage measurements from relay and asynchronous tasks:

- Actual token counts.
- Request counts.
- Model name and endpoint classification.
- Cache/read/write token dimensions if needed later.

Quota Engine 10.2 should accept usage-shaped inputs but should not collect relay usage itself. It can define `CommitUsage` inputs that future Usage Metering can call.

## Boundary With Billing

Billing will eventually decide monetary settlement, invoice implications, wallet deductions, or revenue attribution. Quota Engine does not do that in 10.2.

Quota Engine may emit usage records with enough metadata for future billing, but records created by the foundation are not invoices, payments, ledger entries, or wallet deductions.

## Minimal Interface Design

### CheckQuota

Checks whether a user has an eligible active subscription that can cover the requested dimensions.

Input:

- `user_id`
- ownership scope or concrete ownership snapshot
- `model_name`
- `token_amount`
- `request_amount`
- optional `user_subscription_id`
- `request_id`

Output: `QuotaDecision`.

No state mutation.

### ReserveQuota

Creates an idempotent reservation for a quota request. Reservation should lock capacity for later commit or rollback without touching wallet quota.

Input:

- all `CheckQuota` inputs
- `reservation_id` or idempotent `request_id`
- `expires_at`

Output: `QuotaReservation`.

State mutation:

- insert or return an existing `quota_reservations` record.
- optionally insert `quota_usage_records` with `status = reserved`.

No relay integration in 10.2.

### CommitUsage

Commits actual usage against an existing reservation.

Input:

- `reservation_id`
- `request_id`
- actual `token_amount`
- actual `request_amount`
- `model_name`
- metadata

Output:

- committed usage record.
- updated reservation status.

State mutation:

- mark reservation committed.
- create committed usage record.

No wallet deduction. No billing deduction.

### RollbackReservation

Releases a reservation that should not be charged against quota usage.

Input:

- `reservation_id` or `request_id`
- reason metadata

Output:

- rollback status.

State mutation:

- mark reservation rolled back.
- create rollback usage record if needed for audit.

Rollback must be idempotent.

### ResetQuota

Resets usage state for a subscription quota window.

Input:

- `user_subscription_id`
- reset timestamp
- reason

Output:

- reset result.

State mutation:

- create reset usage records.
- advance/reset usage accounting state.
- update `last_reset_time` and `next_reset_time` through subscription-compatible reset logic.

10.2 should preserve the existing reset task behavior and may only define the formal model for later implementation.

## Quota Decision Model

`QuotaDecision` should be a pure service result:

- `allowed`: boolean
- `reason`: machine-readable reason
- `message`: human-readable message
- `user_id`
- `user_subscription_id`
- `plan_id`
- `model_name`
- `token_limit`
- `token_used`
- `token_remaining`
- `request_limit`
- `request_used`
- `request_remaining`
- `model_allowed`
- `reset_at`
- `ownership`

Reason examples:

- `allowed`
- `no_active_subscription`
- `subscription_inactive`
- `token_quota_exceeded`
- `request_quota_exceeded`
- `model_not_allowed`
- `tenant_scope_denied`
- `reservation_conflict`

## Reservation Model

Reservation status:

- `active`
- `committed`
- `rolled_back`
- `expired`

Reservation dimensions:

- `token_reserved`
- `request_reserved`
- `model_name`

Reservation must carry ownership fields:

- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`

Reservation must be idempotent by one of:

- unique `reservation_id`
- unique `request_id`

For 10.2, prefer both. `reservation_id` is the internal primary idempotency key; `request_id` links to relay/task requests.

## Usage Record Model

Usage records are append-oriented audit records for quota dimensions. They are not payment records.

Record dimensions:

- `token_delta`
- `request_delta`
- `model_name`
- `quota_dimension`

Record statuses:

- `reserved`
- `committed`
- `rolled_back`
- `reset`

Usage records should preserve enough metadata for later Usage Metering and Billing, but must not perform billing behavior in 10.2.

## Model Quota Parsing Rules

`model_quota_snapshot` is the Alpha model entitlement source.

Recommended 10.2 parser rules:

- Empty string means unrestricted model access for that subscription.
- Whitespace-only string means unrestricted model access.
- Invalid JSON should fail closed for writes/reservations and return a clear validation error.
- Initial supported shape should be allowlist based:

```json
{
  "allow": ["gpt-4o", "gpt-4o-mini"]
}
```

- Optional future shape can include per-model token caps, but 10.2 should not implement dynamic pricing semantics:

```json
{
  "allow": ["gpt-4o"],
  "limits": {
    "gpt-4o": {
      "token_quota": 100000
    }
  }
}
```

Do not reuse billing expression syntax for `model_quota_snapshot` in 10.2.

## Tenant Safety Principles

- Every quota record must carry ownership fields.
- Non-root reads and operations must be scoped by ownership.
- No quota operation may infer tenant access from `user_id` alone after request authentication.
- Root bypass must be explicit.
- Reservations and usage records must copy ownership from the selected `user_subscriptions` row.
- Quota operations must reject mismatched user/subscription ownership.
- Cache keys, if introduced later, must include tenant scope when the cached data differs by tenant.

## Idempotency Principles

- `ReserveQuota` is idempotent by `reservation_id` and should also tolerate duplicate `request_id`.
- `CommitUsage` is idempotent for a committed reservation.
- `RollbackReservation` is idempotent for a rolled back reservation.
- `ResetQuota` should create one reset record per subscription reset window.
- Duplicate relay retries must not double reserve, double commit, or double rollback.

## Compatibility With amount_total / amount_used

`amount_total` and `amount_used` are legacy compatibility fields.

10.2 should treat them as compatibility state only:

- `amount_total` may mirror old `total_amount`.
- `amount_used` may reflect legacy aggregate subscription usage.
- New quota decisions should prefer:
  - `token_quota_snapshot`
  - `request_quota_snapshot`
  - `model_quota_snapshot`
- If compatibility requires fallback, it must be explicit and tested:
  - `token_quota_snapshot == 0` may fall back to `amount_total` only for legacy subscriptions.
  - fallback must not redefine `amount_total` as the Alpha entitlement source.

## Why 10.2 Does Not Connect Relay

Relay integration changes runtime behavior and can cause production traffic to be allowed, denied, reserved, committed, or rolled back differently.

10.2 should first prove:

- quota decisions are correct.
- reservation idempotency is correct.
- tenant isolation is correct.
- reset behavior is correct.
- model quota parsing is correct.

Connecting relay should be a later iteration after the service is independently tested.

## Why 10.2 Does Not Do Billing

Billing requires monetary semantics, ledger behavior, invoice/reconciliation readiness, and revenue attribution. Quota Engine Foundation only defines entitlement usage state.

In 10.2:

- no wallet deduction.
- no payment mutation.
- no invoice mutation.
- no revenue-share record.
- no billing ledger entry.
- no expansion of `subscription_orders`.

Quota usage records may later become inputs to billing, but they are not billing records in this iteration.
