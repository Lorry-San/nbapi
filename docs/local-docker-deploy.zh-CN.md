# 本地 Docker 镜像部署

这套配置会从当前源码构建镜像，包含本地修改后的后端和两套前端构建产物。

## 1. 准备环境变量

PowerShell:

```powershell
copy .env.docker.example .env.docker
notepad .env.docker
```

至少修改 `POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`SESSION_SECRET`。密码如果包含 `@`、`:`、`/`、`?` 等字符，需要 URL 编码后再用于连接串。

## 2. 构建并启动

从当前目录源码构建：

```powershell
docker compose --env-file .env.docker -f docker-compose.local.yml up -d --build
```

从 GitHub 分支远程构建：

```powershell
docker compose --env-file .env.docker -f docker-compose.github.yml up -d --build
```

启动后访问：

```text
http://localhost:3000
```

## 3. 常用命令

查看日志：

```powershell
docker compose --env-file .env.docker -f docker-compose.local.yml logs -f new-api
```

更新代码后重新构建：

```powershell
docker compose --env-file .env.docker -f docker-compose.local.yml up -d --build new-api
```

停止服务：

```powershell
docker compose --env-file .env.docker -f docker-compose.local.yml down
```

连数据库数据一起删除：

```powershell
docker compose --env-file .env.docker -f docker-compose.local.yml down -v
```
