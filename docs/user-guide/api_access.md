# API 接入说明

Quinta AI Gateway 提供统一的 AI API 网关入口，用于连接多个上游模型供应商，并对请求进行鉴权、路由、计量和计费。

## 接入前准备

- 已获得 Quinta AI Gateway 访问地址。
- 已创建可用 API Key。
- 当前账号或租户有可用额度、订阅权益或管理员分配的访问权限。
- 管理员已配置模型渠道和可访问模型。

## 基础请求方式

将客户端 Base URL 配置为 Quinta AI Gateway 的网关地址，并使用 API Key 作为 Bearer Token。

```bash
curl https://your-gateway.example.com/v1/chat/completions \
  -H "Authorization: Bearer $QUINTA_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "your-model",
    "messages": [
      { "role": "user", "content": "你好" }
    ]
  }'
```

## 模型选择

可在模型广场或管理员提供的模型清单中查看可用模型。不同模型可能对应不同供应商、计费倍率和访问权限。

## 错误排查

- `401`：API Key 无效或未携带。
- `403`：当前账号没有访问权限。
- `429`：触发限流或额度不足。
- `5xx`：网关或上游服务异常，请查看使用日志和渠道状态。

## 日志与计费

每次请求会记录用量、Token、模型、供应商和计费结果。可在使用日志、额度与用量、账单中心进行核对。
