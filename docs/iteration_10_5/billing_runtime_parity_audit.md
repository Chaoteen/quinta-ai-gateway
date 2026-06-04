# Iteration 10.5E Billing Runtime Parity Audit

Branch: `feat/iteration-10-5-billing-runtime-foundation`

This audit compares the current production deduction path with the Billing Runtime shadow result. It does not change production code or tests.

## Scope

Reviewed modules:

- `service/`
- `model/`
- `relay/`

Focus objects and functions:

- `PostTextConsumeQuota()`
- `PostAudioConsumeQuota()`
- `ModelPriceHelper()`
- `GetCompletionRatio()`
- `GetAudioRatio()`
- `GetAudioCompletionRatio()`
- `SettleBilling()`
- `BillingSession`
- `FundingSource`
- `QuotaUsageRecord`
- `BillingRecord`
- `UsageMetering`

## Current Production Text Billing Chain

Current synchronous text billing flow:

```text
Relay request
  -> token estimation
  -> ModelPriceHelper()
  -> PreConsumeBilling()
  -> provider request / response
  -> dto.Usage
  -> Usage Metering dry fact
  -> PostTextConsumeQuota()
  -> calculateTextQuotaSummary()
  -> optional TryTieredSettle()
  -> UpdateUserUsedQuotaAndRequestCount()
  -> UpdateChannelUsedQuota()
  -> SettleBilling()
  -> BillingSession.Settle()
  -> FundingSource.Settle()
  -> wallet / token / subscription mutation
  -> RecordConsumeLog()
```

`ModelPriceHelper()` builds the `PriceData` snapshot used later by `PostTextConsumeQuota()`:

- `ModelPrice`
- `ModelRatio`
- `CompletionRatio`
- `CacheRatio`
- `CacheCreationRatio`
- `CacheCreation5mRatio`
- `CacheCreation1hRatio`
- `ImageRatio`
- `AudioRatio`
- `AudioCompletionRatio`
- `GroupRatioInfo`
- `UsePrice`
- `QuotaToPreConsume`
- `TieredBillingSnapshot` for `tiered_expr` models

## Production Text Quota Calculation

`PostTextConsumeQuota()` delegates quota math to `calculateTextQuotaSummary()`.

Input dimensions:

- `PromptTokens`
- `CompletionTokens`
- `TotalTokens`
- `PromptTokensDetails.CachedTokens`
- `PromptTokensDetails.CachedCreationTokens`
- `ClaudeCacheCreation5mTokens`
- `ClaudeCacheCreation1hTokens`
- `PromptTokensDetails.ImageTokens`
- `PromptTokensDetails.AudioTokens`
- `UsageSemantic`
- `UsageSource`
- provider-specific `Cost` for some OpenRouter Claude cache inference

Pricing dimensions:

- model ratio
- completion ratio
- cache read ratio
- cache creation ratio
- Claude 5m / 1h cache creation ratio split
- image ratio
- Gemini input audio price
- group ratio
- model fixed price
- other ratios
- tiered expression billing
- tool surcharges

Non-fixed-price formula shape:

```text
base_prompt_tokens = prompt_tokens

if cache read tokens:
  non-Claude/non-legacy path subtracts cache tokens from base prompt tokens
  cache quota = cache_tokens * cache_ratio

if cache creation tokens:
  non-Claude/non-legacy path subtracts cache creation tokens from base prompt tokens
  cache creation quota = cache_creation_tokens * cache_creation_ratio
  Claude semantic path may split 5m and 1h cache creation tokens

if image tokens:
  base prompt tokens subtracts image tokens
  image quota = image_tokens * image_ratio

if audio input tokens with Gemini audio input price:
  base prompt tokens subtracts audio tokens
  audio input quota = audio_price_per_million * audio_tokens * group_ratio * quota_per_unit

prompt quota = base prompt tokens + cache quota + image quota + cache creation quota
completion quota = completion_tokens * completion_ratio

quota = (prompt quota + completion quota) * model_ratio * group_ratio
quota += tool surcharge quota
quota += audio input quota
quota *= each other_ratio
quota = round(quota)
```

Fixed-price formula shape:

```text
quota = model_price * quota_per_unit * group_ratio
quota += tool surcharge quota
quota += audio input quota
quota *= each other_ratio
quota = round(quota)
```

Special cases:

- If `TotalTokens == 0`, quota is set to `0`.
- If ratio is non-zero and calculated quota rounds to `0`, quota becomes `1`.
- OpenRouter Claude usage can infer missing cache creation tokens from upstream `Cost`.
- Tiered billing can override the quota using `TryTieredSettle()` and `composeTieredTextQuota()`.
- Tool surcharges include web search, Claude web search, file search, and image generation call surcharge.
- `RecordConsumeLog()` stores a UI/support log, but settlement occurs through `SettleBilling()`.

## Current Production Audio Billing Chain

`PostAudioConsumeQuota()` computes audio quota separately.

Input dimensions:

- input text tokens
- output text tokens
- input audio tokens
- output audio tokens
- total tokens

Pricing dimensions:

- model ratio
- completion ratio
- audio ratio
- audio completion ratio
- group ratio
- model fixed price
- tiered billing override

Non-fixed-price formula shape:

```text
quota = input_text_tokens
quota += output_text_tokens * completion_ratio
quota += input_audio_tokens * audio_ratio
quota += output_audio_tokens * audio_ratio * audio_completion_ratio
quota *= model_ratio * group_ratio
quota = round(quota)
```

Fixed-price formula:

```text
quota = model_price * quota_per_unit * group_ratio
```

Special cases:

- If `TotalTokens == 0`, quota is set to `0`.
- If tiered billing applies, tiered quota replaces the normal audio quota.
- After quota calculation, audio follows the same `SettleBilling()` path.

## BillingRecord Shadow Calculation

Current `BillingRuntime.CalculateCharge()` behavior:

```text
validate committed QuotaUsageRecord
quota_charged = usage.TokenDelta
if token_delta == 0:
  quota_charged = usage.TotalTokens
currency = "QUOTA"
unit_price_snapshot = {"mode":"shadow"}
price_snapshot = {"source":"quota_usage_record"}
metadata = {
  "mode": "shadow",
  "usage_record_id": usage.Id,
  "usage_source": usage.UsageSource,
  "usage_semantic": usage.UsageSemantic
}
```

Current BillingRecord fields copied from Usage Fact:

- ownership fields
- request id
- reservation id
- usage record id
- user id
- user subscription id
- provider name
- channel id
- model name
- input tokens
- output tokens
- total tokens
- request count

Current BillingRecord does not read:

- `RelayInfo.PriceData`
- model ratio
- completion ratio
- group ratio
- cache ratio
- image ratio
- audio ratio
- tiered billing expression snapshot
- tool surcharge context
- OpenRouter `Cost`
- consume log `other`

## Difference Analysis

High-impact differences:

- Shadow `quota_charged` is currently `token_delta` or `total_tokens`, not production quota.
- Production quota applies model ratio and group ratio; shadow does not.
- Production quota applies completion ratio; shadow does not.
- Production quota handles cache read and cache write discounts/surcharges; shadow does not.
- Production quota handles Claude 5m/1h cache write split; shadow does not.
- Production quota handles image tokens; shadow does not.
- Production quota handles audio tokens and Gemini input audio pricing; shadow does not.
- Production quota handles fixed-price models; shadow does not.
- Production quota handles `OtherRatios`; shadow does not.
- Production quota handles tiered expression billing; shadow does not.
- Production quota handles tool surcharges; shadow does not.
- Production quota has zero-token and minimum-one-quota rules; shadow does not match those rules.

Medium-impact differences:

- Shadow does not snapshot `PriceData`.
- Shadow does not snapshot `TieredBillingSnapshot`.
- Shadow does not snapshot user group special ratio.
- Shadow has no parity link to `Log.Quota`.
- Shadow currently supports only committed usage facts available in the 10.4 dry metering path.

Low-impact similarities:

- Shadow preserves tenant and distribution channel attribution from `QuotaUsageRecord`.
- Shadow preserves provider, channel, model, user, subscription, request, and usage record correlation.
- Shadow is idempotent by `usage_record_id`.

Conclusion:

Current BillingRecord shadow mode is not a production billing calculation. It is a durable usage-linked billing fact placeholder. It is useful for attribution, reconciliation scaffolding, and future parity tests, but it cannot replace `PostTextConsumeQuota()` or `PostAudioConsumeQuota()` yet.

## Parity Test Design

The parity test goal is:

```text
For the same RelayInfo + dto.Usage input:
  production quota calculation result
  equals
  Billing Runtime calculated quota_charged
```

Recommended prerequisite:

- Extract production quota calculation into pure or semi-pure service helpers that do not mutate balances.
- Keep mutation functions (`SettleBilling`, `UpdateUserUsedQuotaAndRequestCount`, `UpdateChannelUsedQuota`) out of parity tests.
- Create test fixtures for `RelayInfo.PriceData`, `dto.Usage`, provider semantic fields, and context tool flags.

Recommended helper shape:

```text
CalculateTextBillingSnapshot(ctx, relayInfo, usage) -> BillingCalculationSnapshot
CalculateAudioBillingSnapshot(ctx, relayInfo, usage) -> BillingCalculationSnapshot
BillingRuntime.CalculateChargeFromSnapshot(snapshot, usageRecord) -> BillingCharge
```

The snapshot should include:

- quota charged
- model name
- provider name
- usage semantic
- input tokens
- output tokens
- total tokens
- cached tokens
- cache creation tokens
- cache creation 5m / 1h tokens
- image tokens
- audio tokens
- model ratio
- completion ratio
- cache ratio
- cache creation ratio
- image ratio
- audio ratio
- audio completion ratio
- group ratio
- model price
- use price
- other ratios
- tiered billing trace
- tool surcharge trace

## Provider Coverage

OpenAI compatible:

- plain prompt/completion tokens
- cached tokens
- reasoning tokens if represented in token details
- Responses API built-in web search / file search surcharge
- image generation call surcharge
- fixed price and ratio modes

Claude / Anthropic:

- anthropic usage semantic
- cache read tokens
- cache creation tokens
- 5m cache creation tokens
- 1h cache creation tokens
- Claude web search count surcharge
- thinking model names

Gemini:

- prompt/completion tokens
- image input tokens
- audio input tokens
- Gemini input audio price
- audio completion tokens when routed through audio path
- model ratio and group ratio

DeepSeek:

- compatible prompt/completion usage
- reasoning models
- cache ratio if configured
- model ratio and group ratio

Qwen / Ali:

- compatible prompt/completion usage
- model ratio and group ratio
- provider/channel attribution

OpenRouter:

- compatible usage
- OpenRouter Claude semantic
- upstream `Cost`-based cache creation inference
- provider name and channel attribution

## Test Dimensions

Token dimensions:

- `input_tokens`
- `output_tokens`
- `total_tokens`
- `prompt_tokens`
- `completion_tokens`
- `cached_tokens`
- `cache_creation_tokens`
- `cache_creation_tokens_5m`
- `cache_creation_tokens_1h`
- `reasoning_tokens`
- `audio_tokens`
- `image_tokens`
- `request_count`

Pricing dimensions:

- model ratio
- completion ratio
- cache ratio
- cache creation ratio
- image ratio
- audio ratio
- audio completion ratio
- group ratio
- user-group special ratio
- fixed model price
- `OtherRatios`
- tiered billing expression
- tool surcharge prices

Behavior dimensions:

- zero total tokens
- minimum quota rounding
- free model
- fixed price mode
- ratio mode
- tiered expression mode
- missing upstream usage
- estimated usage
- converted usage

## Risk Classification

High:

- Replacing production settlement with current BillingRecord shadow would materially mischarge most non-trivial requests.
- Cache, image, audio, tool surcharge, tiered billing, fixed price, and group-ratio logic are absent from `CalculateCharge()`.
- BillingRecord currently lacks price snapshots needed for finance audit parity.
- Streaming/audio/task/realtime flows are not covered by BillingRecord shadow generation.

Medium:

- `QuotaUsageRecord` uses normalized usage fields, while production quota still reads provider-specific `dto.Usage` details.
- `token_delta` may equal normalized total tokens, not billed quota.
- `PostTextConsumeQuota()` and Billing Runtime currently have separate calculation responsibilities, increasing drift risk.
- Multi-fact request handling intentionally fails closed, so future streaming/realtime billing policy remains open.

Low:

- BillingRecord idempotency by `usage_record_id` is sound for single-fact synchronous text.
- Ownership and channel attribution are preserved in shadow records.
- Shadow failures are non-blocking and do not mutate balances.

## Should 10.5F Add Parity Tests?

Recommendation:

Yes. 10.5 should continue with `10.5F Billing Runtime Parity Test Foundation` before opening a PR that claims Billing Runtime is ready for migration.

Reasoning:

- 10.5B-10.5D establish durable facts and shadow generation, but not billing calculation parity.
- Without parity tests, there is no defensible path to replace `PostTextConsumeQuota()`.
- Parity tests can be added without changing real settlement logic.
- The tests will clarify what data BillingRecord must store in `price_snapshot` and `unit_price_snapshot`.

Recommended 10.5F scope:

- No production settlement migration.
- No Relay behavior changes.
- Add pure calculation helpers or test-only wrappers if necessary.
- Add tests comparing production calculation snapshots to BillingRuntime charge snapshots.
- Start with synchronous non-stream text.
- Cover OpenAI, Claude, Gemini, DeepSeek, Qwen/Ali, and OpenRouter fixture classes.

## Can The Branch Prepare PR Now?

Recommendation:

Prepare PR only if the stated scope is "Billing Runtime shadow foundation" and not "Billing Runtime replacement."

Acceptable PR claims:

- BillingRecord model foundation exists.
- BillingRecord shadow generation from committed usage facts exists.
- Relay dry path can generate shadow BillingRecords.
- No real balances are modified by Billing Runtime.

Claims that are not yet supported:

- Billing Runtime calculates production charges.
- Billing Runtime can replace `PostTextConsumeQuota()`.
- Billing Runtime can replace `PostAudioConsumeQuota()`.
- Finance reports can rely on `BillingRecord.quota_charged` as authoritative revenue.

## Recommended Route

10.5F:

- Build Billing Runtime parity test foundation.
- Extract or expose non-mutating billing calculation snapshots.
- Confirm where `price_snapshot` and `unit_price_snapshot` need richer data.
- Keep BillingRecord shadow-only.

10.6:

- If parity is proven, introduce Billing Runtime calculation snapshots as production-compatible facts.
- Continue shadow mode for at least one iteration.
- Only later migrate settlement from `PostTextConsumeQuota()` / `PostAudioConsumeQuota()` to Billing Runtime.

Do not move to real settlement migration until:

- text parity tests pass
- audio parity tests pass
- tiered billing parity tests pass
- cache and tool surcharge parity tests pass
- BillingRecord stores enough price snapshot data for finance audit

## 10.5F Implementation Notes

Implemented parity foundation scope:

- Added `BillingCalculationSnapshot`.
- Added `BuildBillingCalculationSnapshotFromUsage()`.
- Added `CompareBillingCalculationSnapshot()`.
- Added service tests for simple text usage, shadow quota matching, total token fallback, mismatch detection, attribution preservation, and no balance mutation.

Snapshot purpose:

- Provide a non-mutating structure for comparing billing calculation outputs.
- Preserve request, usage record, provider, channel, model, tenant, organization, department, distribution channel, token counts, quota, currency, calculation source, and metadata.
- Establish a stable target for future production quota calculation snapshots.

What 10.5F can compare now:

- `BillingRecord.quota_charged` against an expected quota value.
- `BillingRuntime.CalculateCharge()` token-delta behavior.
- `total_tokens` fallback behavior when `token_delta` is zero.
- Tenant/provider/channel/model attribution copied from the usage fact and BillingRecord.

What 10.5F cannot compare yet:

- Production `PostTextConsumeQuota()` quota math.
- Production `PostAudioConsumeQuota()` quota math.
- cache read/write pricing.
- Claude 5m/1h cache split.
- image pricing.
- audio pricing.
- fixed price models.
- group ratio and user-group special ratio.
- tiered expression billing.
- tool surcharge pricing.
- OpenRouter `Cost`-based cache inference.

Why this is still useful:

- It creates the comparison contract before migrating formulas.
- It keeps parity tests non-balance-affecting.
- It makes quota mismatch explicit via `CompareBillingCalculationSnapshot()`.

Next expansion path:

1. Add a non-mutating text billing calculation snapshot helper that mirrors `calculateTextQuotaSummary()`.
2. Compare that production-style snapshot against Billing Runtime shadow snapshots.
3. Add missing fields to BillingRecord `price_snapshot` and `unit_price_snapshot`.
4. Repeat for audio, cache, image, tiered billing, and tool surcharge fixtures.
