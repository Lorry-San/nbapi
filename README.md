# NBAPI

NBAPI 是基于 [New API](https://github.com/QuantumNous/new-api) 的二开版本，保留 OpenAI/Claude/Gemini 等多协议 AI API 网关能力，并加入适合自部署运营的管理功能。

## 主要改动

- 默认品牌名调整为 `NBAPI`，页脚、关于页和默认系统名同步改名。
- 保留上游来源说明：关于页显示“基于 New API”。
- 默认强制记录用户 IP，用户侧不可关闭。
- 支持设置其他用户为超级管理员。
- 超级管理员支持一键登录其他用户账号。
- classic 使用记录支持导出 CSV。
- 支付设置页移除合规确认锁定提示，兑换码、订阅、邀请返利和支付配置不再被该确认项阻断。
- 提供 GHCR 镜像和 Docker Compose 部署文件。

## 镜像

```text
ghcr.io/lorry-san/nbapi:custom
```

推送到 `codex/admin-usage-export-docker` 分支后，GitHub Actions 会自动构建并推送：

- `ghcr.io/lorry-san/nbapi:custom`
- `ghcr.io/lorry-san/nbapi:custom-<sha>`

## 快速部署

复制环境变量模板：

```bash
cp .env.docker.example .env.docker
```

编辑 `.env.docker`，至少修改：

```text
POSTGRES_PASSWORD=...
REDIS_PASSWORD=...
SESSION_SECRET=...
```

启动：

```bash
docker compose --env-file .env.docker -f docker-compose.github.yml pull new-api
docker compose --env-file .env.docker -f docker-compose.github.yml up -d
```

默认访问：

```text
http://localhost:3000
```

## 从旧部署更新

如果你已经有 New API/NBAPI 的 Docker Compose 部署，先备份数据库，再把应用服务镜像改为：

```text
ghcr.io/lorry-san/nbapi:custom
```

只更新应用容器：

```bash
docker compose pull new-api
docker compose up -d --no-deps new-api
```

不要执行 `docker compose down -v`，否则会删除数据库卷。

## 本地源码构建

```bash
docker compose --env-file .env.docker -f docker-compose.local.yml up -d --build
```

## 文档

- GHCR 部署说明：[docs/local-docker-deploy.zh-CN.md](docs/local-docker-deploy.zh-CN.md)
- 上游项目：[QuantumNous/new-api](https://github.com/QuantumNous/new-api)

## 许可证

本项目基于 New API 二次开发，遵循上游项目的 AGPL v3.0 许可证要求。
