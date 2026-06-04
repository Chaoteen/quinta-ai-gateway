# Iteration 10.4C Usage Metering Relay Integration Audit

Branch: `feat/iteration-10-4-usage-metering-foundation`

This audit reviews where Usage Metering should enter the relay hot path after the 10.4B model and service foundation. It does not change production code.

## Scope

Reviewed modules:

- `relay/`
- `relay/common/`
- `relay/common_handler/`
- `relay/channel/`
- `service/`
- `model/`

Focus files and objects:

- `relay_adaptor.go`
- `compatible_handler.go`
- `responses_handler.go`
- `audio_handler.go`
- `relay_task.go`
- `text_quota.go`
- `quota_engine.go`
- `usage_metering.go`
- `relay_info.go`
- `dto.Usage`

## Current Synchronous Text Relay Chain

Current `TextHelper()` chain:

```text
Request
  -> RelayInfo.InitChannelMeta()
  -> request type validation
  -> model mapping
  -> stream_options normalization
  -> adaptor selection and Init()
  -> request conversion / pass-through body
  -> adaptor.DoRequest()
  -> HTTP status handling
  -> adaptor.DoResponse()
  -> dto.Usage
  -> PostTextConsumeQuota() or PostAudioConsumeQuota()
  -> quota calculation
  -> model.UpdateUserUsedQuotaAndRequestCount()
  -> model.UpdateChannelUsedQuota()
  -> SettleBilling()
  -> BillingSession / FundingSource
  -> RecordConsumeLog()
```

Current `ResponsesHelper()` chain is structurally similar:

```text
Request
  -> RelayInfo.InitChannelMeta()
  -> responses request validation
  -> model mapping
  -> adaptor selection and Init()
  -> request conversion / pass-through body
  -> adaptor.DoRequest()
  -> HTTP status handling
  -> adaptor.DoResponse()
  -> dto.Usage
  -> PostTextConsumeQuota() or PostAudioConsumeQuota()
  -> BillingSession / FundingSource
  -> Log
```

`ClaudeHandler` also follows the same essential shape:

```text
Claude request
  -> RelayInfo
  -> adaptor.DoRequest()
  -> adaptor.DoResponse()
  -> dto.Usage
  -> PostTextConsumeQuota()
```

Recommended insertion point for 10.4D:

```text
adaptor.DoResponse()
  -> dto.Usage
  -> UsageMetering.NormalizeUsage()
  -> UsageMetering.CommitUsageFact()
  -> existing PostTextConsumeQuota() / PostAudioConsumeQuota()
```

Reasoning:

- `dto.Usage` is already normalized by channel handlers.
- `RelayInfo` already contains ownership, channel, request, model, and pricing context.
- This location avoids duplicating provider-specific usage parsing in every channel.
- Existing billing remains unchanged while Usage Metering is introduced as a fact writer.

## Quota Runtime And Usage Metering Relationship

Quota Runtime owns:

- `CheckQuota()`
- `ReserveQuota()`
- `CommitUsage()`
- `RollbackReservation()`

Usage Metering owns:

- `NormalizeUsage()`
- `CommitUsageFact()`

Current overlap:

- `QuotaEngine.CommitUsage()` writes a committed `QuotaUsageRecord`.
- `UsageMeteringService.CommitUsageFact()` also writes a committed `QuotaUsageRecord`.

This creates a real double-write risk if both are called independently for the same relay request.

Recommended 10.4D strategy:

1. Use Usage Metering as the only writer of the final committed usage fact in relay dry integration.
2. Do not call `QuotaEngine.CommitUsage()` from relay in 10.4D.
3. Preserve `reservation_id` on the Usage Metering committed record so Quota Runtime state can be correlated.
4. Leave reservation status finalization deferred until a later Quota Runtime integration decision.

Alternative future strategy:

- Refactor `QuotaEngine.CommitUsage()` to accept a metering payload and internally call Usage Metering once.
- That would make Quota Runtime the orchestrator and Usage Metering the fact normalizer/writer.
- This should not be done in 10.4D unless the team is ready to modify Quota Runtime.

Non-goal for 10.4D:

- Do not have both services create committed rows for the same `request_id` / `reservation_id`.

## Billing Compatibility

Current `PostTextConsumeQuota()` already performs:

- usage semantic interpretation
- token/cache/audio/image/tool quota calculation
- tiered billing expression settlement
- user used quota and request count update
- channel used quota update
- `SettleBilling()`
- `BillingSession.Settle()`
- `FundingSource.Settle()`
- wallet mutation through wallet funding
- token quota adjustment through `BillingSession`
- legacy subscription amount mutation through `SubscriptionFunding`
- consume log creation

Therefore Usage Metering must not:

- call `SettleBilling()`
- call `BillingSession`
- call `FundingSource`
- change `User.Quota`
- change `Token.RemainQuota`
- change `UserSubscription.amount_used`
- create or refund `SubscriptionPreConsumeRecord`

Recommended 10.4D billing compatibility rule:

```text
Usage Metering writes facts.
Existing billing continues to settle from PostTextConsumeQuota().
No billing behavior changes in 10.4D.
```

Avoiding double counting:

- `CommitUsageFact()` writes a fact only.
- `PostTextConsumeQuota()` remains the only settlement path.
- Future 10.5 Billing integration should switch Billing to read committed usage facts, but that is a separate migration.

## Failure Path Recommendations

### CheckQuota Fails

Recommendation:

- Do not call provider.
- Do not create a usage fact.
- Return a quota denial error.
- No rollback is needed if no reservation exists.

10.4D note:

- If 10.4D does not yet call Quota Runtime, this path remains theoretical and should stay out of scope.

### ReserveQuota Succeeds But Provider Request Fails

Recommendation:

- Call `RollbackReservation()`.
- Do not write committed usage fact.
- Optionally write a non-committed diagnostic event in a later iteration, but not in 10.4D.

10.4D dry integration:

- If reservation is not integrated yet, no rollback path should be added.

### Provider Succeeds But Usage Is Empty

Current behavior:

- `PostTextConsumeQuota()` treats zero total tokens as non-billable and logs an error message.
- Some stream handlers estimate usage from response text if upstream usage is absent.

Recommendation:

- If `dto.Usage` is nil or total tokens are zero and no reliable estimate exists, do not write a committed usage fact.
- If an existing channel handler already returns estimated usage, write the fact with `usage_source = estimated`.
- Do not invent new estimation policy in relay integration.

### Provider Succeeds But CommitUsageFact Fails

Recommended 10.4D behavior:

- Treat metering failure as non-fatal for the user response.
- Continue existing `PostTextConsumeQuota()` billing path.
- Log a structured error with `request_id`, `reservation_id`, `user_id`, `tenant_id`, `channel_id`, and model.

Reasoning:

- 10.4D is a dry integration.
- Making user responses fail because of metering write failure would change production behavior.

### Billing Succeeds But Usage Metering Fails

Recommended 10.4D behavior:

- Billing remains authoritative for current balance mutation.
- Metering failure is logged for audit gap follow-up.
- Do not retry billing.
- Do not correct wallet/token/subscription balances from Usage Metering.

Future 10.5:

- Billing should read committed usage facts before settlement. At that point metering failure would become a pre-settlement error, not a post-settlement audit gap.

### Streaming Interrupted

Current behavior:

- Stream handlers aggregate usage while reading chunks.
- Some handlers fall back to `ResponseText2Usage()` if final provider usage is missing.
- If a handler returns an error, relay returns before `PostTextConsumeQuota()`.

Recommendation:

- If `adaptor.DoResponse()` returns an error, do not write committed usage fact.
- If `adaptor.DoResponse()` returns a usable estimated `dto.Usage`, write with `usage_source = estimated`.
- Do not write per-chunk usage facts in 10.4D.

### Client Abort

Recommendation:

- If provider response processing cannot complete, do not write committed usage fact.
- If the provider completed and the handler produced a final `dto.Usage`, writing a committed fact is acceptable even if the client disconnected late.
- Use the same rule as existing billing path: only meter what the handler returns as final usage.

## Streaming Path Review

### OpenAI-Compatible Streaming

Current behavior:

- If stream usage is present, handlers use it.
- Audio stream models may extract usage from the second-last SSE chunk.
- If stream usage is missing, handlers estimate from response text and estimated prompt tokens.

Recommendation:

- Include OpenAI-compatible streaming in 10.4D only after non-stream text path is stable.
- Use `usage_source = upstream` when stream usage is present.
- Use `usage_source = estimated` when handler fallback was used, if that can be detected.

Implementation caveat:

- `dto.Usage` does not always explicitly mark estimated usage today. A small helper may be needed later to infer or set source.

### Claude Streaming

Current behavior:

- `message_start` and `message_delta` provide usage.
- The handler preserves Anthropic semantic fields.
- If final usage is incomplete, it estimates missing completion/prompt tokens from response text.

Recommendation:

- Include Claude streaming only if metering captures `usage_semantic = anthropic`.
- Preserve cache read/write values in metadata or future richer metering fields.
- Avoid converting Anthropic usage into OpenAI totals for authoritative metering.

### Gemini Streaming

Current behavior:

- Gemini stream chunks may include `usageMetadata`.
- The handler maps usage metadata into `dto.Usage`.
- If usage is missing but text was received, it estimates usage from response text.
- If no response chunks were received, usage can remain empty.

Recommendation:

- Include Gemini streaming only after non-stream Gemini usage tests are stable.
- Use `usage_source = upstream` when `usageMetadata` was present.
- Use `usage_source = estimated` when fallback usage is used.
- Skip committed facts for empty usage.

### 10.4D Streaming Recommendation

Recommended scope:

- Start with synchronous text and Responses non-stream path.
- Include streaming only as a second step in the same iteration if tests prove the usage source can be classified.

Reasoning:

- Streaming has provider-specific fallback rules.
- A wrong committed usage fact is worse than no dry-run fact during initial integration.

## Audio, Realtime, And Task

### Audio

Audio routes share the same post-response shape:

```text
adaptor.DoResponse()
  -> dto.Usage
  -> PostAudioConsumeQuota() or PostTextConsumeQuota()
```

Recommendation:

- Do not include audio in the first 10.4D pass.
- Add audio after text metering is stable because audio billing uses audio-specific token details and ratios.

### Realtime / WebSocket

Current behavior:

- Realtime usage can be accumulated and pre-consumed during the websocket session.
- It may settle multiple local usage chunks.

Recommendation:

- Exclude Realtime/WebSocket from 10.4D.
- Design session-level idempotency separately.
- Do not use one text-style request fact for a multi-event websocket session.

### Async Task

Current behavior:

- Task submission uses per-call pricing, estimated ratios, `Task`, and `TaskPrivateData.BillingContext`.
- Task final state can adjust billing later.

Recommendation:

- Exclude async task metering from 10.4D.
- Task usage facts should have a task-specific lifecycle and probably reference `task_id`.

## Idempotency

Important identifiers:

- `request_id`: request-level idempotency and correlation.
- `reservation_id`: quota reservation correlation.
- `usage_record_id`: committed fact identity.

Current risk:

- `CommitUsageFact()` currently writes a new row each time it is called.
- There is no database unique constraint on `request_id + status` or `reservation_id + status`.
- Calling it twice for the same request can create multiple committed usage facts.

Recommended 10.4D policy:

- Introduce a relay-level idempotency wrapper before using `CommitUsageFact()` in hot path.
- Lookup existing committed usage fact by `reservation_id` when present.
- Otherwise lookup by `request_id`, `user_id`, and `status = committed`.
- If found, return existing record and do not insert another.

Recommended future index:

- Cross-database-compatible index on `request_id`, `status`.
- Optional unique constraint only after product policy confirms one committed fact per request.

One-request one-fact policy:

- For synchronous text relay, one `request_id` should produce at most one committed usage fact.
- Realtime and task flows are excluded because they may require multiple usage facts per logical session/task.

## Performance Impact

10.4D adds at least one database write to the relay hot path.

Risks:

- Increased response latency.
- Additional DB pressure on high-volume text traffic.
- Transaction contention if coupled with quota reservation updates.
- Retry storms if metering write failures are retried synchronously.

Recommendation for 10.4D:

- Use a synchronous write for the dry committed fact so tests and audit behavior are deterministic.
- Do not retry synchronously more than once.
- Do not block the user response on metering failure.
- Log failures with correlation identifiers.
- Keep writes compact: one committed fact per request.
- Do not write per stream chunk.

Future optimization:

- Add async queue only after the committed fact schema and idempotency are stable.
- If async queue is introduced, it must preserve tenant ownership and idempotency keys.

## Recommended 10.4D Development Scope

Recommended iteration name:

`Iteration 10.4D Usage Metering Relay Dry Integration`

Recommended scope:

- Add a small relay/service adapter that builds `UsageMeteringInput` from `RelayInfo` and `dto.Usage`.
- Insert dry metering after `adaptor.DoResponse()` and before `PostTextConsumeQuota()` for synchronous text non-stream paths.
- Do not change billing behavior.
- Do not call `QuotaEngine.CommitUsage()` from relay in this iteration.
- Add idempotency guard around committed usage fact creation.
- Add tests for:
  - OpenAI non-stream text.
  - Claude non-stream text.
  - Gemini non-stream text.
  - no double committed fact for same request.
  - metering failure does not double bill.
  - tenant/channel attribution preserved.

Explicitly defer:

- Billing reads from usage facts.
- Wallet/token/subscription settlement changes.
- Audio.
- Realtime/WebSocket.
- Async Task.
- Revenue share.
- Admin Console.

## Final Recommendation

Proceed to 10.4D only as dry integration.

The correct first relay insertion point is:

```text
usage, err := adaptor.DoResponse(...)
if err == nil:
    CommitUsageFactDryRun(RelayInfo, usage)
    PostTextConsumeQuota(...) / PostAudioConsumeQuota(...)
```

The dry integration must be non-balance-affecting and idempotent. Billing integration should wait for 10.5, where Billing can explicitly read one committed usage fact and settle once.

## 10.4D Implementation Notes

Implemented dry integration scope:

- Added a service-layer relay usage metering adapter that converts `RelayInfo + dto.Usage` into `UsageMeteringInput`.
- Inserted the dry metering call after `adaptor.DoResponse()` and before `PostTextConsumeQuota()` in synchronous text non-stream paths.
- Covered OpenAI-compatible text, Claude text, Gemini text, and OpenAI Responses text paths.
- Explicitly skipped audio branches and guarded all relay calls with `!info.IsStream`.

Actual insertion points:

- `relay/compatible_handler.go`
- `relay/claude_handler.go`
- `relay/gemini_handler.go`
- `relay/responses_handler.go`

Double-write avoidance:

- Relay dry integration calls `UsageMetering.NormalizeUsage()` and `UsageMetering.CommitUsageFact()`.
- Relay dry integration does not call `QuotaEngine.CommitUsage()`.
- The adapter checks `request_id + status=committed` before writing, so repeated dry integration for the same request returns the existing committed fact instead of inserting another row.

Failure behavior:

- `dto.Usage == nil` skips metering.
- Input normalization or committed fact write failures are logged and swallowed by the relay-facing helper.
- Existing `PostTextConsumeQuota()` still runs after metering failure.
- No provider response is blocked by Usage Metering failure in this dry integration.

Billing impact:

- No `BillingSession` calls are added.
- No `FundingSource` calls are added.
- No wallet mutation is added.
- No token quota mutation is added.
- No `UserSubscription.amount_used` mutation is added.
- Existing billing and settlement still run only through `PostTextConsumeQuota()`.

Tenant and channel attribution:

- The committed usage fact preserves `tenant_id`, `organization_id`, `department_id`, `distribution_channel_id`, `user_id`, `user_subscription_id`, `channel_id`, provider name, model name, and upstream model name from `RelayInfo`.
- Missing tenant ownership fails validation and is logged rather than defaulting to a cross-tenant record.

Deferred items:

- Streaming usage facts.
- Audio usage facts.
- Realtime/WebSocket usage facts.
- Async task usage facts.
- Billing reads from committed usage facts.
- Quota Runtime reservation finalization.
