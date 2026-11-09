# API 接入指南

通过统一网关，为企业 Agent、业务系统和开发工具接入平台已开通的 AI 模型。以下以聊天补全接口为例，不同模型支持的协议和参数可能不同。

## 1. 准备账号、额度和 API Key

1. [注册账号](__REGISTER_URL__)并登录。普通账号初始余额为 0，注册不代表获得模型额度；企业账号及权限由平台协助开通。
2. 在[账单中心](__BILLING_URL__)核对额度。当前预览环境未启用真实在线收款，试用请先联系管理员配置资源和额度。
3. 在[API 密钥](__KEYS_URL__)为应用创建 Key，按需限制额度、有效期、模型和来源 IP，保存在服务端环境变量或密钥管理服务中。
4. 在[模型广场](__MODELS_URL__)了解模型，再通过下面的接口确认当前 Key 实际可访问的模型 ID。供应商品牌展示不代表已配置可调用渠道。

使用网关签发的 API Key，不要用登录密码、管理员令牌或上游供应商 Key 代替。不要在前端代码、截图、公共仓库和日志中暴露真实 Key。

## 2. Base URL 与鉴权

本网站的 API Base URL：

```text
__GATEWAY_BASE_URL__
```

客户端的 Base URL 填入以上地址，包含 `/v1`。手动拼接路径时不要重复添加 `/v1`，也不要丢掉部署前缀 `/quinta`。地址随网站部署路径生成，迁移域名后以本页面显示为准。

以下 Bash 示例中的 Key 为占位符，请在自己的终端替换。Windows 用户可使用 WSL 或 Git Bash。

```bash
export QUINTA_BASE_URL="__GATEWAY_BASE_URL__"
export QUINTA_API_KEY="YOUR_GATEWAY_API_KEY"
```

每次请求均通过 `Authorization: Bearer <网关 API Key>` 鉴权；JSON 请求还需携带 `Content-Type: application/json`。

## 3. 查询可用模型

```bash
curl --fail-with-body "$QUINTA_BASE_URL/models" \
  -H "Authorization: Bearer $QUINTA_API_KEY"
```

响应示意（模型 ID 是占位符，不代表已开通）：

```json
{"object":"list","data":[{"id":"MODEL_ID_FROM_YOUR_ACCOUNT","object":"model"}]}
```

复制 `data[].id` 作为后续 `model`，保持大小写一致。列表为空时，请让管理员检查渠道、Key 的模型限制和账号分组，不要直接猜测模型名称。

## 4. 第一次聊天请求

替换下面的模型 ID 后执行：

```bash
curl --fail-with-body "$QUINTA_BASE_URL/chat/completions" \
  -H "Authorization: Bearer $QUINTA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_ID_FROM_YOUR_ACCOUNT",
    "messages": [{"role": "user", "content": "请用一句话介绍你自己。"}],
    "stream": false
  }'
```

通常从 `choices[0].message.content` 读取聊天文本，`usage` 提供模型返回的用量信息。工具调用、推理参数及响应字段取决于模型。费用核对以平台记录和合同约定为准。

## 5. Python 示例（标准库，无第三方依赖）

先设置上述两个环境变量，并将 `QUINTA_MODEL` 设置为实际可用模型 ID。

```python
import json
import os
import urllib.error
import urllib.request

request = urllib.request.Request(
    os.environ["QUINTA_BASE_URL"].rstrip("/") + "/chat/completions",
    data=json.dumps({
        "model": os.environ["QUINTA_MODEL"],
        "messages": [{"role": "user", "content": "你好"}],
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

生产代码还需处理超时、网络异常和日志脱敏，不要记录完整密钥。

## 6. 流式输出（SSE）

```bash
curl --no-buffer --fail-with-body "$QUINTA_BASE_URL/chat/completions" \
  -H "Authorization: Bearer $QUINTA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "MODEL_ID_FROM_YOUR_ACCOUNT",
    "messages": [{"role": "user", "content": "你好"}],
    "stream": true
  }'
```

聊天流通常以 `data:` 事件逐段传输 `choices[].delta`，以 `[DONE]` 结束；流内也可能出现错误。不要把整个流当成单个 JSON。`stream_options` 等可选参数须确认目标模型支持后再添加。

## 7. 常用接口与能力边界

| 方法 | 路径（相对于 Base URL） | 用途 |
| --- | --- | --- |
| GET | `/models` | 查询当前 Key 可访问的模型 |
| POST | `/chat/completions` | 聊天、工具调用、流式输出，取决于模型能力 |
| POST | `/responses` | 支持 Responses 协议的模型及渠道 |
| POST | `/embeddings` | 已开通的向量模型 |

存在接口不代表所有模型都支持该协议。图片、视频、音频和 Seedance 等任务需先确认供应商、模型、任务接口及计价单位，不能直接套用聊天示例。异步任务还需按其协议查询状态和结果。

## 8. 错误排查与费用核对

| 现象 | 检查方向 |
| --- | --- |
| 400 | JSON 格式、模型 ID、目标模型支持的参数 |
| 401 | 网关 Key 是否携带、有效、过期或被停用 |
| 403 | 账号、权限、IP 限制、分组和余额；结合错误正文判断 |
| 404 或返回 HTML | Base URL 前缀、`/v1` 是否正确，路径是否重复拼接 |
| 429 | 账号、Key 或上游速率限制；查看错误正文并适当退避 |
| 5xx、无可用渠道或超时 | 渠道和上游状态；保留请求时间及请求标识交给管理员 |

不同上游及校验环节可能使用不同状态码，请同时查看错误正文，不能仅凭状态码判断是否扣费。参数和鉴权错误不应自动重试；超时、流中断或任务提交后的失败，应先核实处理状态与用量。平台未承诺推理请求自动去重，重复请求可能产生重复费用。

调用后在[使用日志](__LOGS_URL__)和[账单中心](__BILLING_URL__)核对模型、时间、用量及费用。输入、输出、缓存或视频任务可能采用不同计价方式，以实际价格和合同为准。开票可在[发票页面](__INVOICE_URL__)发起申请，当前仍需财务审核和人工办理。
