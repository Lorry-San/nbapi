<div align="center">

<img src="https://pic.itxiaohui.top/20260501/34f446414f26798b1b4cce1fe30c6307.png" alt="NBAPI Logo" width="200"/>

# NBAPI

**🚀 企业级 AI 网关与运营管理平台**

*基于 New API 的强化版本，提供监控、告警与运营增强功能*

[![License](https://img.shields.io/github/license/Lorry-San/nbapi?color=brightgreen)](https://github.com/Lorry-San/nbapi/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/Lorry-San/nbapi?color=brightgreen&include_prereleases)](https://github.com/Lorry-San/nbapi/releases)
[![Docker](https://img.shields.io/badge/docker-GHCR-blue)](https://github.com/Lorry-San/nbapi/pkgs/container/nbapi)
[![Based on New API](https://img.shields.io/badge/based%20on-New%20API-8A2BE2)](https://github.com/QuantumNous/new-api)

[📖 文档](#文档) • [🎯 核心特性](#核心特性) • [🚀 快速开始](#快速开始) • [🐳 Docker 部署](#docker-部署) • [🌏 English](../README.md)

</div>

---

## 🎯 NBAPI 是什么？

NBAPI 是基于 [New API](https://github.com/QuantumNous/new-api) 的**高级 AI API 网关与运营管理系统**，提供：

✨ **统一 API 网关**，支持 50+ AI 提供商（OpenAI、Claude、Gemini、DeepSeek、通义千问等）  
📊 **实时监控与告警**，包含 SLA 跟踪、QPS/TPS 指标、P95/P99 延迟  
🔄 **协议转换**，支持 OpenAI、Claude Messages、Gemini、Responses、Realtime API  
💰 **额度管理**，集成支付网关与使用量统计  
🛡️ **企业级运营**，包括合规声明、IP 记录、管理员工具

> [!IMPORTANT]
> **仅限合法授权使用**  
> NBAPI 适用于**授权的 AI API 分发、私有化部署、额度管理和成本追踪**场景。使用者需确保上游 API Key、模型服务、支付渠道、用户数据处理及公开运营行为符合当地法律法规及上游服务条款。

---

## ✨ 核心特性

| 分类 | 功能 |
|----------|----------|
| **📊 监控与告警** <br/> *(NBAPI 增强)* | • 实时仪表盘：SLA、QPS/TPS、P95/P99 延迟 <br/> • 渠道级可观测性与错误追踪 <br/> • 自定义告警规则（错误率、延迟阈值等） <br/> • SMTP 邮件通知 <br/> • 请求时延指标（平均、P95/P99、首 Token 时间） <br/> • 上游错误聚合与慢渠道识别 |
| **🔧 管理与运营** <br/> *(NBAPI 增强)* | • 超级管理员管理与一键登录其他用户 <br/> • 强制 IP 记录以满足合规要求 <br/> • 使用记录 CSV 导出（Classic 主题） <br/> • 自定义 OpenAI API 版本路径（`/v1`、`/v3` 等） <br/> • 支付前强制合规声明确认 <br/> • 双主题界面（Default 现代 UI + Classic 管理面板） |
| **🤖 模型支持** | • **50+ 提供商**：OpenAI、Claude、Gemini、Azure OpenAI、DeepSeek、通义千问、豆包、混元、零一万物、月之暗面、百川、Minimax、Groq、Ollama 等 <br/> • 支持所有主流 LLM 系列及区域性提供商 |
| **🔌 API 协议** | • **OpenAI Compatible**：`/v1/chat/completions`、Embeddings、Images、Audio <br/> • **Claude Messages**：Anthropic 原生格式 <br/> • **Gemini**：Google AI 格式 <br/> • **Responses API**：扩展响应处理 <br/> • **Realtime API**：WebSocket 流式传输 <br/> • **Rerank**：Cohere、Jina 语义搜索重排序 |
| **🔀 路由与容错** | • 模型映射与别名 <br/> • 基于优先级的渠道分组路由 <br/> • 错误时自动故障转移 <br/> • 跨渠道负载均衡 <br/> • 自定义重试策略 |
| **👥 用户管理** | • 基于角色的多用户系统 <br/> • 基于令牌的 API 认证 <br/> • 每用户额度限制与速率限制 <br/> • 订阅计划与分级管理 <br/> • 邀请系统与推荐奖励 |
| **💳 支付集成** | • 兑换码与礼品卡 <br/> • 易支付、Waffo Pancake 支付网关 <br/> • Stripe、Creem 国际支付 <br/> • 自定义支付网关支持 <br/> • 交易历史与发票生成 |
| **📈 数据分析** | • 详细的请求日志与筛选 <br/> • 令牌消耗跟踪（prompt/completion） <br/> • 成本计算与计费 <br/> • 日报/月报使用统计 <br/> • 会计导出功能 |

---

## 🚀 快速开始

### 方式一：使用 GHCR Docker 镜像（推荐）

```bash
# 拉取最新稳定版本
docker pull ghcr.io/lorry-san/nbapi:1.2.0

# 使用 SQLite 运行（简单部署）
docker run -d \
  --name nbapi \
  -p 3000:3000 \
  -v ./data:/data \
  -e SQL_DSN="file:/data/nbapi.db?_busy_timeout=9999999" \
  -e SESSION_SECRET="your-random-secret-here" \
  ghcr.io/lorry-san/nbapi:1.2.0
```

访问地址：`http://localhost:3000`

### 方式二：Docker Compose + PostgreSQL + Redis（生产环境）

```bash
# 克隆仓库
git clone https://github.com/Lorry-San/nbapi.git
cd nbapi

# 配置环境变量
cp .env.docker.example .env.docker
nano .env.docker  # 修改 POSTGRES_PASSWORD、REDIS_PASSWORD、SESSION_SECRET

# 部署
docker compose --env-file .env.docker -f docker-compose.github.yml up -d
```

> [!TIP]
> **首次部署**：部署前请至少修改 `.env.docker` 中的 `POSTGRES_PASSWORD`、`REDIS_PASSWORD` 和 `SESSION_SECRET`。

---

## 🐳 Docker 部署

### 可用镜像

| 镜像 | 说明 |
|-------|-------------|
| `ghcr.io/lorry-san/nbapi:1.2.0` | 最新稳定版本 |
| `ghcr.io/lorry-san/nbapi:main` | 主分支开发构建 |

### Compose 文件

| 文件 | 用途 |
|------|---------|
| `docker-compose.github.yml` | **生产环境**：使用预构建的 GHCR 镜像（推荐） |
| `docker-compose.local.yml` | 从源码本地构建镜像 |
| `docker-compose.dev.yml` | 开发环境 |
| `docker-compose.yml` | 默认配置 |

### 关键环境变量

| 变量 | 说明 | 示例 |
|----------|-------------|---------|
| `SESSION_SECRET` | 会话加密密钥（必需） | 随机 32+ 字符字符串 |
| `SQL_DSN` | 数据库连接字符串 | `postgres://user:pass@db:5432/nbapi` |
| `REDIS_CONN_STRING` | Redis 连接（可选但推荐） | `redis://redis:6379` |
| `SMTP_*` | 告警邮件设置 | 见 `.env.docker.example` |
| `FRONTEND_BASE_URL` | 公网访问 URL | `https://api.example.com` |

---

## 🔄 协议支持

| 协议 | 端点 | 用途 |
|----------|-----------|----------|
| **OpenAI Compatible** | `/v1/chat/completions`、`/v1/embeddings`、`/v1/images/generations`、`/v1/audio/*` | 标准 OpenAI 客户端 |
| **OpenAI Responses** | Responses API 格式 | 扩展响应处理 |
| **OpenAI Realtime** | Realtime API WebSocket | 实时流式传输 |
| **Claude Messages** | Claude 原生格式 | Anthropic Claude 模型 |
| **Gemini** | Google Gemini 格式 | Google AI 模型 |
| **Rerank** | Cohere/Jina reranker 格式 | 语义搜索重排序 |
| **自定义上游** | 可配置路径 | 自托管或自定义 API |

---

## 🛠️ 从 New API 迁移

现有 New API 用户可无缝迁移到 NBAPI：

```bash
# 1. 备份数据库和 docker-compose.yml

# 2. 更新 compose 文件中的镜像
image: ghcr.io/lorry-san/nbapi:1.2.0

# 3. 仅更新应用容器
docker compose pull
docker compose up -d --no-deps nbapi
```

> [!WARNING]
> **切勿运行 `docker compose down -v`**，否则可能删除数据库卷。

所有现有用户、渠道和配置将被保留。新的 NBAPI 功能（监控、告警、强制 IP 记录等）将立即生效。

---

## 📖 文档

| 资源 | 链接 |
|----------|------|
| **📘 English README** | [README.md](../README.md) |
| **🚀 发布版本** | [GitHub Releases](https://github.com/Lorry-San/nbapi/releases) |
| **🐳 Docker 镜像** | [GHCR Package](https://github.com/Lorry-San/nbapi/pkgs/container/nbapi) |
| **📝 本地 Docker 部署指南** | [docs/local-docker-deploy.zh-CN.md](./local-docker-deploy.zh-CN.md) |
| **🔗 上游 New API** | [QuantumNous/new-api](https://github.com/QuantumNous/new-api) |
| **📚 New API 文档** | [docs.newapi.pro](https://docs.newapi.pro/) |

---

## 🤝 相关项目

| 项目 | 说明 |
|---------|-------------|
| [QuantumNous/new-api](https://github.com/QuantumNous/new-api) | 上游项目（NBAPI 基于此项目） |
| [One API](https://github.com/songquanpeng/one-api) | New API 的原始基础 |
| [Midjourney-Proxy](https://github.com/novicezk/midjourney-proxy) | Midjourney API 集成 |

---

## 📜 许可协议

NBAPI 基于 [New API](https://github.com/QuantumNous/new-api) 开发，遵循 [GNU Affero General Public License v3.0](../LICENSE)。

根据 AGPLv3 及上游条款，衍生作品必须保留归属说明。NBAPI 在用户界面中保留可见的"基于 New API"归属信息。

---

<div align="center">

### 🌟 感谢使用 NBAPI

**[最新版本](https://github.com/Lorry-San/nbapi/releases)** • **[Docker 镜像](https://github.com/Lorry-San/nbapi/pkgs/container/nbapi)** • **[基于 New API](https://github.com/QuantumNous/new-api)**

<sub>基于 New API 二次开发 | Based on New API</sub>

</div>
