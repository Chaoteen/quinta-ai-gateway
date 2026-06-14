# 宝塔面板部署 Quinta AI Gateway

本文档提供使用宝塔面板 Docker 功能部署 Quinta AI Gateway 的基础流程。部署前请先准备可用的 Quinta AI Gateway 镜像、数据库连接信息和运行环境变量。

## 步骤一：准备环境

1. 安装宝塔面板，并启用 Docker 管理能力。
2. 准备数据库：SQLite、MySQL 或 PostgreSQL。
3. 准备运行目录，例如 `/www/wwwroot/quinta-ai-gateway`。

## 步骤二：创建容器

在宝塔面板 Docker 管理界面创建容器，建议使用以下基础参数：

- 容器名称：`quinta-ai-gateway`
- 镜像：使用你的 Quinta AI Gateway 发布镜像
- 端口映射：按实际部署环境映射服务端口
- 重启策略：`always`
- 数据目录：挂载到持久化目录

## 步骤三：示例 compose

以下示例仅展示结构，请按实际镜像、端口和数据库配置调整：

```yaml
services:
  quinta-ai-gateway:
    image: your-registry/quinta-ai-gateway:latest
    container_name: quinta-ai-gateway
    restart: always
    ports:
      - "3000:3000"
    environment:
      - SQL_DSN=your_database_dsn
      - TZ=Asia/Shanghai
    volumes:
      - ./data:/data
```

## 步骤四：验证

1. 确认容器启动成功。
2. 打开部署域名或服务器地址。
3. 完成初始化配置。
4. 检查管理后台、渠道配置、计费基础能力和租户隔离功能。

## 相关文档

- 项目文档：`docs/`
- 产品重命名与商业化 UI 收尾：`docs/iteration_12_1/`
- 环境变量和部署参数以当前仓库配置为准。
