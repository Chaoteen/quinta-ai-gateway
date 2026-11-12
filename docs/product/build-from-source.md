# 构建 Quinta AI Gateway 对应源码

在网站“开源许可与对应源码”页面打开部署版本的源码链接，下载该版本后构建，不使用占位前端替代实际产物。构建需要 Go 1.26.11 和 Bun 1.3.11；依赖按 go.sum 和前端锁文件校验。

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

## 本机 WSL 迁移检查（2026-11-12）

本项目可在 Windows 或 Linux 构建，并不要求使用 WSL。当前实际工作目录是
`C:\Users\boris\Documents\ChatGPT\中转站项目`，属于 Windows C 盘上的源码快照，当前没有 `.git`，所以无法提供本地 Git diff/status 或核实未提交修改清单；本轮没有覆盖业务源码。Windows Conda 位于 `D:\ProgramData\anaconda3`，不是项目的 Go/Bun 运行环境。

用户已将 Ubuntu 发行版存储迁移到 `D:\WSL\Ubuntu`。只读检查确认发行版名仍为 Ubuntu、WSL 2、注册 BasePath 为该目录，`ext4.vhdx` 文件存在。WSL 版本 2.6.1.0。该目录是虚拟磁盘存储目录，不是项目目录，不能加入项目 PATH 或 PYTHONPATH。

**挂载问题与后续冷启动：** 初查时，沙箱内返回 `Wsl/E_ACCESSDENIED`，正常权限上下文返回 `Wsl/Service/CreateInstance/MountDisk/HCS/E_ACCESSDENIED`。用户随后明确授权仅修正该文件的 Windows Owner。已核实当前 Windows 账户为 `QYZHAISTATION01\boris`，SID 与预期一致，进程不是管理员；操作前已将原 Owner、完整 SDDL 和发行版注册信息保存到私有 `.audit-tools/wsl-owner-before-20261112.json`。

执行了用户指定的单文件 `icacls /setowner` 命令，命令返回 Access is denied（退出码5），但随后读回 Owner 已是 boris、DACL 与修改前完全一致。因此只记录“目标 Owner 状态已达到”，不将该命令报告为成功执行，也无法仅据此判断是否存在同时发生的其他所有者操作。未递归授权、未修改 DACL 或磁盘内容。最终状态记录见私有 `.audit-tools/wsl-owner-after-20261112.json`。

确认所有发行版均 Stopped 且没有 WSL 宿主进程后，按授权执行 `wsl --shutdown`，再次确认 Stopped，再以同一 boris 非管理员身份执行冷启动。启动退出码0，`id` 为 UID/GID 1000、HOME 为 `/home/boris`；根目录 ext4 可读，`/mnt/c` 和 `/mnt/d` 为正常 Windows 盘挂载。冷启动记录见私有 `.audit-tools/wsl-cold-start-20261112.json`。该结果证明本次上下文的挂载失败已消失，不证明所有 Linux 开发依赖都已齐备。

未在项目脚本、部署配置及环境文档中找到旧 `AppData\Local\wsl` 路径或把新磁盘目录当作源码路径的引用。当前无项目级 `.vscode` / `.devcontainer` 配置；Windows 已安装 Remote WSL 扩展，但实际 IDE 连接尚未验证。Windows PATH 中没有 Go、Bun、Docker；Go/Bun 使用下述迁移前已存在的本地工具。这些既存情况不是本次迁移新引入的问题。当前项目没有本地 CUDA/PyTorch、Java 或 Android 编译需求，不依赖 Windows Conda 环境。

### 已验证的 Windows 命令

PowerShell 中使用已有工具与依赖，不安装、不升级：

```powershell
Set-Location 'C:\Users\boris\Documents\ChatGPT\中转站项目'
$quintaRoot = (Get-Location).Path
$quintaGo = Join-Path $quintaRoot '.audit-tools/patched-1.26.11/go/bin/go.exe'
$quintaBun = 'C:/Users/boris/AppData/Local/npm-cache/_npx/6b41d1ab8d6987af/node_modules/bun/bin/bun.exe'
$env:GOTOOLCHAIN = 'local'
$env:GOMODCACHE = Join-Path $quintaRoot '.audit-tools/gomodcache'
$env:GOCACHE = Join-Path $quintaRoot '.audit-tools/gocache'
& $quintaGo test -p 2 ./common ./model ./service ./controller ./middleware ./router ./pkg/billingexpr
& $quintaGo build -p 2 -o .audit-tools/quinta-migration-check.exe .
Set-Location web/default
$env:VITE_BASE_PATH = '/quinta'
& $quintaBun run build:check
# 本地开发前端，另开终端运行后端：
$env:VITE_BASE_PATH = ''
$env:VITE_REACT_APP_SERVER_URL = 'http://localhost:3000'
& $quintaBun run dev --port 3001
```

已验证 Go 1.26.11、Bun 1.3.11、Node 22.18.0、Git 2.49.0.windows.1。上述七个 Go 包组测试、后端编译、默认前端 TypeScript 检查与生产构建通过。已有 `web/classic/dist` 被正常嵌入后端；本轮未重新构建 classic。

后端启动另用 PowerShell 终端，从**隔离目录**运行，不读取项目根目录可能存在的生产 `.env`：

```powershell
$quintaRoot = 'C:\Users\boris\Documents\ChatGPT\中转站项目'
$quintaRun = Join-Path $quintaRoot '.audit-tools/local-dev'
New-Item -ItemType Directory -Force $quintaRun | Out-Null
$env:APP_ENV = 'development'
$env:LISTEN_ADDRESS = '127.0.0.1'
$env:PORT = '3000'
$env:SQL_DSN = ''
$env:LOG_SQL_DSN = ''
$env:REDIS_CONN_STRING = ''
$env:SQLITE_PATH = Join-Path $quintaRun 'development.db'
$env:SESSION_SECRET = [guid]::NewGuid().ToString('N')
$env:CRYPTO_SECRET = [guid]::NewGuid().ToString('N')
$env:BOOTSTRAP_TOKEN = [guid]::NewGuid().ToString('N')
$env:BOOTSTRAP_ROOT_PASSWORD = Read-Host '本地首次管理员密码，至少16位' -MaskInput
Push-Location $quintaRun
try { & (Join-Path $quintaRoot '.audit-tools/quinta-migration-check.exe') }
finally { Pop-Location }
```

上面的命令需使用空闲的本地 3000/3001 端口，不终止占用端口的现有任务；初始化密码与临时密钥只用于开发，不用于正式部署。不要在此开发终端加载生产凭据。已实际完成一次回环动态端口、空白临时 SQLite、独立随机凭据的启动测试，`/api/status` 返回成功；随后仅关闭了本次创建的测试进程并清理其临时目录，没有连接生产数据库或模型服务。

### Ubuntu 与 Docker 待验证项

普通、非管理员 PowerShell 中启动 Ubuntu：

```powershell
wsl -d Ubuntu
```

Ubuntu 中已核实用户为 boris、HOME 为 `/home/boris`。当前项目的实际访问路径为：

```bash
cd '/mnt/c/Users/boris/Documents/ChatGPT/中转站项目'
```

没有假定 `/home` 下另有网关项目副本。Ubuntu 的 `/usr/local/go/bin/go` 实际版本是1.25.1；以 `GOTOOLCHAIN=local` 执行项目 `go test ./common`，明确被 go.mod 的1.26.11最低版本要求阻止，没有下载新工具链。Linux Bun未找到；Git 2.43.0、Python 3.12.3可执行；Node现有安装路径是 `/home/boris/.nvm/versions/node/v24.13.0/bin/node`。这些依赖版本与PATH差异不能据此归因于磁盘迁移。`/etc/wsl.conf` 原有 `appendWindowsPath=false` 保持不变，避免混入 Windows Conda或其他 Windows 工具；没有将虚拟磁盘目录加入 PATH。

Ubuntu有 Docker CLI 29.1.3，但不存在 `/var/run/docker.sock`，PID1 为 `init(Ubuntu)`，没有可用daemon。本轮未擅自启用systemd、启动Docker或升级Go/Bun。因此 Linux 原生构建、Linux前端安装、Docker项目容器、独立MySQL/PostgreSQL/Redis联调和 VS Code Remote WSL 连接均未验证；隔离SQLite测试不能替代这些检查。当前推荐使用上面已通过验证的 Windows 工具链继续本项目开发。没有因休眠关闭而修改项目配置。

另外修正了 `docker-compose.dev.yml` 的既存说明错误：默认前端目录应为 `web/default`，而非 `web`。这是文档路径纠正，与磁盘迁移无关；没有启动容器。
