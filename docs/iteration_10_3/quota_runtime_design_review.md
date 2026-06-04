# Iteration 10.3A Quota Runtime Design Review

Branch: `feat/iteration-10-3-quota-runtime-foundation`

This review audits the current quota-related architecture after Iteration 10.2. It does not introduce runtime behavior. The goal is to clarify quota ownership before implementing `CheckQuota`, `ReserveQuota`, `CommitUsage`, and rollback/reset behavior.

## Scope

Reviewed areas:

- `model/`
- `service/`
- `controller/`
- `docs/architecture/`
- `docs/database/`

Primary objects reviewed:

- `QuotaReservation`
- `QuotaUsageRecord`
- `QuotaEngine`
- `UserSubscription`
- `SubscriptionPlan`
- `BillingSession`
- `FundingSource`
- `SubscriptionPreConsumeRecord`
- `User.Quota`
- `Token.RemainQuota`

## Current Quota Systems

The repository currently has multiple quota-like systems. They are not equivalent and should not be merged without an explicit migration plan.

| System | Data source | Current purpose | Still called | Future status |
| --- | --- | --- | --- | --- |
| Wallet/user quota | `users.quota`, `users.used_quota`, `users.request_count` | Wallet-style balance and user usage accounting | Yes. Used by wallet pre-consume, post-consume, `BillingSession`, and user usage updates. | Keep as Billing Runtime / wallet balance. Do not use as subscription quota truth. |
| Token quota | `tokens.remain_quota`, `tokens.used_quota`, `tokens.unlimited_quota`, token model limits | Per-token spend cap and token-level request guard | Yes. Used by token pre-consume/post-consume and billing settlement/refund paths. | Keep as token guard. Do not use as subscription entitlement truth. |
| Legacy subscription deduction | `user_subscriptions.amount_total`, `user_subscriptions.amount_used`, `subscription_pre_consume_records` | Existing subscription pre-consume and settlement compatibility path | Yes. Used by `SubscriptionFunding`, `BillingSession`, and subscription pre/post consume helpers. | Keep temporarily as legacy compatibility. Migrate behind Quota Runtime later. |
| Alpha subscription entitlements | `user_subscriptions.token_quota_snapshot`, `request_quota_snapshot`, `model_quota_snapshot` | Alpha subscription entitlement snapshot copied from plan | Partially. Exposed by DTOs and parsed by foundation code, but not enforced by relay/runtime yet. | Keep as authoritative subscription entitlement source. |
| Quota Runtime foundation | `quota_reservations`, `quota_usage_records` | Future reservation, usage event, rollback, reset, and audit state | No relay/runtime calls yet. Foundation models and service interface exist. | Keep as Quota Runtime state source once implemented. |
| Usage/log accounting | request logs, `QuotaData`, usage counters | Reporting and historical usage | Yes. Used by consume logging and user usage reporting. | Keep as Usage Metering / observability, not as quota entitlement truth. |

### Key Observation

There are currently at least five quota-like domains:

1. Wallet balance.
2. Token-level quota.
3. Legacy subscription amount deduction.
4. Alpha subscription entitlement snapshots.
5. New Quota Runtime reservation and usage records.

They overlap in vocabulary but represent different responsibilities. Iteration 10.3 should avoid binding Quota Runtime directly to wallet deduction, token deduction, or `amount_used` mutation.

## Single Source Of Truth Analysis

`QuotaReservation` and `QuotaUsageRecord` should not become the only quota source in isolation. They do not define which plan a user owns, whether the subscription is active, what reset window applies, or what the entitlement limits are.

Recommended source-of-truth split:

| Responsibility | Source of truth |
| --- | --- |
| Subscription entitlement | `user_subscriptions.token_quota_snapshot`, `request_quota_snapshot`, `model_quota_snapshot` |
| Subscription lifecycle eligibility | `user_subscriptions.lifecycle_status`, `start_time`, `end_time`, `next_reset_time`, ownership fields |
| Runtime reservation state | `quota_reservations` |
| Runtime committed usage and reset audit | `quota_usage_records` |
| Wallet balance | `users.quota` and billing wallet records |
| Token guard | `tokens.remain_quota` and token settings |
| Monetary settlement | `BillingSession`, `FundingSource`, future billing ledger |

Therefore, Quota Runtime can become the single source of truth for subscription quota runtime state, but not for subscription entitlement or billing balance.

## Subscription, Quota, Billing, And Funding Boundaries

### Subscription

Subscription should own:

- Plan assignment.
- Subscription lifecycle.
- Tenant, organization, department, and distribution channel ownership.
- Entitlement snapshots copied from plan.
- Reset schedule fields such as `last_reset_time` and `next_reset_time`.

Subscription should not own:

- Per-request reservation state.
- Relay request settlement.
- Wallet deduction.
- Payment/order callback behavior.

### Quota Runtime

Quota Runtime should own:

- Model quota parsing and enforcement.
- Quota checks against active subscription snapshots.
- Request reservation creation.
- Reservation rollback.
- Usage commit records.
- Reset records.
- Idempotency and tenant-safe lookup rules for quota operations.

Quota Runtime should not own:

- Wallet balance mutation.
- Token balance mutation.
- Invoice/payment/revenue-share behavior.
- Public purchase flow.

### Billing Runtime

Billing Runtime should own:

- Funding source selection.
- Wallet deduction and refund.
- Legacy subscription funding compatibility while migration is incomplete.
- Future monetary settlement and ledger integration.

Billing Runtime should not decide subscription model entitlement or subscription runtime availability once Quota Runtime is implemented.

### FundingSource

`FundingSource` is currently an execution adapter for where consumption comes from. It should remain an adapter layer, not the authority for entitlement. In the future, a subscription funding source may delegate to Quota Runtime instead of directly mutating `amount_used`.

## Relay Integration Points

Quota Runtime should be connected to relay only after the runtime service has deterministic service-level tests. The future relay sequence should be:

1. Resolve user, token, tenant scope, requested model, and request identity.
2. Call `CheckQuota()` before upstream dispatch.
3. Call `ReserveQuota()` before upstream dispatch when the request is allowed.
4. Call upstream provider.
5. Call `CommitUsage()` after final usage is known.
6. Call `RollbackReservation()` when upstream fails, client aborts, or the request is otherwise not committed.

`ResetQuota()` should not run in the hot relay path. It should be invoked by a scheduled task or lifecycle maintenance path.

Important integration constraint:

- Quota Runtime commit must not also trigger wallet deduction, token deduction, or `amount_used` mutation unless a later billing integration explicitly routes that behavior through one controlled adapter.

## State Machine Review

### Reservation Lifecycle

Current reservation statuses:

- `active`
- `committed`
- `rolled_back`
- `expired`

Recommended transitions:

| From | To | Allowed |
| --- | --- | --- |
| `active` | `committed` | Yes |
| `active` | `rolled_back` | Yes |
| `active` | `expired` | Yes |
| `committed` | any other state | No, except idempotent commit replay |
| `rolled_back` | any other state | No, except idempotent rollback replay |
| `expired` | any other state | No, except idempotent expire replay |

Runtime requirements:

- Terminal states should be immutable.
- Commit and rollback must be idempotent by `reservation_id`.
- Expiration should be deterministic and safe to retry.
- A commit after rollback or expiration should be denied.

No additional reservation state is required for 10.3 foundation. A future `pending` state is unnecessary unless the implementation introduces asynchronous reservation creation.

### Usage Record Lifecycle

Current usage statuses:

- `reserved`
- `committed`
- `rolled_back`
- `reset`

`QuotaUsageRecord` is best treated as an append-only event table rather than a mutable state row. Under that model:

- `reserved` records reservation creation.
- `committed` records final usage.
- `rolled_back` records reservation rollback.
- `reset` records quota reset events.

Potential future additions:

- `expired` for reservation expiration events.
- `adjusted` if post-settlement usage corrections are later required.

These additions are not required before the first runtime implementation, but expiration audit should be considered before relay integration.

## Database Index Recommendations

The 10.2 model foundation already includes core indexes such as `reservation_id`, `request_id`, `user_id`, `user_subscription_id`, `status`, and ownership fields. Runtime implementation should validate and extend indexes based on query shape.

### QuotaReservation

Recommended indexes:

- Unique `reservation_id`.
- `request_id` for request-level lookup.
- Consider unique `request_id` only if the product guarantees one reservation per request.
- Composite ownership scope plus `user_subscription_id` and `status`.
- `user_id`, `status`, `expires_at` for active reservation scans.
- `user_subscription_id`, `status`, `expires_at` for subscription-level reservation aggregation.
- `status`, `expires_at` for expiration jobs.

Avoid database-specific partial indexes until cross-database compatibility is designed for SQLite, MySQL, and PostgreSQL.

### QuotaUsageRecord

Recommended indexes:

- `reservation_id`, `status`.
- `request_id`, `status`.
- `user_subscription_id`, `status`, `occurred_at`.
- `user_id`, `occurred_at`.
- Ownership scope plus `status` and `occurred_at`.
- `quota_dimension`, `occurred_at` for quota dimension aggregation.

If reset records become idempotent by generated request ID, `request_id` uniqueness or a separate idempotency key should be added.

## Problems Found

1. `QuotaReservation.request_id` is indexed but not unique. This is acceptable for foundation, but runtime idempotency must decide whether one request can have multiple reservations.
2. `QuotaUsageRecord` has no unique idempotency constraint. Commit, rollback, and reset replay protection must be handled by service logic or future indexes.
3. Legacy `amount_used` reset still exists and does not emit `QuotaUsageRecord` reset events.
4. `SubscriptionPreConsumeRecord` and `QuotaReservation` can double-reserve the same request if both paths are connected without an adapter boundary.
5. `BillingSession` currently mutates funding source state and token quota. Quota Runtime must not duplicate those mutations.
6. Tenant scope enforcement for quota runtime is not implemented yet. It must use ownership fields copied from `UserSubscription`.
7. Model quota parsing is fail-closed for invalid JSON and empty allowlists, which is appropriate, but the schema is not versioned.
8. Metadata is stored as text without schema. This is acceptable for foundation but should not be used for authoritative quota decisions.
9. Existing wallet/token quota fields use different numeric types and semantics from subscription quota snapshots. They should remain separate systems.

## Recommended 10.3 Route

10.3 should be Quota Runtime Foundation, not relay integration and not billing integration.

Recommended implementation order:

1. Implement `CheckQuota()` using active `UserSubscription` snapshots, lifecycle status, reset window, and ownership scope.
2. Implement `ReserveQuota()` as a transaction that creates `QuotaReservation` and a `reserved` usage event only.
3. Implement `RollbackReservation()` as an idempotent terminal transition plus a `rolled_back` usage event.
4. Implement `CommitUsage()` as an idempotent terminal transition plus a `committed` usage event.
5. Implement `ResetQuota()` as a reset event writer and subscription reset-window coordinator.
6. Add service-level tests for tenant isolation, idempotency, insufficient quota, model denial, request quota, token quota, reset, rollback, and concurrency.

Explicitly defer:

- Relay integration.
- Wallet deduction.
- Token quota mutation.
- `amount_used` migration.
- Billing, invoice, payment, voucher, and revenue-share behavior.

## Next Iteration Recommendation

Recommended next iteration name:

`Iteration 10.3B Quota Runtime Service Implementation`

Recommended scope:

- Service-only implementation of quota runtime behavior.
- No controller.
- No router.
- No relay integration.
- No billing mutation.
- No wallet or token quota mutation.
- Full service tests before any hot-path integration.

The main architectural risk is double accounting. That risk is controlled by keeping Quota Runtime as an auditable reservation and usage event system first, then connecting relay and billing in later iterations through one explicit adapter boundary.

## 10.3B Implementation Notes

Implemented service interfaces:

- `CheckQuota()`
- `ReserveQuota()`
- `CommitUsage()`
- `RollbackReservation()`
- `ResetQuota()`

Runtime behavior added:

- `CheckQuota()` reads active `UserSubscription` rows by `user_id`, lifecycle status, start/end time, and exact ownership scope.
- Model quota enforcement uses `model_quota_snapshot`; empty snapshot remains unrestricted and allowlist snapshots enforce exact model names.
- Token and request quota checks use subscription snapshot limits plus current runtime usage.
- Runtime usage includes committed `QuotaUsageRecord` rows and non-expired active `QuotaReservation` rows in the current reset window.
- `ReserveQuota()` creates one active `QuotaReservation` and one `reserved` `QuotaUsageRecord`.
- Reservation idempotency is enforced by looking up existing `reservation_id` or `request_id` before creating a new row.
- `CommitUsage()` performs `active -> committed`, writes a `committed` usage event, and is idempotent by `reservation_id`.
- `RollbackReservation()` performs `active -> rolled_back`, writes a `rolled_back` usage event, and is idempotent by `reservation_id`.
- `ResetQuota()` writes a `reset` usage event and updates the subscription reset window.

Still not connected:

- No relay hot-path integration.
- No controller or router integration.
- No Admin Console integration.
- No BillingSession integration.
- No FundingSource integration.
- No wallet deduction.
- No token quota deduction.
- No payment, invoice, voucher, billing deduction, or revenue-share behavior.

Why wallet, token quota, and legacy subscription quota are not modified:

- `User.Quota` is wallet/billing runtime state, not subscription runtime entitlement.
- `Token.RemainQuota` is token-level access guard state, not subscription quota entitlement.
- `UserSubscription.amount_used` belongs to the legacy subscription deduction path and is still used by `SubscriptionPreConsumeRecord` and `SubscriptionFunding`.
- Updating any of those fields inside Quota Runtime would create double-accounting risk before relay and billing are connected through an explicit adapter boundary.

Current limitation:

- `QuotaReservation.request_id` is still an indexed field, not a unique constraint. The 10.3B service enforces request idempotency in code, but stronger cross-process concurrency guarantees may require a future migration or a dedicated idempotency key.
