# Iteration 10.7A Payment Gateway Architecture Audit

Branch: `feat/iteration-10-6-payment-gateway-foundation`

This audit reviews the current commercial and payment-adjacent architecture before introducing a unified Payment Gateway Foundation. It does not change production code, tests, migrations, or commits.

## Scope

Reviewed modules:

- `model/`
- `service/`
- `controller/`
- `router/`

Primary objects reviewed:

- `TopUp`
- `Redemption`
- `SubscriptionOrder`
- `UserSubscription`
- `BillingRecord`
- `RevenueShareRecord`
- Current payment controllers for Epay, Stripe, Creem, Waffo, and Waffo Pancake

## Current State

The repository already has business-order records and payment-specific callback paths, but it does not yet have a single payment gateway abstraction.

Current payment-shaped records:

- `model.TopUp` is the recharge order and wallet/quota fulfillment record.
- `model.SubscriptionOrder` is the subscription purchase order and webhook fulfillment record.
- `model.Redemption` is a code-based quota grant and should remain outside online payment collection.
- `model.BillingRecord` records usage billing facts and should not collect money.
- `model.RevenueShareRecord` can already reference future `payment`, `order`, or `subscription` sources through `source_type` and `source_id`.

Current payment providers:

- Epay supports user top-up and subscription purchase.
- Stripe supports user top-up and subscription purchase.
- Creem supports user top-up and subscription purchase.
- Waffo and Waffo Pancake support user top-up paths.

Current important safeguards:

- `trade_no` is unique on `TopUp` and `SubscriptionOrder`.
- Payment callbacks use `LockOrder(tradeNo)` in controller code.
- Several model completion paths use transactions and `FOR UPDATE`.
- `CompleteSubscriptionOrder()` is idempotent for already successful subscription orders.
- Provider guard checks reject mismatched payment providers in key completion/expiration paths.
- Ownership metadata is propagated through `ApplyOwnershipFromContext()` and `ownershipFromSubscriptionOrder()`.

Current limitations:

- Payment state, provider payloads, callback timestamps, refund state, and manual review state are split across business objects or not modeled.
- `TopUp` and `SubscriptionOrder` duplicate payment fields but have different fulfillment behavior.
- Payment callbacks decide whether a `trade_no` belongs to a subscription order or a top-up order.
- Some recharge completion paths treat duplicate successful callbacks as an error instead of an idempotent success.
- Refunds are not first-class for payment orders.
- Bank transfer cannot fit cleanly into the current online callback-only model.

## Recommended Architecture

Introduce `PaymentOrder` as the unified payment collection fact. Keep business objects separate:

```text
PaymentOrder
  -> TopUp fulfillment
  -> SubscriptionOrder fulfillment
  -> future Agent Marketplace order fulfillment
  -> future Skill Marketplace order fulfillment
```

`PaymentOrder` should own provider interaction, callback verification result, payment lifecycle, refund lifecycle, and reconciliation metadata. Business orders should own product-specific fulfillment.

Recommended boundary:

| Layer | Responsibility |
| --- | --- |
| Payment Gateway | Create payable order, call provider, verify callback, update payment status, deduplicate provider events |
| TopUp | Credit wallet/quota after a paid payment order |
| Subscription | Create `UserSubscription` after a paid payment order |
| Billing Runtime | Record usage billing facts; no payment collection |
| Revenue Share | Calculate split records from billing/order/payment facts; no payment collection |
| Redemption | Code grant workflow; no payment provider integration |

## 1. PaymentOrder Model Design

Recommended model fields:

| Field | Purpose |
| --- | --- |
| `id` | Internal primary key |
| `order_no` | Platform payment order number, globally unique |
| `tenant_id` | Required tenant ownership |
| `organization_id` | Organization ownership snapshot |
| `department_id` | Department ownership snapshot |
| `distribution_channel_id` | Distribution channel attribution |
| `user_id` | Paying user |
| `payment_provider` | Provider adapter: `WECHAT_PAY`, `ALIPAY`, `BANK_TRANSFER` |
| `payment_method` | Provider method or channel variant, such as `native`, `qr`, `web`, `manual_transfer` |
| `amount` | Exact payable amount in minor units or decimal-safe storage |
| `currency` | Currency code, initially `CNY` for WeChat Pay, Alipay, and domestic bank transfer |
| `status` | Payment lifecycle status |
| `paid_at` | Provider-confirmed or manually approved payment time |
| `callback_at` | Last provider callback/notification processing time |
| `metadata` | Provider payload, product link, QR code URL, bank transfer proof metadata, and reconciliation details |
| `created_at` | Creation timestamp |
| `updated_at` | Update timestamp |

Recommended additional fields before implementation:

| Field | Purpose |
| --- | --- |
| `business_type` | `topup`, `subscription`, `agent`, `skill`, `manual` |
| `business_order_no` | Existing `TopUp.TradeNo`, `SubscriptionOrder.TradeNo`, or future marketplace order number |
| `provider_trade_no` | Provider transaction number |
| `provider_callback_id` | Provider event/notification id when available |
| `expire_at` | Pay window close time |
| `closed_at` | Close/cancel time |
| `refunded_at` | Refund completion time |
| `refund_amount` | Refunded amount |
| `failure_reason` | Normalized failure reason |
| `review_status` | Bank transfer review state if manual transfer is represented in the same table |
| `reviewed_by` | Admin user id for manual review |
| `reviewed_at` | Manual review timestamp |

Money should not be persisted only as `float64`. Current models use `float64` in `Money`, but the payment foundation should use an exact representation, preferably integer minor units plus `currency`, or a decimal string if cross-currency precision requires it. This matters for reconciliation, refunds, and revenue share.

Cross-database constraints should use GORM-compatible indexes and avoid database-specific partial indexes. Recommended indexes:

- Unique `order_no`.
- Unique provider tuple when available: `payment_provider`, `provider_trade_no`.
- Business lookup: `business_type`, `business_order_no`.
- Ownership query indexes: `tenant_id`, `organization_id`, `department_id`, `distribution_channel_id`.
- User history: `user_id`, `created_at`.
- Operations queue: `status`, `expire_at`.

## 2. PaymentProvider

Recommended provider constants:

| Code | Meaning |
| --- | --- |
| `WECHAT_PAY` | WeChat Pay |
| `ALIPAY` | Alipay |
| `BANK_TRANSFER` | Manual or semi-manual bank transfer |

Recommended method constants:

| Provider | Methods |
| --- | --- |
| `WECHAT_PAY` | `native`, `refund` |
| `ALIPAY` | `qr`, `page`, `refund` |
| `BANK_TRANSFER` | `manual_transfer`, `voucher_upload`, `admin_review` |

Provider adapters should expose a common interface:

```text
CreatePayment(ctx, PaymentOrder) -> provider payment payload
VerifyCallback(ctx, raw request) -> verified provider event
QueryPayment(ctx, order_no/provider_trade_no) -> provider status
ClosePayment(ctx, order_no/provider_trade_no) -> close result
RefundPayment(ctx, order_no/provider_trade_no, amount) -> refund result
```

The callback verifier must return normalized fields: `order_no`, `provider_trade_no`, `amount`, `currency`, `status`, `event_id`, `paid_at`, and raw payload metadata.

## 3. Payment State Machine

Recommended payment statuses:

- `pending`
- `paid`
- `failed`
- `closed`
- `refunded`

Recommended transitions:

| From | To | Allowed | Notes |
| --- | --- | --- | --- |
| none | `pending` | Yes | Payment order created locally |
| `pending` | `paid` | Yes | Verified provider success or bank transfer approved |
| `pending` | `failed` | Yes | Provider reports definitive failure |
| `pending` | `closed` | Yes | User cancel, timeout close, provider close |
| `paid` | `refunded` | Yes | Full refund completed |
| `paid` | `paid` | Idempotent | Duplicate success callback |
| `failed` | `failed` | Idempotent | Duplicate failure callback |
| `closed` | `closed` | Idempotent | Duplicate close callback |
| `refunded` | `refunded` | Idempotent | Duplicate refund callback |

Disallowed transitions:

- `failed` -> `paid` unless a provider query proves the local failure was provisional and the state machine explicitly supports correction.
- `closed` -> `paid` unless provider rules allow late payment and the implementation has a documented recovery path.
- `refunded` -> `paid`.
- Any terminal state should not re-run business fulfillment.

Current `common.TopUpStatusSuccess` maps to payment `paid`, and `common.TopUpStatusExpired` maps closest to payment `closed`. Future payment code should use `paid` rather than `success` for payment state while adapters can translate legacy business status.

## 4. WeChat Pay Architecture

Recommended WeChat Pay scope for the first foundation:

- Native Pay QR code for desktop/web console recharge and subscription purchase.
- Provider callback signature verification.
- Provider transaction query for reconciliation and fallback.
- Close unpaid order after expiration.
- Refund API foundation.

Recommended flow:

```text
User selects product
  -> create business order
  -> create PaymentOrder(status=pending, provider=WECHAT_PAY, method=native)
  -> call WeChat Native Pay unified order API
  -> store code_url/prepay metadata
  -> frontend renders QR code
  -> WeChat payment notification
  -> verify signature and decrypt resource
  -> lock PaymentOrder by order_no
  -> validate amount/currency/provider/order_no
  -> mark PaymentOrder paid
  -> fulfill business order exactly once
```

Callback verification requirements:

- Verify WeChat platform certificate/signature.
- Verify notification resource decryption.
- Validate `out_trade_no` equals local `order_no`.
- Validate amount and currency exactly.
- Store raw notification and provider transaction id in `metadata`.
- Return provider-required success response only after local idempotent processing succeeds.

Refund requirements:

- Create refund attempt record or append refund metadata before calling provider.
- Use a unique local refund number.
- Verify refund callback before marking `refunded`.
- Do not reverse quota/subscription automatically without a separate business refund policy.

## 5. Alipay Architecture

Recommended Alipay scope:

- QR code / face-to-face style scan payment for console.
- Web page payment for browser redirect.
- Async notify signature verification.
- Return URL should be display-only and must not be the source of truth.
- Refund API foundation.

Recommended flow:

```text
User selects product
  -> create business order
  -> create PaymentOrder(status=pending, provider=ALIPAY, method=qr/page)
  -> call Alipay precreate/page pay API
  -> frontend renders QR code or redirects to payment page
  -> Alipay async notify
  -> verify RSA/RSA2 signature
  -> validate app_id, seller_id, order_no, amount, currency, trade_status
  -> mark PaymentOrder paid
  -> fulfill business order exactly once
```

Callback verification requirements:

- Verify Alipay signature using the configured Alipay public key.
- Validate `trade_status` values such as `TRADE_SUCCESS` and `TRADE_FINISHED`.
- Validate `out_trade_no`, `total_amount`, `app_id`, and merchant identity.
- Treat browser return as non-authoritative.
- Store provider `trade_no` and raw notify payload in `metadata`.

Refund requirements:

- Use local refund idempotency keys.
- Query refund result when callback is unavailable or delayed.
- Keep refund lifecycle separate from original payment paid state until full refund is confirmed.

## 6. Bank Transfer Architecture

Bank transfer is not a real-time provider callback flow. It should be modeled as a manual payment provider with explicit review states.

Recommended flow:

```text
User creates bank transfer payment
  -> PaymentOrder(status=pending, provider=BANK_TRANSFER, method=manual_transfer)
  -> system shows bank account and transfer instructions
  -> user uploads voucher/proof
  -> admin reviews proof and bank receipt
  -> approve: PaymentOrder -> paid, fulfill business order
  -> reject: keep pending with rejection reason or mark failed/closed
```

Recommended metadata:

- Bank account id or configured receiving account snapshot.
- Expected payer name.
- Expected amount and currency.
- Transfer memo/reference code.
- Uploaded voucher file id/path.
- Reviewer notes.
- Bank receipt transaction id when known.

Manual review rules:

- Approval must be tenant-safe and finance/admin guarded.
- Approval must lock the `PaymentOrder` row.
- Approval must validate amount and target business order.
- Duplicate approval must be idempotent and must not credit quota twice.
- Rejection should preserve audit metadata and should not delete voucher history.

## 7. Idempotency Design

Recommended idempotency keys:

| Operation | Idempotency key |
| --- | --- |
| Create payment order | `order_no` |
| Provider callback | `payment_provider` + `provider_callback_id` or provider transaction id |
| Mark paid | `order_no` + current status guard |
| Business fulfillment | `business_type` + `business_order_no` |
| Refund | local `refund_no` |
| Manual bank approval | `order_no` + review transition guard |

Recommended implementation pattern:

```text
Begin transaction
  -> SELECT PaymentOrder FOR UPDATE by order_no
  -> validate provider, amount, currency, tenant ownership, status
  -> if already paid/refunded/closed, return idempotent result
  -> update PaymentOrder status and provider metadata
  -> call one business fulfillment function inside the same transaction
Commit
```

Business fulfillment should be invoked through a registry or switch on `business_type`, for example:

- `topup`: credit quota and mark `TopUp` success.
- `subscription`: complete `SubscriptionOrder` and create `UserSubscription`.
- `agent`: grant purchased agent entitlement.
- `skill`: grant purchased skill entitlement.

The payment layer should never credit quota or create subscriptions directly without going through the business fulfillment boundary.

Duplicate callback response policy:

- Verified duplicate success for an already paid order should return provider success.
- Verified duplicate failed/closed event for a terminal order should return provider success if no local mutation is needed.
- Provider mismatch, amount mismatch, currency mismatch, or business order mismatch should be logged and rejected.

## 8. Top-Up Integration

Recommended top-up path:

```text
TopUp(status=pending)
  -> PaymentOrder(status=pending, business_type=topup, business_order_no=TopUp.TradeNo)
  -> provider payment
  -> PaymentOrder(status=paid)
  -> TopUp fulfillment
  -> User.Quota increment
  -> top-up log
```

`TopUp` should remain the business fulfillment record for wallet/quota crediting. `PaymentOrder` should become the payment collection record.

Recommended changes for future implementation:

- Keep `TopUp.TradeNo` as business order number during migration, or explicitly map it to `PaymentOrder.order_no`.
- Add a service function that completes top-up by a paid `PaymentOrder` inside one transaction.
- Normalize duplicate successful callback behavior across all providers.
- Preserve ownership fields from `PaymentOrder` to `TopUp`.
- Record payment amount/currency in `PaymentOrder`, and keep quota amount in `TopUp`.

## 9. Subscription Integration

Recommended subscription path:

```text
SubscriptionOrder(status=pending)
  -> PaymentOrder(status=pending, business_type=subscription, business_order_no=SubscriptionOrder.TradeNo)
  -> provider payment
  -> PaymentOrder(status=paid)
  -> CompleteSubscriptionOrder()
  -> UserSubscription created from plan snapshot
  -> optional TopUp compatibility record
```

`SubscriptionOrder` should remain the purchase intent and plan snapshot linkage. `PaymentOrder` should own provider payment state and callback metadata.

Current `CompleteSubscriptionOrder()` is a good foundation because it:

- Locks by `trade_no`.
- Checks expected provider.
- Is idempotent for already successful orders.
- Creates `UserSubscription` in the same transaction.
- Creates or updates a compatibility `TopUp` record.
- Propagates ownership metadata.

Future payment integration should call subscription fulfillment only after `PaymentOrder` is verified paid. Provider payload should move primarily to `PaymentOrder.metadata`; `SubscriptionOrder.ProviderPayload` can be retained temporarily for backward compatibility.

## 10. Future Agent Marketplace And Skill Marketplace Reuse

The same payment foundation should support future marketplace products by treating payment as a generic collection layer and fulfillment as product-specific.

Recommended marketplace order model relationship:

```text
PaymentOrder
  business_type=agent
  business_order_no=AgentOrder.OrderNo

PaymentOrder
  business_type=skill
  business_order_no=SkillOrder.OrderNo
```

Marketplace fulfillment should own:

- Product snapshot.
- Seller/developer identity.
- Tenant and distribution channel ownership.
- Entitlement grant.
- Refund policy.
- Revenue share source metadata.

Payment foundation should expose paid facts to Revenue Share:

- `RevenueShareRecord.SourceType = payment` for payment-level split.
- `RevenueShareRecord.SourceType = order` for marketplace order-level split.
- `RevenueShareRecord.SourceType = subscription` for subscription purchase split.

Recommended product types for revenue share matching:

- `subscription`
- `agent`
- `skill`
- `billing`

This aligns with existing `RevenueShareRule` scopes and avoids adding a separate payment-specific revenue share mechanism.

## Tenant Safety

`PaymentOrder` must carry the full ownership snapshot:

- `tenant_id`
- `organization_id`
- `department_id`
- `distribution_channel_id`
- `user_id`

Creation should use `ApplyOwnershipFromContext()` or the equivalent service-level ownership snapshot. Callback paths have no authenticated user context, so they must never infer tenant from request parameters. They should load by `order_no`, then operate only on the loaded order and its linked business order.

Admin query and manual bank review must use `ApplyOwnershipScope()` and must not expose cross-tenant payment data to non-root users.

## Router And Controller Shape

Recommended future route shape:

```text
POST /api/payment/orders
GET  /api/payment/orders/:order_no
POST /api/payment/wechat/notify
POST /api/payment/alipay/notify
POST /api/payment/bank-transfer/:order_no/voucher
POST /api/payment/admin/bank-transfer/:order_no/approve
POST /api/payment/admin/bank-transfer/:order_no/reject
POST /api/payment/admin/orders/:order_no/close
POST /api/payment/admin/orders/:order_no/refund
```

Existing provider-specific top-up/subscription routes can be migrated gradually. The compatibility phase can create both the existing business order and the new `PaymentOrder`, then route callbacks through the unified payment service.

## Testing Note

ADR-002 requires Go test code to live in `*_test.go` files. This audit adds no tests. Future payment foundation tests should cover:

- Provider callback idempotency.
- Provider mismatch rejection.
- Amount/currency mismatch rejection.
- Duplicate paid callback does not duplicate quota or subscription fulfillment.
- Bank transfer approve/reject idempotency.
- Tenant-scoped payment order listing and manual review.
- Cross-database migration and query compatibility for SQLite, MySQL, and PostgreSQL.

## Recommended Implementation Order

1. Add `PaymentOrder` model and constants without changing existing payment behavior.
2. Add provider adapter interfaces and normalized callback event DTOs.
3. Implement payment service state transitions and idempotent paid handling.
4. Integrate top-up fulfillment behind `PaymentOrder`.
5. Integrate subscription fulfillment behind `PaymentOrder`.
6. Add WeChat Pay Native Pay.
7. Add Alipay QR/page pay.
8. Add bank transfer manual review.
9. Connect payment/order facts to `RevenueShareRecord` where product policy requires it.
10. Migrate old payment routes to compatibility wrappers or deprecate them after frontend adoption.

## Conclusion

The next foundation should not expand `TopUp` or `SubscriptionOrder` into a universal payment table. They are business objects with payment fields. A separate `PaymentOrder` gives the project a clean payment boundary for WeChat Pay, Alipay, bank transfer, refunds, reconciliation, and future marketplace products while preserving the existing quota, subscription, billing, and revenue share boundaries.
