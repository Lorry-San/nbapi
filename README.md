<div align="center">

<img src="https://pic.itxiaohui.top/20260501/34f446414f26798b1b4cce1fe30c6307.png" alt="NBAPI Logo" width="200"/>

# NBAPI

**🚀 Enterprise-Grade AI Gateway & Operations Platform**

*A powerful fork of New API with enhanced monitoring, alerts, and operational features*

[![License](https://img.shields.io/github/license/Lorry-San/nbapi?color=brightgreen)](https://github.com/Lorry-San/nbapi/blob/main/LICENSE)
[![Release](https://img.shields.io/github/v/release/Lorry-San/nbapi?color=brightgreen&include_prereleases)](https://github.com/Lorry-San/nbapi/releases)
[![Docker](https://img.shields.io/badge/docker-GHCR-blue)](https://github.com/Lorry-San/nbapi/pkgs/container/nbapi)
[![Based on New API](https://img.shields.io/badge/based%20on-New%20API-8A2BE2)](https://github.com/QuantumNous/new-api)

[📖 Documentation](#documentation) • [🎯 Features](#key-features) • [🚀 Quick Start](#quick-start) • [🐳 Docker Deploy](#docker-deployment) • [🌏 中文文档](./docs/README_CN.md)

</div>

---

## 🎯 What is NBAPI?

NBAPI is an **advanced AI API gateway and operations management system** built on top of [New API](https://github.com/QuantumNous/new-api). It provides:

✨ **Unified API Gateway** for 50+ AI providers (OpenAI, Claude, Gemini, DeepSeek, Qwen, etc.)  
📊 **Real-time Monitoring & Alerting** with SLA tracking, QPS/TPS metrics, and P95/P99 latency  
🔄 **Protocol Conversion** supporting OpenAI, Claude Messages, Gemini, Responses, Realtime API  
💰 **Quota Management** with payment gateway integration and usage tracking  
🛡️ **Enterprise Operations** including compliance statements, IP logging, and admin tools

> [!IMPORTANT]
> **Legal & Authorized Use Only**  
> NBAPI is designed for **authorized AI API distribution, private deployment, quota management, and cost tracking**. Users must ensure upstream API keys, model services, payment channels, user data handling, and public operations comply with local laws and upstream service terms.

---

## ✨ Key Features

| Category | Features |
|----------|----------|
| **📊 Monitoring & Alerts** <br/> *(NBAPI Enhanced)* | • Real-time dashboard with SLA, QPS/TPS, P95/P99 latency <br/> • Channel-level observability with error tracking <br/> • Customizable alert rules (error rate, latency thresholds) <br/> • Email notifications via SMTP <br/> • Request time delay metrics (avg, P95/P99, first token time) <br/> • Upstream error aggregation and slow channel identification |
| **🔧 Admin & Operations** <br/> *(NBAPI Enhanced)* | • Super admin management & one-click user login <br/> • Forced IP logging for compliance <br/> • CSV export for usage records (Classic theme) <br/> • Custom OpenAI API version paths (`/v1`, `/v3`, etc.) <br/> • Compliance statement enforcement before payment <br/> • Dual themes (Default modern UI + Classic admin panel) |
| **🤖 Model Support** | • **50+ Providers**: OpenAI, Claude, Gemini, Azure OpenAI, DeepSeek, Qwen (通义千问), Doubao (豆包), Hunyuan (混元), Yi (零一万物), Moonshot (月之暗面), Baichuan (百川), Minimax, Groq, Ollama, etc. <br/> • Support for all major LLM families and regional providers |
| **🔌 API Protocols** | • **OpenAI Compatible**: `/v1/chat/completions`, Embeddings, Images, Audio <br/> • **Claude Messages**: Anthropic native format <br/> • **Gemini**: Google AI format <br/> • **Responses API**: Extended response handling <br/> • **Realtime API**: WebSocket streaming <br/> • **Rerank**: Cohere, Jina semantic search reranking |
| **🔀 Routing & Fallback** | • Model mapping and aliasing <br/> • Channel groups with priority-based routing <br/> • Automatic failover on errors <br/> • Load balancing across channels <br/> • Custom retry strategies |
| **👥 User Management** | • Multi-user system with role-based access <br/> • Token-based API authentication <br/> • Per-user quota limits and rate limiting <br/> • Subscription plans and tier management <br/> • Invite system with referral rewards |
| **💳 Payment Integration** | • Redemption codes and gift cards <br/> • Epay, Waffo Pancake payment gateways <br/> • Stripe, Creem international payments <br/> • Custom payment gateway support <br/> • Transaction history and invoice generation |
| **📈 Analytics & Logs** | • Detailed request logs with filters <br/> • Token consumption tracking (prompt/completion) <br/> • Cost calculation and billing <br/> • Daily/monthly usage reports <br/> • Export capabilities for accounting |

---

## 🚀 Quick Start

### Option 1: Docker with GHCR Image (Recommended)

```bash
# Pull the latest stable release
docker pull ghcr.io/lorry-san/nbapi:1.2.0

# Run with SQLite (simple setup)
docker run -d \
  --name nbapi \
  -p 3000:3000 \
  -v ./data:/data \
  -e SQL_DSN="file:/data/nbapi.db?_busy_timeout=9999999" \
  -e SESSION_SECRET="your-random-secret-here" \
  ghcr.io/lorry-san/nbapi:1.2.0
```

Access at: `http://localhost:3000`

### Option 2: Docker Compose with PostgreSQL + Redis (Production)

```bash
# Clone repository
git clone https://github.com/Lorry-San/nbapi.git
cd nbapi

# Configure environment
cp .env.docker.example .env.docker
nano .env.docker  # Edit POSTGRES_PASSWORD, REDIS_PASSWORD, SESSION_SECRET

# Deploy
docker compose --env-file .env.docker -f docker-compose.github.yml up -d
```

> [!TIP]
> **First-time setup**: Change at least `POSTGRES_PASSWORD`, `REDIS_PASSWORD`, and `SESSION_SECRET` in `.env.docker` before deployment.

### Default Credentials

- **Username**: `root`
- **Password**: `123456`

⚠️ **Change the root password immediately after first login!**

---

## 🐳 Docker Deployment

### Available Images

| Image | Description |
|-------|-------------|
| `ghcr.io/lorry-san/nbapi:1.2.0` | Latest stable release |
| `ghcr.io/lorry-san/nbapi:main` | Development build from main branch |

### Compose Files

| File | Purpose |
|------|---------|
| `docker-compose.github.yml` | **Production**: Uses pre-built GHCR images (recommended) |
| `docker-compose.local.yml` | Local build from source |
| `docker-compose.dev.yml` | Development environment |
| `docker-compose.yml` | Default configuration |

### Key Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `SESSION_SECRET` | Session encryption key (required) | Random 32+ char string |
| `SQL_DSN` | Database connection string | `postgres://user:pass@db:5432/nbapi` |
| `REDIS_CONN_STRING` | Redis connection (optional but recommended) | `redis://redis:6379` |
| `SMTP_*` | Email settings for alerts | See `.env.docker.example` |
| `FRONTEND_BASE_URL` | Public access URL | `https://api.example.com` |

---

## 🔄 Protocol Support

| Protocol | Endpoints | Use Case |
|----------|-----------|----------|
| **OpenAI Compatible** | `/v1/chat/completions`, `/v1/embeddings`, `/v1/images/generations`, `/v1/audio/*` | Standard OpenAI clients |
| **OpenAI Responses** | Responses API format | Extended response handling |
| **OpenAI Realtime** | Realtime API WebSocket | Real-time streaming |
| **Claude Messages** | Claude-native format | Anthropic Claude models |
| **Gemini** | Google Gemini format | Google AI models |
| **Rerank** | Cohere/Jina reranker format | Semantic search reranking |
| **Custom Upstream** | Configurable paths | Self-hosted or custom APIs |

---

## 🛠️ Migration from New API

Existing New API users can seamlessly migrate to NBAPI:

```bash
# 1. Backup your database and docker-compose.yml

# 2. Update image in your compose file
image: ghcr.io/lorry-san/nbapi:1.2.0

# 3. Update only the application container
docker compose pull
docker compose up -d --no-deps nbapi
```

> [!WARNING]
> **Never run `docker compose down -v`** as it may delete your database volumes.

All existing users, channels, and configurations will be preserved. New NBAPI features (monitoring, alerts, forced IP logging, etc.) will activate immediately.

---

## 📖 Documentation

| Resource | Link |
|----------|------|
| **📘 Chinese README** | [docs/README_CN.md](./docs/README_CN.md) |
| **🚀 Releases** | [GitHub Releases](https://github.com/Lorry-San/nbapi/releases) |
| **🐳 Docker Images** | [GHCR Package](https://github.com/Lorry-San/nbapi/pkgs/container/nbapi) |
| **📝 Local Docker Deploy Guide** | [docs/local-docker-deploy.zh-CN.md](./docs/local-docker-deploy.zh-CN.md) |
| **🔗 Upstream New API** | [QuantumNous/new-api](https://github.com/QuantumNous/new-api) |
| **📚 New API Docs** | [docs.newapi.pro](https://docs.newapi.pro/) |

---

## 🤝 Related Projects

| Project | Description |
|---------|-------------|
| [QuantumNous/new-api](https://github.com/QuantumNous/new-api) | Upstream project (NBAPI is based on this) |
| [One API](https://github.com/songquanpeng/one-api) | Original foundation of New API |
| [Midjourney-Proxy](https://github.com/novicezk/midjourney-proxy) | Midjourney API integration |

---

## 📜 License

NBAPI is based on [New API](https://github.com/QuantumNous/new-api) and licensed under [GNU Affero General Public License v3.0](./LICENSE).

In accordance with AGPLv3 and upstream terms, derivative works must preserve attribution. NBAPI maintains visible "Based on New API" attribution in the user interface.

---

<div align="center">

### 🌟 Thank You for Using NBAPI

**[Latest Release](https://github.com/Lorry-San/nbapi/releases)** • **[Docker Image](https://github.com/Lorry-San/nbapi/pkgs/container/nbapi)** • **[Based on New API](https://github.com/QuantumNous/new-api)**

<sub>基于 New API 二次开发 | Based on New API</sub>

</div>
