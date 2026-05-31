<div align="center">

![NBAPI](./web/default/public/logo.png)

# NBAPI

**基于 New API 的二开版 LLM 网关与运营管理系统**

<p align="center">
  <strong>简体中文</strong> |
  <a href="./README.en.md">English</a> |
  <a href="./README.zh_CN.md">New API 简体中文</a>
</p>

<p align="center">
  <a href="https://github.com/Lorry-San/nbapi/blob/main/LICENSE">
    <img src="https://img.shields.io/github/license/Lorry-San/nbapi?color=brightgreen" alt="license">
  </a>
  <a href="https://github.com/Lorry-San/nbapi/releases">
    <img src="https://img.shields.io/github/v/release/Lorry-San/nbapi?color=brightgreen&include_prereleases" alt="release">
  </a>
  <a href="https://github.com/Lorry-San/nbapi/pkgs/container/nbapi">
    <img src="https://img.shields.io/badge/docker-GHCR-blue" alt="docker">
  </a>
  <a href="https://github.com/QuantumNous/new-api">
    <img src="https://img.shields.io/badge/based%20on-New%20API-8A2BE2" alt="based on New API">
  </a>
</p>

<p align="center">
  <a href="#项目说明">项目说明</a> ·
  <a href="#快速开始">快速开始</a> ·
  <a href="#主要功能">主要功能</a> ·
  <a href="#docker-部署">Docker 部署</a> ·
  <a href="#从-new-api-迁移">从 New API 迁移</a> ·
  <a href="#许可说明">许可说明</a>
</p>

</div>

## 项目说明

NBAPI 是基于 [New API](https://github.com/QuantumNous/new-api) 的二次开发版本，保留 OpenAI、Claude、Gemini、Responses、Realtime、Rerank 等多协议 AI API 网关能力，并加入更适合自部署运营的管理功能。

> [!IMPORTANT]
> - 本项目适用于合法授权的 AI API 网关、模型分发、额度管理、使用统计、成本核算和私有化部署场景。
> - 使用者需要自行确保上游 API Key、模型服务、支付渠道、用户数据处理和公开运营行为符合所在地法律法规及上游服务条款。
> - 本项目仍基于 New API，用户界面中保留“基于 New API”的上游来源说明。

## 快速开始

### 使用 GHCR 镜像

```bash
docker pull ghcr.io/lorry-san/nbapi:1.0.0-rc4
```

### 使用 Docker Compose

```bash
git clone https://github.com/Lorry-San/nbapi.git
cd nbapi

cp .env.docker.example .env.docker
nano .env.docker

docker compose --env-file .env.docker -f docker-compose.github.yml pull nbapi
docker compose --env-file .env.docker -f docker-compose.github.yml up -d
```

部署完成后访问：

```text
http://localhost:3000
```

> [!TIP]
> 首次部署前请至少修改 `.env.docker` 中的 `POSTGRES_PASSWORD`、`REDIS_PASSWORD`、`SESSION_SECRET`。

## 主要功能

### NBAPI 二开能力

| 功能 | 说明 |
|---|---|
| 品牌调整 | 默认品牌改为 NBAPI，并保留“基于 New API”的上游来源说明 |
| Docker 镜像 | 提供可直接部署的 `ghcr.io/lorry-san/nbapi` 镜像 |
| CSV 导出 | classic 主题使用记录支持导出为 CSV |
| 强制 IP 记录 | 默认强制记录用户 IP，普通用户不可关闭 |
| 超级管理员管理 | 支持把其他用户设置为超级管理员 |
| 一键登录用户 | 超级管理员可一键登录其他用户账号，便于排查问题 |
| OpenAI API 版本路径 | OpenAI 兼容渠道可配置 `/v1`、`/v3` 或其他自定义版本路径 |
| default / classic 双主题 | default 与 classic 主题均保留 NBAPI 品牌和二开功能入口 |
| 合规声明 | 支付/充值前可要求用户确认合规声明，适合公开运营场景 |

### 运维监控与告警

| 功能 | 说明 |
|---|---|
| 运维监控看板 | 管理员和超级管理员可查看指定时间段内的请求量、Token 数、平均 QPS、平均 TPS、SLA、错误数 |
| 请求时延指标 | 支持平均请求时长、P95/P99 请求时长、首 Token 时间等关键性能指标 |
| 渠道观测 | 支持按渠道查看请求量、错误数、SLA、平均时长、首 Token 时间、渠道切换 |
| 上游错误列表 | 聚合展示上游错误、状态码、错误码、渠道、模型和错误内容，便于定位慢渠道和异常渠道 |
| 吞吐趋势 | 展示 QPS/TPS 趋势与渠道切换趋势 |
| 告警规则 | 支持创建阈值告警规则，内置错误率、成功率、P95/P99 延迟、CPU、内存等默认规则 |
| 告警事件 | 记录告警触发和恢复事件，保留规则、指标、当前值、持续时间和维度信息 |
| 邮件通知 | 告警触发/恢复可通过 SMTP 邮件通知已启用的管理员和超级管理员 |

### 继承自 New API 的核心能力

| 能力 | 说明 |
|---|---|
| 多模型网关 | 支持 OpenAI、Claude、Gemini、Azure、DeepSeek、Qwen 等上游 |
| 协议转换 | 支持 OpenAI Compatible、Claude Messages、Gemini 等格式转发 |
| 数据看板 | 提供额度、请求量、消费、日志等运营统计 |
| 渠道管理 | 支持模型映射、分组、倍率、优先级和失败重试 |
| 用户管理 | 支持用户、令牌、额度、兑换码、订阅和邀请奖励 |
| 充值体系 | 支持兑换码、易支付、Stripe、Creem、Waffo 等支付/充值方案 |

## 模型与接口支持

| 类型 | 说明 |
|---|---|
| OpenAI Compatible | `/v1/chat/completions`、Embeddings、Images、Audio 等兼容接口 |
| OpenAI Responses | Responses API 兼容转发 |
| OpenAI Realtime | Realtime API 兼容转发 |
| Claude | Claude Messages 格式 |
| Gemini | Google Gemini 格式 |
| Rerank | Cohere、Jina 等重排序模型 |
| 自定义上游 | 支持配置合法授权的自定义 API 地址和版本路径 |

## Docker 部署

### 推荐镜像

```text
ghcr.io/lorry-san/nbapi:1.0.0-rc4
ghcr.io/lorry-san/nbapi:main
```

### Compose 文件

| 文件 | 用途 |
|---|---|
| `docker-compose.github.yml` | 使用 GHCR 预构建镜像部署，推荐生产使用 |
| `docker-compose.local.yml` | 本地源码构建镜像 |
| `docker-compose.dev.yml` | 开发环境 |
| `docker-compose.yml` | 默认 Compose 配置 |

### 常用环境变量

| 变量 | 说明 |
|---|---|
| `SESSION_SECRET` | 会话密钥，多机部署必须固定 |
| `POSTGRES_PASSWORD` | PostgreSQL 密码 |
| `REDIS_PASSWORD` | Redis 密码 |
| `SQL_DSN` | 外部数据库连接串，使用内置 Postgres 时无需手动配置 |
| `REDIS_CONN_STRING` | 外部 Redis 连接串，使用内置 Redis 时无需手动配置 |

## 从 New API 迁移

如果你已经部署了官方 New API，可以直接替换应用镜像：

```text
ghcr.io/lorry-san/nbapi:1.0.0-rc4
```

推荐流程：

```bash
# 1. 先备份数据库和 docker-compose 文件

# 2. 修改 compose 中应用服务镜像为 NBAPI
image: ghcr.io/lorry-san/nbapi:1.0.0-rc4

# 3. 只更新应用容器
docker compose pull
docker compose up -d --no-deps nbapi
```

> [!WARNING]
> 不要执行 `docker compose down -v`，否则可能删除数据库卷。

迁移后已有用户不会被删除，NBAPI 会使用原有数据库继续运行。强制 IP 记录、合规声明、运维监控、告警规则等 NBAPI 行为会随新版本生效。

## 文档

| 内容 | 链接 |
|---|---|
| NBAPI Release | [GitHub Releases](https://github.com/Lorry-San/nbapi/releases) |
| NBAPI GHCR 镜像 | [ghcr.io/lorry-san/nbapi](https://github.com/Lorry-San/nbapi/pkgs/container/nbapi) |
| 本地 Docker 部署说明 | [docs/local-docker-deploy.zh-CN.md](./docs/local-docker-deploy.zh-CN.md) |
| 上游 New API | [QuantumNous/new-api](https://github.com/QuantumNous/new-api) |
| New API 官方文档 | [docs.newapi.pro](https://docs.newapi.pro/) |

## 相关项目

| 项目 | 说明 |
|---|---|
| [QuantumNous/new-api](https://github.com/QuantumNous/new-api) | 本项目上游 |
| [One API](https://github.com/songquanpeng/one-api) | New API 的原始项目基础 |
| [Midjourney-Proxy](https://github.com/novicezk/midjourney-proxy) | Midjourney 接口支持 |

## 许可说明

本项目基于 [New API](https://github.com/QuantumNous/new-api) 二次开发，遵循 [GNU Affero General Public License v3.0](./LICENSE)。

根据 AGPLv3 及上游附加条款，修改版本需要保留上游项目来源和界面归属说明。NBAPI 在用户界面中保留“基于 New API”的可见链接与说明。

---

<div align="center">

### Thank you for using NBAPI

**[Latest Release](https://github.com/Lorry-San/nbapi/releases)** ·
**[GHCR Image](https://github.com/Lorry-San/nbapi/pkgs/container/nbapi)** ·
**[Based on New API](https://github.com/QuantumNous/new-api)**

<sub>基于 New API 二次开发</sub>

</div>
