# NBAPI Docker 部署

这套配置支持两种方式：

- `docker-compose.github.yml`：直接拉取 GitHub Actions 发布到 GHCR 的镜像。
- `docker-compose.local.yml`：在本机从当前源码构建镜像。

## 1. 准备环境变量

PowerShell:

```powershell
copy .env.docker.example .env.docker
notepad .env.docker
```

至少修改 `POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`SESSION_SECRET`。如果密码包含 `@`、`:`、`/`、`?` 等特殊字符，用在连接串里时需要先做 URL 编码。

默认镜像为：

```text
ghcr.io/lorry-san/nbapi:custom
```

## 2. 使用 GHCR 镜像启动

```powershell
docker compose --env-file .env.docker -f docker-compose.github.yml pull new-api
docker compose --env-file .env.docker -f docker-compose.github.yml up -d
```

启动后访问：

```text
http://localhost:3000
```

## 3. 本地源码构建启动

```powershell
docker compose --env-file .env.docker -f docker-compose.local.yml up -d --build
```

## 4. 常用命令

查看日志：

```powershell
docker compose --env-file .env.docker -f docker-compose.github.yml logs -f new-api
```

更新 GHCR 镜像：

```powershell
docker compose --env-file .env.docker -f docker-compose.github.yml pull new-api
docker compose --env-file .env.docker -f docker-compose.github.yml up -d --no-deps new-api
```

停止服务：

```powershell
docker compose --env-file .env.docker -f docker-compose.github.yml down
```

连数据库数据一起删除：

```powershell
docker compose --env-file .env.docker -f docker-compose.github.yml down -v
```
