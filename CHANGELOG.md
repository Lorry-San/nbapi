# NBAPI Changelog

本文件记录 NBAPI 正式版本的重要变化。更细的发布说明见
[`.github/release-notes`](.github/release-notes)。

## 1.5.3 - 2026-08-09

### 可信代理控制

- 在“系统设置 → 安全设置 → 可信代理”中增加网页端总开关，默认开启并保持 1.5.2 的现有行为。
- 关闭后立即忽略 `X-Forwarded-For`、`X-Real-IP` 等代理客户端 IP 头，仅使用 TCP 直连地址，无需重启容器。
- 将访问日志、使用日志、审计、登录会话、IP 限流、Token IP 白名单、Turnstile 与支付风控统一接入同一客户端 IP 解析逻辑。
- 保留 `TRUSTED_PROXIES` 的代理 IP/CIDR 白名单能力；网页开关关闭时优先覆盖环境变量配置。

## 1.5.2 - 2026-08-02

### 客户端 IP 识别

- 将 Cloudflare 官方 IPv4/IPv6 代理网段加入默认可信代理列表，恢复经 Cloudflare 反代时的真实客户端 IP 记录。
- 保留公网伪造防护：非 Cloudflare 的公网直连来源仍不能通过 `X-Forwarded-For` 覆盖客户端 IP。
- 显式配置 `TRUSTED_PROXIES` 时仍会完全替代默认列表，`none` 严格模式行为不变。
- 修复生产 Compose 未向应用容器传递 `TRUSTED_PROXIES` 的问题。

## 1.5.1 - 2026-07-27

### HA 稳定性

- 让 `/api/ha/health` 绕过全局 API 限流，避免只读 Redis replica 因执行写入型限流脚本而返回 500。
- 恢复 standby 健康探针应有的只读状态响应，不再被限流中间件的内部错误覆盖。
- slave 节点不再向只读 PostgreSQL standby 写入系统实例心跳。
- 增加只读 HA 健康路径和 slave 系统实例上报的回归测试。

## 1.5.0 - 2026-07-27

### 上游同步

- 基于上游 `v1.0.0-rc.22` 完成更新，吸收 `rc.17` 至 `rc.22` 的模型发现、计费、订阅、配额、用户管理、任务退款、流式响应、代理与后台管理修复。
- 增加 Sub2API、Alpha Search 计费、可配置工具价格、当前 Gemini 图像模型和 OpenAI Realtime GA 路由支持。
- 合入 Responses 转 ChatCompletions 流式处理中避免重复工具调用的上游修复。

### 协议兼容

- 保留 Moonshot/Kimi `/v1/responses` 自动转 ChatCompletions 的独立接管逻辑，并继续过滤上游不支持的工具类型。
- 保留渠道级强制转换开关、宽松/严格工具转换策略，以及 `/v1`、`/v3` Base URL 路径兼容。
- 修复合并后 Responses 转换类型、常量、流式策略与兼容层缺失的问题。
- 保留 reasoning 默认隐藏行为；仅在显式开启后将思考内容输出为 `<think>`。

### 前端与管理

- 按计划删除 Classic 前端源码和构建链路，Default 成为唯一内置前端。
- 将 NBAPI 独有的渠道、HA、运维监控、订阅、OAuth、更新、品牌和系统信息功能迁移到新版前端结构。
- 保留兼容旧安装所需的认证请求头、令牌签发方、Cookie 名称和浏览器存储键。

### 部署与价格

- 统一为一个生产用 `docker-compose.yml`，默认拉取 `ghcr.io/lorry-san/nbapi:latest`。
- 补齐 PostgreSQL、Redis、会话和加密密钥配置、Redis 持久化与服务健康检查。
- 更新生产部署文档，使用四个独立 UUID 生成 `.env` 密钥。
- OpenRouter 价格导入支持 `input_cache_write`，并映射到 NBAPI 的 `create_cache_ratio`。

### 升级提醒

- 升级前请备份数据库和当前环境变量。
- PostgreSQL 数据卷会保留首次初始化时的密码；只修改 `.env` 中的 `POSTGRES_PASSWORD` 不会更新数据库内已有密码。
- Classic 已不再内置。旧数据库若仍配置为 Classic，将通过兼容路径显示 Default 前端。

## 1.4.4 - 2026-07-10

- 新安装默认使用 Default 前端，并移除 Default 中的主题切换入口。
- 加固外部下载、Webhook、通知和媒体代理的 SSRF、重定向与响应大小限制。
- 增强配额转账、兑换、充值、订阅结算和用户更新的事务一致性。
- 增加 GPT-5.6 价格、订阅额度重置/充值和 OAuth 回调地址管理。

## 1.4.3 - 2026-07-09

- 恢复 Moonshot/Kimi 专用 Responses 转 ChatCompletions 路径，不再依赖渠道转换开关。
- 仅向 Moonshot 转发兼容的 `function` 工具，过滤 `namespace`、`plugin` 和 `custom_tool_call` 等不支持类型。
- 补充工具历史、孤立工具输出和内部路由保护测试。

## 1.4.2 - 2026-07-09

- 增加 Responses 工具转换的“宽松兼容”和“严格白名单”全局策略。
- 规范化不受支持的 `tool_choice`，并补充 `namespace` 转换回归测试。

## 1.4.0 - 2026-07-07

- 修复计费配额整数溢出风险，并限制图像数量、任务时长和 token 参数。
- 保留 Moonshot/Kimi `/v3` 路由和模型列表路径兼容。

## 1.3.1 - 2026-07-05

- 完成公开品牌、仓库链接、更新检查、镜像、服务文件和桌面端元数据的 NBAPI 化。
- 切换到 `NBAPI-User` 和 `X-NBAPI-*` 请求头，同时兼容旧客户端请求头。

## 1.3.0 - 2026-07-04

- 更新基础源码并优先采用共享的 Responses、ChatCompletions、Compact 和高级自定义转换实现。
- 恢复渠道级强制 Responses 转换、Claude/Opus 回退桥接和 reasoning 隐藏策略。
- 保留 HA、渠道认证、强制 IP 日志和首页背景等 NBAPI 独有功能。

## 1.2.0 - 2026-06-30

- 增加 Default 首页背景设置。
- 引入按版本维护的 GitHub Release Notes。

## 1.1.0 - 1.1.3

- 优化运维监控界面和告警规则弹窗。
- 改进 Responses Compact 回退以及串行、连续工具调用的配对和分组。
- 规范化工具参数，并让 Responses 回退路径默认隐藏思考内容。

## 1.0.0 - 1.0.2

- 增加渠道级 Responses 强制转 ChatCompletions 控制。
- 增加 Moonshot/Kimi Responses 回退支持。
- 增加 reasoning 输出控制，并改为默认隐藏。
