# 构建 Quinta AI Gateway 对应源码

在网站“开源许可与对应源码”页面打开部署版本的源码链接，下载该版本后构建，不使用占位前端替代实际产物。构建需要 Go 1.26.10 和 Bun 1.3.11；依赖按 go.sum 和前端锁文件校验。

以下为 Linux 示例：

```bash
cd web/default
bun install --frozen-lockfile
VITE_BASE_PATH=/quinta bun run build
bun run typecheck
cd ../classic
bun install --frozen-lockfile
bun run build
cd ../..
CGO_ENABLED=0 GOTOOLCHAIN=local go build -trimpath -o quinta .
```

`/quinta` 是当前默认前端的部署前缀，根目录部署时将其设为空字符串。classic 前端尚未适配此子路径，当前服务应选择 default 主题。实际前端产物由 Go embed 编入可执行文件。

发布时随程序交付 LICENSE、NOTICE、THIRD-PARTY-LICENSES.md 和实际源提交标识。运行参数和环境变量参考 .env.example 及 deploy/ecs 模板；不要把生产密钥、商户私钥、数据库内容或真实客户资料包含在源码或镜像中。

## 回归检查

```bash
go test ./common ./relay/... ./pkg/billingexpr ./service ./controller ./model ./router .
cd web/default
bun test scripts/base-path.test.mjs scripts/base-path-build.test.mjs scripts/usage-log-access.test.mjs
```

SQLite、MySQL、PostgreSQL 并发测试及漏洞扫描参见 .github/workflows/commercial-security.yml。CI、依赖扫描和测试通过都不等于获得商业安全认证；当前已知未完成的业务流程见 commercial-release-plan.md。
