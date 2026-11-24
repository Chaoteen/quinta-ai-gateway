# HSTS 与独立审计接入

本轮提供可部署的 HSTS 条件控制和 ECS 审计导出适配器。云端未配置时，不能把本机日志称为独立审计，也不能把本机 fsync 当作云端接收成功。

## HSTS

当前入口 https://47.110.42.3/quinta/ 使用 IP，浏览器不支持对 IP 启用 HSTS。保持 SECURITY_HSTS_HOST 未设置。
正式上线使用有有效 HTTPS 证书的专用主机名后，在私有环境文件设置：

```dotenv
SECURITY_HSTS_HOST=gateway.example.com
SECURITY_HSTS_MAX_AGE=300
TRUSTED_PROXIES=127.0.0.1,::1
```

替换示例域名，只信任实际代理。反向代理覆盖 Host、X-Forwarded-Proto，禁止客户端直连应用端口。HSTS 影响整个域名，不能限定 /quinta/。先验证 HTTP 到 HTTPS 的跳转、全部 HTTPS 页面和证书续期，再逐步增加到 31536000；本轮不使用 includeSubDomains 或 preload。收到头的浏览器在 max-age 内不会因为服务器回滚而立即解除约束。
中间件仅在 TLS 连接或明确可信代理声明 HTTPS、请求主机与配置一致且配置非 IP 时发送 HSTS。

## 当前 ECS 导出

scripts/export_security_audit.py 是当前 SQLite ECS 部署适配器；应用的 MySQL/PostgreSQL 支持不受影响，但此运维适配器不可直接用于这两种数据库，切换数据库前需提供对应 CDC/导出方案。

- 单独 systemd 定时任务每分钟运行，使用只读 SQLite 连接读取已提交的 credential_reset_audits 与 supplier_audits。
- 不查询密钥、种子、恢复码、请求正文、供应商备注；输出字段白名单、稳定 event_id，支持下游去重。
- 文件持久化成功后才推进断点。中途崩溃可能重发；下游按 event_id 去重。检测到源表最大 ID 小于断点时停止，交由运维审核恢复。不能检测任意同 ID 数据篡改或所有数据库恢复情形。
- 首次启用导出所有现存记录，每次每表最多 1000 行。每次产生心跳；查询失败、输出失败不生成成功心跳。
- /var/log/quinta-audit 为 root:root 0700；应用用户不能修改这些导出文件。每天一个 audit-YYYY-MM-DD.jsonl，checkpoint.json 为本地导出进度。导出器没有网络权限。
- 不自动删除档案。接入云端并完成接收与保留策略验证后，再制定本地清理策略；监控磁盘容量。不要用重新生成断点掩盖故障。
- root 被攻破仍可修改本机证据。此方案只有完成异机/云端采集才具备独立存储边界。

## 接入阿里云 SLS（待资源准备）

按官方文档在 ECS 同地域创建 Project 和独立 Logstore；安装/注册 LoongCollector，配置 JSON 单行文件采集：目录 /var/log/quinta-audit，文件 audit-*.jsonl。不采集 checkpoint.json、应用 runtime.env 或其他配置。采集器使用独立权限；应用无查询、删除 Logstore 或修改保留时间的权限。创建资源的管理员与运行时写入身份分离，避免长期 AccessKey 放入源码或应用界面。

后台启动本轮导出 timer 并不代表已安装采集器。必须看到 SLS 中的心跳和真实审计事件，验证重启后续传、event_id 去重查询及无敏感字段，才记录远端接通。

建议告警：心跳超过 5 分钟未到达、管理员凭据重置、供应商暂停/渠道变更、采集失败、磁盘剩余不足 20%。心跳告警和投递监控要配置在 SLS 或另一台监控设备，避免 ECS 停机时一起失效。日志保留期限由安全/法务确认；可先评估至少 180 天及费用。OSS WORM 锁定具有不可逆约束，本轮不自动创建或锁定。

当前覆盖仅上述两类事务审计。登录失败、普通配置变更、全部资金操作、云账号操作及网络攻击日志仍需补齐事件源/ActionTrail/入口日志；不能据此声明完成等保测评。

部署：安装脚本到 /opt/quinta-audit/export_security_audit.py，安装 deploy/ecs/quinta-audit-export.{service,timer} 到 /etc/systemd/system/，执行 systemctl daemon-reload，然后 systemctl enable --now quinta-audit-export.timer 和 systemctl start quinta-audit-export.service。通过 systemctl status quinta-audit-export.service 和 timer 检查本地运行；云端接收必须另行验证。

参考：
- https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers/Strict-Transport-Security
- https://www.alibabacloud.com/help/en/sls/getting-started
- https://www.alibabacloud.com/help/en/sls/install-run-upgrade-and-uninstall-logtail
