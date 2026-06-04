# Iteration 10.5A Billing Runtime Architecture Audit

Branch: `feat/iteration-10-5-billing-runtime-foundation`

This audit reviews the current billing and deduction path after Subscription, Quota Runtime, and Usage Metering foundations. It does not change production code.

## Scope

Reviewed modules:

- `service/`
- `model/`
- `relay/`
- `controller/`

Focus objects:

- `BillingSession`
- `FundingSource`
- `SubscriptionPreConsumeRecord`
- `PostTextConsumeQuota()`
- `PostAudioConsumeQuota()`
- `QuotaUsageRecord`
- `QuotaReservation`
- `User`
- `Token`
- `UserSubscription`
- `SubscriptionPlan`

## Current Billing Call Chain

Current synchronous text request chain:

```text
Client request
  -> controller/relay.go
  -> RelayInfo
  -> token estimation
  -> ModelPriceHelper()
  -> PreConsumeBilling()
  -> NewBillingSession()
  -> FundingSource.PreConsume()
  -> relay adaptor DoRequest()
  -> relay adaptor DoResponse()
  -> dto.Usage
  -> UsageMetering dry committed fact
  -> PostTextConsumeQuota()
  -> quota calculation
  -> UpdateUserUsedQuotaAndRequestCount()
  -> UpdateChannelUsedQuota()
  -> SettleBilling()
  -> BillingSession.Settle()
  -> FundingSource.Settle()
  -> User.Quota / Token.RemainQuota / UserSubscription.AmountUsed mutation
  -> RecordConsumeLog()
```

Current audio path is similar after provider response:

```text
dto.Usage
  -> PostAudioConsumeQuota()
  -> audio quota calculation
  -> UpdateUserUsedQuotaAndRequestCount()
  -> UpdateChannelUsedQuota()
  -> SettleBilling()
  -> BillingSession / FundingSource
  -> RecordConsumeLog()
```

Failure path before provider success:

```text
Relay error
  -> deferred Billing.Refund()
  -> FundingSource.Refund()
  -> token quota refund
  -> optional violation fee
```

## Current Runtime Objects

### BillingSession

Current role:

- Runtime-only pre-consume and settlement coordinator.
- Holds one in-memory `FundingSource`.
- Implements `relay/common.BillingSettler`.
- Mutates wallet/subscription funding and token quota during `PreConsume`, `Settle`, `Reserve`, and `Refund`.

Limitations:

- It is not persisted.
- It cannot be queried by finance tools.
- It is not an auditable billing fact.
- If process state is lost, only side effects and auxiliary records remain.

### FundingSource

Current implementations:

- `WalletFunding`
- `SubscriptionFunding`

Current mutations:

- Wallet path mutates `User.Quota`.
- Subscription path mutates `UserSubscription.AmountUsed`.
- Both paths can also affect `Token.RemainQuota` through `BillingSession`.

### SubscriptionPreConsumeRecord

Current role:

- Idempotency record for subscription pre-consume.
- Unique by `request_id`.
- Stores ownership fields.
- Supports `consumed` and `refunded` states.

Limitations:

- It is subscription-specific.
- It is not a generic billing event.
- It does not represent wallet deductions.
- It does not encode price calculation details, usage facts, or finance settlement fields.

### QuotaUsageRecord

Current role:

- Append-oriented quota and usage fact table.
- Stores tenant, org, department, distribution channel, user, subscription, provider, channel, model, token, request, and semantic usage fields.
- 10.4 dry relay integration writes committed usage facts before legacy billing settlement.

Non-role:

- It is not a billing, payment, wallet, or settlement record.
- It should not mutate balances.

### QuotaReservation

Current role:

- Quota Runtime reservation state.
- Tracks active, committed, rolled back, and expired quota reservation states.

Non-role:

- It does not settle money or wallet/subscription balances.
- It should not become a billing ledger.

### User / Token / UserSubscription

Current balance-bearing fields:

- `User.Quota`
- `Token.RemainQuota`
- `UserSubscription.AmountUsed`

Current metrics/display fields:

- `User.UsedQuota`
- `User.RequestCount`
- `Token.UsedQuota`
- `UserSubscription.AmountTotal`

These fields remain runtime balances and legacy compatibility state. They should not be the only auditable source for future billing and finance reporting.

## Funding Source Priority

Recommended future priority policy:

1. Free Tier
2. Channel Sponsored
3. Voucher
4. Subscription
5. Wallet

Reasoning:

- Free Tier should be consumed first when the product explicitly grants no-cost usage.
- Channel Sponsored should apply before user-owned paid balances because it is attribution-driven and may be funded by a distribution channel or commercial partner.
- Voucher should apply before subscription/wallet because it is usually promotional or constrained by rule.
- Subscription should apply before wallet by default for entitlement products, while preserving user preference for `wallet_first` or `subscription_first`.
- Wallet remains the universal fallback because it is the most liquid paid balance.

Current code supports:

- Subscription-first, wallet-first, subscription-only, and wallet-only preferences through `UserSetting.BillingPreference`.
- Wallet funding.
- Subscription funding.

Current code does not support:

- Voucher funding.
- Channel sponsored funding.
- Free tier as a first-class funding source.

## Billing Runtime Boundary

Billing Runtime should own:

- Funding source selection.
- Billing idempotency.
- Price calculation snapshot.
- Pre-consume, settle, refund, and adjustment events.
- Balance-affecting transaction orchestration.
- Append-only billing records.
- Finance and reconciliation source data.

Quota Runtime should own:

- Entitlement checks.
- Reservation state.
- Quota reset windows.
- Request/token/model quota enforcement.
- No wallet or payment mutation.

Usage Metering should own:

- Provider usage normalization.
- `QuotaUsageRecord(status=committed)` usage facts.
- Provider/channel/model usage attribution.
- No settlement or deduction.

Subscription should own:

- Plan assignment.
- Lifecycle state.
- Entitlement snapshots.
- Reset windows.
- Subscription allocation metadata.

FundingSource should become:

- A pluggable settlement executor used by Billing Runtime.
- Not the authoritative historical record.

## Avoiding Duplicate Measurement And Deduction

The critical split should be:

```text
Usage Metering writes what happened.
Billing Runtime writes what was charged.
FundingSource mutates the selected balance.
```

Billing Runtime must not re-normalize provider usage independently when a committed `QuotaUsageRecord` exists. It should read the committed usage fact, calculate billing from that fact, and store a billing record that references it.

`PostTextConsumeQuota()` currently re-calculates usage and calls settlement. This is acceptable for legacy compatibility but should not be the future single source of truth.

## Billing Fact Recommendation

`BillingSession` should not be the only Billing Fact.

Recommended new foundation table:

`billing_records`

Suggested fields:

- `id`
- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`
- `request_id`
- `reservation_id`
- `usage_record_id`
- `billing_record_id`
- `user_id`
- `token_id`
- `user_subscription_id`
- `funding_source`
- `billing_status`
- `billing_phase`
- `model_name`
- `provider_name`
- `channel_id`
- `input_tokens`
- `output_tokens`
- `total_tokens`
- `request_count`
- `quota_charged`
- `pre_consumed_quota`
- `settled_delta`
- `currency`
- `unit_price_snapshot`
- `price_snapshot`
- `settled_at`
- `refunded_at`
- `metadata`
- `created_at`
- `updated_at`

Optional append-only companion:

`billing_events`

Suggested event statuses:

- `pre_consumed`
- `settled`
- `refunded`
- `adjusted`
- `failed`

Recommendation:

- Add `billing_records` first in 10.5B.
- Add `billing_events` only if the implementation needs multi-step event history immediately.
- Do not use `logs` as billing facts.
- Do not use `BillingSession` as billing facts.
- Do not use `SubscriptionPreConsumeRecord` as generic billing facts.

## Usage To Billing Recommendation

Recommended future source:

```text
QuotaUsageRecord(status=committed)
  -> Billing Runtime price calculation
  -> BillingRecord
  -> FundingSource settlement
```

Why this is better:

- `QuotaUsageRecord` already carries provider/channel/model/user/tenant attribution.
- It separates provider measurement from commercial settlement.
- It prevents Billing Runtime from parsing provider-specific usage semantics again.
- It gives Revenue Share and Finance Console a stable usage source.

Compatibility path:

- 10.5B should not remove `PostTextConsumeQuota()`.
- 10.5B can introduce Billing Runtime models and service interfaces.
- 10.5B can run in dry or shadow mode against committed usage facts.
- A later iteration should migrate settlement from `PostTextConsumeQuota()` into Billing Runtime after parity tests exist.

## Failure Path Design

### Usage Succeeds, Billing Fails

Recommended behavior:

- Keep the committed `QuotaUsageRecord`.
- Create or update a billing record with `billing_status=failed`.
- Do not delete usage facts.
- Do not silently ignore billing failure.
- Trigger retry or manual reconciliation path.
- Do not double-settle on retry; retry must use `request_id` or `usage_record_id` idempotency.

Current behavior:

- `SettleBilling()` logs errors from `BillingSession.Settle()`.
- Consume log still records usage.
- There is no durable billing failure record.

Risk:

- Finance cannot reliably identify settled versus failed charges from a durable billing table.

### Billing Succeeds, Settlement Fails

Recommended behavior:

- Separate `funding_settled` from `token_settled` and other side effects.
- Mark partial settlement explicitly.
- Do not call refund automatically if money/subscription has already been committed.
- Require a repair event or compensation path.

Current behavior:

- `BillingSession.Settle()` marks `fundingSettled` before token adjustment.
- If token adjustment fails after funding settled, it logs the token failure and prevents refund from undoing already committed funding.

Risk:

- This protects against over-refund, but the failure is not persisted as a finance-reconcilable billing fact.

### Provider Failure After Preconsume

Recommended behavior:

- `BillingSession.Refund()` remains correct for legacy paths.
- Billing Runtime should write `refunded` or `failed` state for pre-consumed records.
- Subscription refund should continue to use `SubscriptionPreConsumeRecord.request_id` idempotency.
- Wallet refund needs stronger idempotency than current in-memory `WalletFunding.consumed` protection if it becomes retryable across processes.

## Idempotency Design

Recommended keys:

- `request_id`: external relay request id; should be unique per billable request.
- `usage_record_id`: committed usage fact id; preferred stable source for billing.
- `reservation_id`: Quota Runtime correlation id when reservation is active.
- `billing_record_id`: durable billing id for finance and retries.

Recommended constraints:

- Unique `billing_records.request_id` for synchronous text billing where one request produces one charge.
- Or unique `billing_records.usage_record_id` when billing from committed usage facts.
- Future streaming/realtime/task flows may need multiple usage facts and multiple billing records per logical session, so uniqueness should be scoped by mode.

Recommended idempotent settlement flow:

```text
Find BillingRecord by usage_record_id or request_id
If settled: return existing result
If failed/refundable: decide retry policy
If missing: create pending record
Calculate price snapshot
Settle selected FundingSource in one transaction boundary where possible
Mark settled
```

## Revenue Share Compatibility

Billing Runtime must preserve:

- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`
- `provider_name`
- `channel_id`
- `model_name`
- `upstream_model_name`
- `request_id`
- `usage_record_id`
- `billing_record_id`
- `quota_charged`
- `currency`
- price snapshot

Future Revenue Share should read:

- `billing_records` for charged revenue.
- `quota_usage_records` for usage attribution.
- Channel/distribution ownership metadata for split rules.

Revenue Share should not read:

- `User.Quota` deltas directly.
- `Token.RemainQuota` deltas directly.
- UI consume logs as authoritative revenue.
- `SubscriptionPreConsumeRecord` as generic settlement.

## Finance Console Compatibility

Future finance reports should read:

- `billing_records` for billable charges, refunds, adjustments, statuses, and funding sources.
- `quota_usage_records` for provider/model/channel usage volume.
- `logs` only for user-facing historical display and support context.
- `user_subscriptions` for entitlement state and subscription lifecycle.
- `subscription_orders` only for purchase/order compatibility, not runtime usage settlement.

Required report dimensions:

- tenant
- organization
- department
- distribution channel
- user
- token
- provider
- channel
- model
- funding source
- billing status
- date/time window

## Current Risks

High:

- No durable Billing Fact exists for settled, failed, refunded, or partially settled usage.
- `PostTextConsumeQuota()` mixes usage calculation, billing calculation, balance mutation, and logging.
- Wallet refund is not cross-process idempotent if retried after runtime state loss.

Medium:

- `QuotaUsageRecord` and `PostTextConsumeQuota()` can calculate from similar usage inputs but are not yet unified.
- `User.Quota`, `Token.RemainQuota`, and `UserSubscription.AmountUsed` are mutable balances, not ledgers.
- Billing failure is logged but not persisted for reconciliation.

Low:

- Subscription preconsume has request-level idempotency.
- Tenant ownership is generally preserved through `RelayInfo`, `QuotaUsageRecord`, `Log`, and subscription records.

## Recommended 10.5B Development Scope

Recommended iteration name:

`Iteration 10.5B Billing Runtime Foundation Models And Service`

Scope:

- Add `BillingRecord` model.
- Add optional `BillingEvent` only if needed for append-only phase history.
- Add `BillingRuntimeService` interface:
  - `CreateBillingRecordFromUsage()`
  - `CalculateCharge()`
  - `SettleBillingRecord()`
  - `RefundBillingRecord()`
  - `GetBillingRecordByRequestId()`
- Implement dry/shadow mode first.
- Read committed `QuotaUsageRecord(status=committed)` as the input fact.
- Preserve tenant and distribution channel attribution.
- Do not replace `PostTextConsumeQuota()` settlement in 10.5B.
- Do not introduce Voucher, Invoice, Payment, or Revenue Share.

Tests for 10.5B:

- Create billing record from committed usage fact.
- Preserve ownership fields.
- Preserve provider/channel/model attribution.
- Idempotency by `usage_record_id` or `request_id`.
- Reject missing tenant ownership.
- Reject missing usage fact.
- No mutation of `User.Quota`.
- No mutation of `Token.RemainQuota`.
- No mutation of `UserSubscription.AmountUsed`.
- Compatibility with current `PostTextConsumeQuota()` path in shadow mode.

Explicitly defer:

- Real billing settlement migration from `PostTextConsumeQuota()`.
- Voucher funding.
- Channel sponsored funding.
- Free tier funding.
- Invoice generation.
- Payment integration.
- Revenue Share payout.
- Finance Console UI.

## Final Recommendation

Proceed to 10.5B only as Billing Runtime foundation.

The correct foundation is:

```text
QuotaUsageRecord(status=committed)
  -> BillingRecord(pending/settled/failed/refunded)
  -> future FundingSource settlement
```

10.5B should create durable billing facts and service boundaries without changing current balance mutation behavior. Actual settlement migration should wait until BillingRecord parity and idempotency tests are strong enough to avoid double charging.
