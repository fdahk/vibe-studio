# Vibe Studio

自研 AI 全栈开发平台（对标字节扣子编程的可复刻深核）。

技术选型原则：**优先经典、可迁移、标准化、零 vendor 锁定**——HTTP 用标准库 `net/http`，接口契约用 OpenAPI（spec-first），DDD 分层模块化单体。选型理由详见 [`docs/技术选型/`](docs/技术选型/)，关键概念见 [`docs/KB/`](docs/KB/)。

> **当前状态**：后端 user/auth 域（注册/登录/JWT）已跑通；前端已是 **pnpm + Turborepo monorepo**，web 接通后端 auth（Zustand 管客户端态 + TanStack Query 管服务端态 + 生成的类型安全客户端，Tailwind 样式），前后端均有测试与 lint。画布编辑器等业务待开发。

---

## 技术栈

### 后端（✅ 核心已落地）

> **选型主线**：coze 后端用字节自研/企业级组件（Hertz、Thrift、自研 logs、Atlas）；我们刻意换成**官方标准库 + 开源社区标准**，优先经典、可迁移、零 vendor 锁定。`✅已定` / `⏳待定` 标在「决策理由」列首。

| 维度 | coze 官方 | 我们的选型 | 决策理由 |
|---|---|---|---|
| 语言 | Go | **Go 1.26** | ✅ 与官方一致，无替换动机 |
| HTTP | Hertz（CloudWeGo，字节自研） | **net/http**（标准库，Go 1.22+ ServeMux） | ✅ **异于官方**：Hertz 绑定字节生态、是会变的框架；net/http 零依赖、最经典可迁移，1.22+ 的方法+路径路由 `"POST /api/v1/..."` 已够用。见 [01](docs/技术选型/01-Hertz-HTTP框架选型.md)/[07](docs/技术选型/07-OpenAPI契约与net-http框架决策.md) |
| 接口契约 | Thrift IDL | **OpenAPI 3（spec-first）** | ✅ **异于官方**：Thrift→model 的中间层工具是字节内部、不开源（踩过坑）；OpenAPI 开源、跨语言、前后端共用一份 spec。见 [07](docs/技术选型/07-OpenAPI契约与net-http框架决策.md) |
| 代码生成 | hz / thriftgo | **oapi-codegen** | ✅ 配套 OpenAPI 的开源生成器（types-only，不绑框架） |
| 中间件 | Hertz middleware | 经典 `func(http.Handler) http.Handler` | ✅ 标准库惯用模式：Recovery / RequestID / CORS / AccessLog + 路由级 Auth |
| ORM / 关系库 | GORM / MySQL | **GORM / MySQL 8** | ✅ 与官方一致，Go 生态主流。见 [02](docs/技术选型/02-GORM-ORM选型.md)/[03](docs/技术选型/03-MySQL-关系数据库选型.md) |
| 缓存 | Redis | **Redis**（go-redis） | ✅ 与官方一致 |
| 对象存储 | MinIO / S3 兼容 | **MinIO**（S3 兼容） | ✅ 与官方一致，S3 协议可平迁云厂商。见 [05](docs/技术选型/05-MinIO-对象存储选型.md) |
| 消息队列 | （多实现） | **⏳ 待定**（已调研） | ⏳ 暂无异步/解耦场景，不预先引入；需要时按 [04](docs/技术选型/04-消息队列多实现选型.md) 选型 |
| 鉴权 | JWT | **JWT**（golang-jwt） | ✅ 与官方一致，无状态、跨端通用 |
| 架构 | DDD | **DDD 模块化单体** | ✅ 与官方一致：api → application → domain ← infra，依赖倒置 |
| 配置 | — | env（godotenv） | ✅ 单体阶段够用，暂不引配置中心 |
| 结构化日志 | 自研 logs 包 | **slog**（标准库） | ✅ **异于官方**：自研 logs 是内部包；slog 是 Go 1.21+ 官方结构化日志，零依赖可迁移（`LOG_LEVEL` 控级别） |
| DB 迁移 | Atlas | **golang-migrate** | ✅ **异于官方**：二者同为版本化迁移工具；golang-migrate 更轻、`go:embed` 内嵌 SQL、社区主流（替代仅 dev 的 AutoMigrate） |
| Lint | golangci 类 | **golangci-lint v2** | ✅ 与官方同类，standard 集 + misspell + gofmt/goimports |
| 请求校验 | — | 手写校验 | ⏳ 暂手写；字段变多时引 validator（go-playground） |
| 测试 | — | **go test + testify** | ✅ 标准库 + 主流断言库，fake repo 免 DB |
| 部署 Dockerfile / CI | Dockerfile + helm | **⏳ 待定** | ⏳ 部署阶段再做（Dockerfile + GitHub Actions） |
| 可观测 metrics/tracing | Slardar（字节内部） | **⏳ 待定** | ⏳ 内部平台外部用不了；需要时上 Prometheus / OpenTelemetry |

### 前端（✅ monorepo + auth 已接通，画布等业务待开发）

> **选型主线**：coze 前端用 Rush（~259 包企业级）+ 大量字节内部组件（Semi / FlowGram / idl2ts / Slardar）；我们**右size 到主流轻量组合**，并对「画布 / UI 组件」刻意自研以吃透深核。`✅已定` / `⏳待定` 标在「决策理由」列首。

| 维度 | coze 官方 | 我们的选型 | 决策理由 |
|---|---|---|---|
| Monorepo | Rush + pnpm | **pnpm workspaces + Turborepo** | ✅ **异于官方**：Rush 为 ~259 包/多团队的企业级依赖策略设计，对我们过重；pnpm+Turborepo 主流轻量、配置少，是合适的「右size」。见 [08](docs/技术选型/08-前端Monorepo架构选型.md) |
| 构建 | Rsbuild（Rspack，字节自研） | **Vite** | ✅ **异于官方**：Rspack 为超大仓冷启动优化；我们规模下 Vite 生态更主流、文档全、与 Vitest 同源配置。见 [06](docs/技术选型/06-Rspack-Rsbuild-前端构建选型.md) |
| 框架 | React + TS | **React 18 + TypeScript** | ✅ 与官方一致，行业事实标准 |
| 样式 | Tailwind 3.3 + PostCSS | **Tailwind CSS v4**（Vite 插件） | ✅ **同选 Tailwind**；用 v4 走 CSS-first 配置 + 官方 Vite 插件，免手配 postcss/autoprefixer |
| CSS 预处理 | Less + Sass | **Sass（SCSS Module）** | ✅ **部分同官方**：选 Sass 不选 Less（二者同类，Sass 更主流）；与 Tailwind 分工——Tailwind 管布局/工具类，SCSS Module 管自研组件的复杂样式 |
| 状态管理 | Zustand + ahooks | **Zustand** | ✅ **同选 Zustand**（客户端态：token）；ahooks ⏳ 待定，当前无重复 hook 逻辑，按需再引 |
| 数据请求 | TanStack Query + axios | **TanStack Query + openapi-fetch** | ✅ **服务端态同选 TanStack Query**（缓存/失效/重取）；HTTP 客户端用 openapi-fetch 替代 axios——直接吃 openapi.yaml 生成的类型，端到端类型安全 |
| IDL→TS | idl2ts（字节内部） | **openapi-typescript** | ✅ **异于官方**：idl2ts 不开源（踩过坑）；openapi-typescript 开源、与后端共用一份 openapi.yaml |
| 路由 | react-router v6 | **react-router v6** | ✅ **同选 react-router**（与官方一致 v6）：已用于 web 路由（`/` 主页 / `/login` 登录），含基于 token 的路由守卫 |
| UI 组件 | Semi Design | **自研** | ✅ **异于官方**：本项目目标是自实现深核，UI 自研以吃透布局/受控/可访问性；Semi 是字节生态产物，引入会掩盖学习目标 |
| 画布 | FlowGram（字节自研） | **自研** | ✅ **异于官方**：画布是最核心深核（拖拽/连线/渲染/协同 OT），必须自研以吃透原理 |
| 代码编辑器 | Monaco | **⏳ 待定** | ⏳ code 节点（画布内写代码）出现时引 Monaco（VS Code 同源，事实标准） |
| Lint | ESLint + Stylelint + cspell | **ESLint（flat config）** | ✅ **同选 ESLint**（flat 为 v9 新标准）；Stylelint（主要用 Tailwind 原子类）、cspell（单人项目收益低）暂不引 |
| 格式化 | Prettier | **Prettier** | ✅ 与官方一致，事实标准 |
| 测试 | Vitest | **Vitest + Testing Library** | ✅ 与官方一致 |
| i18n | i18next + ICU | **i18next + react-i18next** | ✅ **同选 i18next**：内联 zh-CN/en 资源 + 语言切换；ICU 复杂消息格式 ⏳ 待定，出现复数/格式化文案再加 i18next-icu |
| 特性开关 | bot-flags（字节内部） | **⏳ 待定** | ⏳ 内部系统；单人项目暂无灰度需求 |
| 前端可观测（APM/埋点） | Slardar / Tea（字节内部） | N/A | — 内部平台外部用不了；需要时用 Sentry / OpenTelemetry |

> Monorepo 选型理由见 [docs/技术选型/08-前端Monorepo架构选型.md](docs/技术选型/08-前端Monorepo架构选型.md)。**原生 iOS(Swift)/Android(Kotlin) 不进 JS monorepo**，经同一份 `openapi.yaml` 契约生成各语言客户端。

---

## 架构

依赖方向（依赖倒置）：`api → application → domain ← infra`。领域层(domain)不依赖任何框架/数据库；基础设施(infra)实现领域层定义的接口。

**契约先行（spec-first）**：`backend/api/openapi/openapi.yaml` 是四端（Go / TS / 未来 Swift / Kotlin）唯一契约源——各端从它生成各自的类型/客户端，永不漂移。

**数据模型分三层**（详见 [docs/KB/IDL与数据模型分层.md](docs/KB/IDL与数据模型分层.md)）：
- DTO（传输模型）`api/openapi/openapi.gen.go` —— 由 openapi.yaml 生成
- Entity（领域实体）`domain/*/entity.go` —— 手写
- PO（持久化对象）`infra/persistence/*` —— 手写（gorm tag）

---

## 目录结构

```
backend/                  Go 后端（DDD 模块化单体）
  cmd/server/main.go      入口：net/http server + 中间件链 + 优雅退出
  api/
    openapi/              ★ 契约：openapi.yaml(单一源) + openapi.gen.go(生成的类型)
    router/router.go      组合根：装配依赖 + 聚合各域路由（不含单个域的路由细节）
    middleware/           经典中间件：recover/请求ID/CORS/访问日志 + Auth
    handler/
      health.go           健康探针
      user/               user 域 HTTP 层（routes.go 声明本域路由 + handler + toModel）
  application/user/       应用层：用例编排（注册/登录/查询）+ 单测
  domain/user/            领域层：实体 + Repository 接口（纯净，无框架依赖）
  infra/                  基础设施：mysql/redis/storage + migrate(golang-migrate) + persistence(GORM 实现) + Deps(手写 DI)
  migrations/             版本化 SQL 迁移（go:embed，启动时由 golang-migrate 执行）
  pkg/                    auth(JWT/密码,含单测) / errorx / ctxkit / response / logger(slog)
frontend/                 JS monorepo（pnpm workspaces + Turborepo；ESLint flat + Prettier）
  apps/web/               React + Vite（react-router 路由 + Tailwind/Sass 样式 + i18next 多语言 + Zustand + TanStack Query + Vitest）
  packages/api-client/    从 openapi.yaml 生成的类型安全客户端（@vibe/api-client）
  packages/shared/        跨包共享类型/工具（@vibe/shared）
ios/ android/             原生端（待建，经 openapi.yaml 契约生成各语言客户端）
docker-compose.yml        本地中间件：mysql / redis / minio
Makefile
```

---

## 快速开始

```bash
# 1) 起本地中间件（mysql / redis / minio）
make up

# 2) 拉后端依赖并启动（默认 :8888）
make tidy
make dev

# 3) 探针
curl localhost:8888/healthz        # 存活
curl localhost:8888/readyz         # 就绪（检查 mysql/redis/minio）

# 4) user 域：注册 → 登录 → 鉴权访问
curl -XPOST localhost:8888/api/v1/auth/register \
  -H 'Content-Type: application/json' -d '{"username":"alice","password":"pw123456","email":"a@x.com"}'
curl -XPOST localhost:8888/api/v1/auth/login \
  -H 'Content-Type: application/json' -d '{"username":"alice","password":"pw123456"}'
curl localhost:8888/api/v1/users/me -H "Authorization: Bearer <token>"

# 5) 前端（另开终端，:5173，已配 /api 代理到 :8888）
make fe-install
make fe-dev          # 浏览器打开 :5173，可注册/登录、看当前用户

# 6) 测试 + Lint
make test            # 前后端全部（后端 go test + 前端 vitest）
make lint            # 前后端全部（golangci-lint + eslint）
```

### 改了接口契约（openapi.yaml）后重新生成类型

```bash
make gen             # 一键重生成：后端 oapi-codegen + 前端 openapi-typescript
# 原生端（待建）：openapi-generator -g swift5 / kotlin，详见 docs/技术选型/08
```

> 说明：后端 **degraded 启动**——中间件没起时服务仍能启动、`/healthz` 正常，`/readyz` 显示哪个依赖未就绪；DB 不可用时 user 域不装配。本机若已占用 6379，compose 的 redis 容器起不来不影响（用本机 redis）。

---

## 相关文档

- [docs/技术选型/](docs/技术选型/) —— 各项选型理由（HTTP 框架 / ORM / DB / MQ / 存储 / 构建 / OpenAPI 决策 / Monorepo）
- [docs/IDL契约与多端代码生成方案.md](docs/IDL契约与多端代码生成方案.md) —— 契约与多端生成的演进
- [docs/KB/](docs/KB/) —— 可复用技术概念（如 IDL 与数据模型分层）
