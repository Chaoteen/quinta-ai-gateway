# Iteration 10.4A Usage Metering Design Review

Branch: `feat/iteration-10-4-usage-metering-foundation`

This review audits the current relay, quota, billing, log, and task paths before implementing Usage Metering Foundation. It does not change production behavior.

## Scope

Reviewed areas:

- `relay/`
- `relay/common/`
- `relay/channel/`
- `service/`
- `model/`

Primary objects reviewed:

- `QuotaUsageRecord`
- `QuotaReservation`
- `BillingSession`
- `FundingSource`
- `Log`
- `Task`
- `RelayInfo`
- provider-specific usage mapping for OpenAI, Claude, Gemini, DeepSeek, Qwen/Ali, and compatible paths

## Current Usage Sources

The current runtime has one important normalization point: most provider handlers return a `*dto.Usage`, then relay calls `PostTextConsumeQuota()` or `PostAudioConsumeQuota()`. That means provider-specific parsing is already mostly isolated in channel adaptors.

### OpenAI And OpenAI-Compatible

Source:

- Chat completions response `usage.prompt_tokens`, `usage.completion_tokens`, `usage.total_tokens`.
- Responses API `usage.input_tokens`, `usage.output_tokens`, and `input_tokens_details.cached_tokens`.
- Streaming responses may include final usage through stream options.
- If upstream usage is missing, the code estimates prompt tokens from `RelayInfo.GetEstimatePromptTokens()` and completion tokens from response text.

Current mapping:

- `PromptTokens` is provider `prompt_tokens` or estimated prompt tokens.
- `CompletionTokens` is provider `completion_tokens` or estimated output tokens.
- `TotalTokens` is usually upstream total or prompt + completion.
- `InputTokens` / `OutputTokens` are populated for Responses API and realtime paths.
- Cache tokens are normalized into `PromptTokensDetails.CachedTokens`.

Risk:

- Estimated fallback usage is useful for billing continuity but should be marked as estimated in Usage Metering.
- OpenAI-compatible providers differ in usage payload shape; Usage Metering should store `usage_source` / `usage_semantic` or equivalent metadata.

### Claude

Source:

- Anthropic message usage:
  - `input_tokens`
  - `output_tokens`
  - `cache_read_input_tokens`
  - `cache_creation_input_tokens`
  - split cache creation fields such as ephemeral 5m and 1h
  - server tool usage such as web search requests

Current mapping:

- `PromptTokens` is Anthropic `input_tokens`.
- `CompletionTokens` is Anthropic `output_tokens`.
- Cache read/write fields are preserved in `PromptTokensDetails`.
- `UsageSemantic` is set to `anthropic`.
- When converted to OpenAI format, code can build OpenAI-style total input by adding input + cache read + cache creation.

Risk:

- Anthropic semantics are not identical to OpenAI prompt token semantics.
- Usage Metering must avoid losing cache read/write fields, otherwise future billing and reconciliation cannot explain differences.

### Gemini

Source:

- Gemini `usageMetadata`:
  - `promptTokenCount`
  - `toolUsePromptTokenCount`
  - `candidatesTokenCount`
  - `thoughtsTokenCount`
  - `totalTokenCount`
  - `cachedContentTokenCount`
  - prompt/tool/candidate token detail arrays by modality

Current mapping:

- `PromptTokens = promptTokenCount + toolUsePromptTokenCount`.
- `CompletionTokens = candidatesTokenCount + thoughtsTokenCount`.
- `TotalTokens = totalTokenCount`.
- Reasoning tokens are written to completion details.
- Cached content tokens are written to prompt token details.
- Text/audio/image modality details are preserved where available.
- If prompt usage is missing, the relay falls back to estimated prompt tokens.

Risk:

- Gemini tool prompt tokens and thoughts tokens are meaningful and should be explicitly preserved.
- Total tokens can be present even when prompt/candidate components are incomplete; Usage Metering needs both normalized totals and raw provider fields.

### DeepSeek

Source:

- DeepSeek adaptor delegates response handling to OpenAI or Claude adaptors depending on relay format.

Current mapping:

- OpenAI path uses OpenAI-style usage.
- Claude path uses Anthropic-style usage.
- DeepSeek V4 thinking suffix handling changes request/model semantics but not the response usage extraction boundary.

Risk:

- Metering should record final upstream model name after suffix/mapping normalization.
- Metering should record final request relay format to distinguish OpenAI-compatible vs Claude-compatible usage semantics.

### Qwen / Ali

Source:

- Ali native usage has:
  - `input_tokens`
  - `output_tokens`
  - `total_tokens`
  - sometimes `image_count`
- Many Qwen-compatible text paths use OpenAI-compatible response handling.

Current mapping:

- Native Ali rerank and task/image paths map usage into `dto.Usage`.
- OpenAI-compatible text paths follow the OpenAI handler.

Risk:

- Some Ali/Qwen paths are not plain text completion paths. Usage Metering should avoid assuming every usage event is a text chat event.

### Other Compatible Paths

Observed examples:

- SiliconFlow maps `input_tokens` / `output_tokens`.
- Tencent maps `PromptTokens`, `CompletionTokens`, `TotalTokens`.
- xAI may return only total tokens and derive completion as total - prompt.
- Audio/realtime paths populate audio/text input/output details and may settle multiple times during a websocket session.
- Async task paths are per-call or task-status based and should be handled separately from synchronous text usage.

## Relay Integration Point

The most stable Usage Metering integration point is after provider response normalization and before billing settlement.

Recommended synchronous text/audio sequence:

1. Build `RelayInfo` and resolve ownership.
2. Resolve model mapping and final upstream model.
3. `CheckQuota()`.
4. `ReserveQuota()`.
5. Provider request.
6. Provider response handler returns normalized `dto.Usage`.
7. Usage Metering normalizes `dto.Usage` into metering input.
8. `CommitUsage()` writes final usage to `QuotaUsageRecord`.
9. Billing Runtime reads the committed usage event or metering result and settles once.
10. Existing `Log` writes remain a UI/audit log path, not the source of truth.

Recommended failure sequence:

1. If quota check fails, no provider request is made.
2. If reserve succeeds but provider request fails, call `RollbackReservation()`.
3. If provider response parsing fails before reliable usage is known, rollback reservation unless a future policy explicitly commits estimated usage.
4. If response succeeds but usage is missing, commit an estimated usage event only if product policy permits estimated metering.

Important ordering rule:

- Usage Metering should not be placed inside individual channel handlers unless provider-specific raw metadata extraction is required.
- Channel handlers should continue returning normalized `dto.Usage`.
- A centralized metering service should consume `RelayInfo`, `dto.Usage`, and reservation context.

## QuotaUsageRecord Field Review

Current `QuotaUsageRecord` fields are sufficient for basic quota runtime state:

- ownership fields
- `user_id`
- `user_subscription_id`
- `request_id`
- `reservation_id`
- `model_name`
- `quota_dimension`
- `token_delta`
- `request_delta`
- `status`
- `metadata`
- timestamps

They are not sufficient as a durable Usage Metering fact table.

Recommended fields for 10.4B:

| Field | Recommendation | Reason |
| --- | --- | --- |
| `provider_name` | Add | Needed for provider attribution and reconciliation. |
| `channel_id` | Add | Needed for channel-level reporting and distribution channel attribution. |
| `channel_name` | Do not add as authoritative field | Channel names can change. Use `channel_id`; name can be denormalized in metadata only. |
| `request_count` | Add | Request quota is a first-class dimension. Avoid overloading `request_delta`. |
| `input_tokens` | Add | Required for metering, billing, and provider reconciliation. |
| `output_tokens` | Add | Required for metering, billing, and provider reconciliation. |
| `total_tokens` | Add | Required for audit and provider invoice comparison. |
| `usage_source` | Add or put in structured metadata | Distinguish upstream usage vs estimated usage vs converted usage. |
| `usage_semantic` | Add or put in structured metadata | Distinguish OpenAI, Anthropic, Gemini, and compatible semantics. |
| `relay_mode` | Metadata is acceptable | Useful for debugging endpoint behavior. |
| `relay_format` | Metadata is acceptable | Important for converted usage semantics. |
| `upstream_model_name` | Add or metadata | `model_name` may be origin or normalized model; future design should define which one is authoritative. |

Recommended interpretation:

- Keep `token_delta` as subscription quota consumption units used by Quota Runtime.
- Add explicit token-metering fields for actual provider usage.
- Keep `request_delta` for quota runtime request accounting.
- Add `request_count` for metering clarity and future aggregation.

## Billing Runtime Compatibility

Current Billing Runtime:

- `BillingSession` does pre-consume, settle, and refund.
- `FundingSource` mutates wallet or legacy subscription amount fields.
- `SettleBilling()` currently receives an `actualQuota` computed by `PostTextConsumeQuota()` / `PostAudioConsumeQuota()`.

Future compatibility requirement:

- Billing must not independently re-count provider response tokens if Usage Metering has already committed a usage event.
- Billing should read one committed metering event by `reservation_id` or `request_id`, then convert usage into charge units.
- The billing input should include a stable `usage_record_id` or `reservation_id` so settlement is traceable.
- Billing should write financial ledger records separately; it should not make `QuotaUsageRecord` a wallet ledger.

Recommended future flow:

1. Usage Metering commits a usage event.
2. Billing receives `usage_record_id`.
3. Billing computes cost from that event and pricing snapshot.
4. Billing settles wallet/subscription/funding source once.
5. Billing stores settlement reference to the usage record.

Double-counting risk:

- Existing `PostTextConsumeQuota()` computes quota and calls `SettleBilling()`.
- If `CommitUsage()` also changes wallet/token/subscription amount fields, the request will be double-counted.
- Therefore 10.4B should only write usage facts and should not change billing settlement behavior until a later integration iteration.

## Distribution Channel Compatibility

Current ownership fields already include `distribution_channel_id` in:

- `RelayInfo`
- `Log`
- `Task`
- `QuotaReservation`
- `QuotaUsageRecord`

Future channel attribution should read:

- `distribution_channel_id`
- `tenant_id`
- `organization_id`
- `department_id`
- `user_id`
- `channel_id`
- `model_name`
- token fields
- request count
- committed status
- occurred time

Recommendation:

- Distribution Channel reporting should aggregate committed `QuotaUsageRecord` rows, not `Log.Other`.
- `channel_id` should be stored directly on usage records.
- `provider_name` should be stored directly or derived consistently from `channel_type`.
- `channel_name` should be display-only because names can change.
- Revenue share must remain out of 10.4; Usage Metering only provides attribution-ready facts.

## Performance Impact

Usage Metering touches the relay hot path. The main performance risks are:

- Extra database writes per request.
- Extra aggregation queries during quota check.
- Transaction contention on reservation and usage rows.
- Slow metadata serialization.
- Synchronous writes for high-volume streaming/realtime requests.

Recommended mitigations:

- Keep one reservation write before provider request and one committed usage write after provider response.
- Do not write per stream chunk.
- Use append-only usage events.
- Store raw/extended provider metadata in compact JSON metadata, not many rarely queried columns.
- Index `request_id`, `reservation_id`, `user_subscription_id`, `status`, `occurred_at`, `channel_id`, and ownership fields.
- Consider async export/aggregation later, but keep the committed usage event synchronous if quota enforcement depends on it.
- Avoid using `Log` as the metering source because logs can be disabled through `LogConsumeEnabled`.

## Problems Found

1. `QuotaUsageRecord` lacks provider/channel/token detail fields needed for production-grade metering.
2. `Log` has useful usage fields but is not a reliable metering source because consume logs can be disabled and `Other` is unstructured.
3. Current `dto.Usage` includes normalized usage but not a durable source confidence field for estimated vs upstream-reported usage.
4. Anthropic, OpenAI, and Gemini semantics differ enough that a single `total_tokens` field is not sufficient for audit.
5. `BillingSession` currently settles from quota computed in `PostTextConsumeQuota()`, not from a committed usage event.
6. Realtime/websocket usage may commit multiple partial usages; 10.4B should either defer realtime or define session-level idempotency.
7. Async `Task` billing is per-call/task lifecycle oriented and should not be mixed with synchronous text Usage Metering in the first implementation.
8. `QuotaReservation.request_id` service idempotency exists, but DB-level uniqueness is still not guaranteed.

## Recommended 10.4B Development Scope

Recommended iteration name:

`Iteration 10.4B Usage Metering Foundation Models And Service`

Recommended scope:

- Add usage metering fields to `QuotaUsageRecord` or add a dedicated usage-metering model if schema separation is preferred.
- Add a service-level normalizer from `RelayInfo` + `dto.Usage` to a metering input DTO.
- Add service method to commit usage facts without touching BillingSession, FundingSource, wallet, token quota, or legacy subscription `amount_used`.
- Add tests for OpenAI, Claude, Gemini, estimated usage, channel attribution, tenant ownership, and no billing mutation.
- Keep relay integration deferred until the metering service has deterministic tests.

Do not include in 10.4B:

- Relay hot-path integration.
- Billing settlement rewrite.
- Wallet/token mutation.
- Revenue share.
- Invoice/payment/voucher behavior.
- Admin Console pages.

## Recommended Integration Plan

Phase 1: service-only foundation

- Define `UsageMeteringInput`.
- Normalize provider usage into explicit metering fields.
- Write committed usage facts linked to `reservation_id` and `request_id`.
- Preserve ownership and channel attribution.

Phase 2: relay dry integration

- Insert metering after `adaptor.DoResponse()` and before post-consume billing.
- Keep existing billing behavior unchanged.
- Verify no duplicate settlement.

Phase 3: billing integration

- Make Billing Runtime read committed usage facts by `usage_record_id`.
- Move cost calculation to one explicit billing path.
- Keep funding settlement separate from usage recording.

Phase 4: distribution channel reporting

- Aggregate committed usage facts by `distribution_channel_id`, channel, model, and time window.
- Revenue share remains a later billing/reconciliation iteration.
