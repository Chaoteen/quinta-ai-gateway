# API access guide

Connect enterprise agents, applications and developer tools to enabled models through the gateway. These examples use chat completions; protocols and parameters depend on the model and configured channel.

## 1. Account, quota and API key

1. [Register](__REGISTER_URL__) and sign in. New personal accounts start with zero balance. Ask the platform to provision enterprise access.
2. Check available quota in [Billing](__BILLING_URL__). Real online payments are not enabled in this preview; contact the administrator for resources and trial quota.
3. Create an application-specific key in [API Keys](__KEYS_URL__). Set quota, expiry, model and IP restrictions as needed. Store the key in a server-side environment variable or secret manager.
4. Browse the [model catalog](__MODELS_URL__), then query the model endpoint below to obtain IDs accessible to your key. Provider logos do not imply enabled channels.

Use a gateway-issued key, never your login password, administrator credential or upstream provider key. Keep real keys out of browser code, screenshots, public repositories and logs.

## 2. Base URL and authentication

The API Base URL for this deployment is:

```text
__GATEWAY_BASE_URL__
```

Include `/v1` once and preserve the deployment prefix. The address shown here follows the current website origin and mount path. The following Bash examples also work in WSL or Git Bash; replace the placeholder privately.

```bash
export QUINTA_BASE_URL="__GATEWAY_BASE_URL__"
export QUINTA_API_KEY="YOUR_GATEWAY_API_KEY"
```

Send `Authorization: Bearer <gateway API key>` with each request and `Content-Type: application/json` with JSON bodies.

## 3. Discover available models

```bash
curl --fail-with-body "$QUINTA_BASE_URL/models" \
  -H "Authorization: Bearer $QUINTA_API_KEY"
```

Example only; the ID below is a placeholder:

```json
{"object":"list","data":[{"id":"MODEL_ID_FROM_YOUR_ACCOUNT","object":"model"}]}
```

Use an exact `data[].id` in the next request. If the list is empty, ask the administrator to check enabled channels, model restrictions and account groups. Do not guess model IDs.

## 4. First chat completion

Replace the model ID before running:

```bash
curl --fail-with-body "$QUINTA_BASE_URL/chat/completions" \
  -H "Authorization: Bearer $QUINTA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_ID_FROM_YOUR_ACCOUNT",
    "messages": [{"role": "user", "content": "Introduce yourself in one sentence."}],
    "stream": false
  }'
```

Chat text is normally in `choices[0].message.content`; `usage` contains model-reported usage. Tool calls, reasoning parameters and response fields depend on the model. Reconcile charges against platform records and your contract.

## 5. Python example (standard library)

Set the two variables above and set `QUINTA_MODEL` to an enabled model ID. No third-party SDK is required.

```python
import json
import os
import urllib.error
import urllib.request

request = urllib.request.Request(
    os.environ["QUINTA_BASE_URL"].rstrip("/") + "/chat/completions",
    data=json.dumps({
        "model": os.environ["QUINTA_MODEL"],
        "messages": [{"role": "user", "content": "Hello"}],
        "stream": False,
    }).encode("utf-8"),
    headers={
        "Authorization": "Bearer " + os.environ["QUINTA_API_KEY"],
        "Content-Type": "application/json",
    },
    method="POST",
)
try:
    with urllib.request.urlopen(request, timeout=120) as response:
        result = json.load(response)
        print(result["choices"][0]["message"]["content"])
except urllib.error.HTTPError as error:
    print("HTTP", error.code, error.read().decode("utf-8"))
```

Production clients also need timeout handling, network error handling and log redaction. Never log complete keys.

## 6. Streaming (SSE)

```bash
curl --no-buffer --fail-with-body "$QUINTA_BASE_URL/chat/completions" \
  -H "Authorization: Bearer $QUINTA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_ID_FROM_YOUR_ACCOUNT",
    "messages": [{"role": "user", "content": "Hello"}],
    "stream": true
  }'
```

Read `data:` events incrementally, usually from `choices[].delta`, until `[DONE]`. Streams may contain errors. Do not parse the complete stream as one JSON object. Add options such as `stream_options` only when the target model supports them.

## 7. Endpoint reference

| Method | Path relative to Base URL | Purpose |
| --- | --- | --- |
| GET | `/models` | Models accessible to this key |
| POST | `/chat/completions` | Chat, streaming and tools where supported |
| POST | `/responses` | Channels and models supporting the Responses protocol |
| POST | `/embeddings` | Enabled embedding models |

An endpoint does not guarantee support by every model. Image, video, audio and Seedance integrations require confirmation of the provider, model, task protocol and pricing unit. Asynchronous tasks require their own status and result handling.

## 8. Troubleshooting and billing

| Symptom | Check |
| --- | --- |
| 400 | JSON body, model ID and supported parameters |
| 401 | Missing, invalid, expired or disabled gateway key |
| 403 | Account, permissions, IP restrictions, group and balance; read the body |
| 404 or HTML | Deployment prefix and `/v1`, missing or duplicated path segments |
| 429 | Account, key or upstream rate limits; read the body and back off |
| 5xx, no channel or timeout | Enabled channels and upstream availability; keep the request time and identifier |

Upstream services and validation layers may use different status codes. Always read the error body; status alone cannot determine billing. Do not retry invalid parameters or authentication failures automatically. After a timeout, interrupted stream or submitted task, check status and usage first. Inference requests are not promised automatic deduplication; repeated calls may create repeated charges.

Reconcile model, time, usage and charges in [Usage Logs](__LOGS_URL__) and [Billing](__BILLING_URL__). Input, output, cached content and video tasks may have different pricing units. Prices and contracts govern settlement. [Invoice requests](__INVOICE_URL__) currently require finance review and manual issuance.
