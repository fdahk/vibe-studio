# 契约与 HTTP 框架最终决策：OpenAPI（spec-first）+ net/http

> 本文记录一次选型的**演进与定稿**：后端 HTTP 框架与"前后端接口契约/代码生成"方案，从最初对齐 coze 的 `Thrift + hz + Hertz`，改为 `OpenAPI(spec-first) + net/http + oapi-codegen/openapi-typescript`。
> 关联：`01-Hertz-HTTP框架选型.md`（Hertz 本身的分析）、本目录 `../IDL契约与多端代码生成方案.md`（更早的 thrift→OpenAPI→TS 讨论）。

---

## 0. 决策摘要

| 维度 | 原方案（对齐 coze） | 新方案（定稿） |
|---|---|---|
| 接口契约 | Thrift IDL | **OpenAPI 3（spec-first，单一事实源）** |
| 后端 HTTP 框架 | Hertz（CloudWeGo） | **net/http（标准库，Go 1.22+ ServeMux）** |
| 后端代码生成 | hz（生成 Hertz 路由/handler/model） | **oapi-codegen**（生成 net/http server 接口 + 类型） |
| 前端客户端生成 | idl2ts（字节内部、未开源） | **openapi-typescript**（开源成熟） |
| 中间件模型 | Hertz `app.HandlerFunc` | **经典 `func(http.Handler) http.Handler`** |

**一句话**：用 **OpenAPI 生态**取代 ByteDance 专有的 Thrift+hz+Hertz——前后端都原生兼容 OpenAPI，后端不再需要 hz，前端也绕开了"thrift→TS 中间层（idl2ts）不开源"的死结。底层换成标准库 net/http，学到/用到的全是**经典、可迁移、零 vendor 锁定**的技术。

---

## 1. 决策是怎么演进的（诚实复盘）

**起点**：项目目标之一是"对齐 coze 架构"，于是照搬了 coze 的后端栈：`Thrift IDL → hz/thriftgo 生成 → Hertz 服务`。

**逐个暴露的问题**：
1. **Thrift 本质是 RPC IDL**。它能对外做 REST，完全依赖 hz 的 `api.post`/`api.body` 注解 + hz 的路由生成——这是 vendor-specific 的扩展，不是 Thrift 本身的能力。
2. **hz 只生成 Hertz 代码**，和框架强绑定。用 hz = 锁定 Hertz。
3. **前端从 Thrift 生成 TS 客户端**卡死：coze 用的 `idl2ts` 是字节内部包、未独立开源；而"手写前端 model"又违背 IDL"单一源多端统一"的初衷。
4. **Hertz 通用性弱**：社区偏字节/国内生态，idiom 不通用；它还用自研 netpoll 替代了标准 `net/http` 模型——离"经典底层"最远。

**原则转变**：把优先级从 **"对齐 coze 的 vendor 栈"** 调整为 **"经典、可迁移的底层技术优先"**（框架会变，HTTP/契约/代码生成的标准与基础不变）。

**结论**：在这个新原则下，**OpenAPI 生态一次性解决上面全部问题**——它是 REST 的原生契约标准，前后端都有成熟开源工具，且能落在标准库 net/http 上。

---

## 2. 关键概念

- **接口契约（IDL/Spec）**：语言无关的接口描述，作为单一事实源，生成多端代码，避免类型漂移。（详见 `../IDL契约与多端代码生成方案.md`）
- **OpenAPI（曾名 Swagger）**：**REST/HTTP API 的契约标准**，用 YAML/JSON 描述路径、方法、请求/响应 schema。语言无关、全行业通用、工具生态最大。
- **spec-first vs code-first**：
  - **spec-first（契约优先，本项目采用）**：先写 `openapi.yaml` 作为唯一源 → 生成后端接口 + 前端客户端。契约是中心，最符合"多端统一"。
  - code-first：先写带注解的代码 → 反向生成 spec（如 swaggo）。spec 是派生物，"单一源"纯度低。
- **三种契约的定位区别**（别混淆）：

| 契约 | 本质 | 适合 | 典型生态 |
|---|---|---|---|
| **OpenAPI** | **REST/HTTP 契约** | 对外/对浏览器的 REST API | 全行业通用 |
| Thrift | RPC IDL | 服务间 RPC（二进制高效） | Facebook/ByteDance |
| Protobuf/gRPC | RPC IDL | 服务间高性能 RPC/流式 | Google/云原生 |

> 我们要做的是**给浏览器前端用的 REST API** → 天然该用 **OpenAPI**，而不是把一门 RPC IDL（Thrift）硬掰成 REST。

---

## 3. 为什么 OpenAPI 生态最合适（三点论证）

### 3.1 前后端都原生兼容 OpenAPI
```
openapi.yaml  ← 单一事实源（spec-first）
   ├─ oapi-codegen ──────→ 后端: Go 类型 + net/http server 接口（你实现接口）
   └─ openapi-typescript ─→ 前端: TS 类型 +（配 openapi-fetch）类型安全客户端
```
改一处 `openapi.yaml` → 重新生成 → 前后端类型同步。这就是 IDL"单一源多端统一"的目的，且工具全部开源、成熟。

### 3.2 后端不再需要 hz
hz 的唯一价值是"从 Thrift 注解生成 Hertz 路由"。走 OpenAPI + net/http 后，路由由 `oapi-codegen` 生成成标准库 net/http 接口，hz 失去存在意义，连带 Hertz、Thrift、apache/thrift 依赖一起移除。

### 3.3 前端的"中间层不开源"死结消失
Thrift→TS 没有成熟开源工具（idl2ts 是字节内部的）；而 **OpenAPI→TS 有一大批开源工具**（openapi-typescript / orval / openapi-generator），随便选。之前的阻塞点不复存在。

### 3.4 对比

| 维度 | Thrift + hz（旧） | OpenAPI + oapi-codegen（新） |
|---|---|---|
| 契约与 REST 的匹配度 | 低（RPC IDL 硬做 REST，靠 hz 注解） | 高（OpenAPI 就是 REST 契约） |
| 后端框架绑定 | 绑 Hertz | 标准库 net/http，零绑定 |
| 前端客户端生成 | ❌ idl2ts 未开源 | ✅ 多个成熟开源工具 |
| 工具/社区通用性 | 偏字节生态 | 全行业标准 |
| 附带产物 | — | Swagger UI、可生成任意语言客户端 |
| vendor 锁定 | 高 | 无 |

---

## 4. 为什么后端框架用 net/http（而非 Hertz/Gin）

- **net/http 是地基**：Gin/Echo/Fiber 都在包它，Hertz 则用 netpoll 替代它。学/用标准库 = 学那个**永不过时**的东西。
- **Go 1.22+ 的 `net/http.ServeMux` 已支持方法+路径路由**：`mux.HandleFunc("POST /api/v1/workflows", h)`、路径参数 `{id}`——标准库自己就够做 REST 路由，不需要第三方框架。
- **中间件是经典模式 `func(http.Handler) http.Handler`**：洋葱式包裹，是 Go Web 最通用、最可迁移的中间件写法（比任何框架自己的中间件 API 都通用）。
- **框架被 DDD 隔离在 api 层**：domain/application/infra 不依赖它，换框架成本极低。
- oapi-codegen 的 `std-http` 目标直接生成 net/http 兼容的 server 接口，和这一选择天然契合。

---

## 5. 选定工具链与工作流

- **契约**：`backend/api/openapi/openapi.yaml`（spec-first，手维护，单一源）
- **后端生成**：`oapi-codegen`（`std-http` 目标）→ 生成 `types.gen.go`（类型）+ `server.gen.go`（`ServerInterface` + 路由注册）。我们在 DDD handler 里**实现** `ServerInterface`，把请求接到 application service。
- **前端生成**：`openapi-typescript` → TS 类型；运行时配 `openapi-fetch` 做类型安全请求。
- **一键同步**：Makefile 加 `gen` target，串起"yaml → Go 生成 + yaml → TS 生成"。
- **工作流**：要加/改接口 → 改 `openapi.yaml` → `make gen` → 后端补实现、前端拿到新类型。

---

## 6. 取舍：放弃了什么、得到了什么

**放弃**：
- Thrift 的二进制紧凑编码与高性能 RPC —— 我们是面向浏览器的 REST，本来就用不到。
- 原生 RPC / 双向流 —— 同样用不到（要做也是另起 gRPC/WebSocket）。
- "对齐 coze 同款 vendor 栈"的叙事 —— 已不是当前优先级。

**得到**：
- 标准化、可迁移、零 vendor 锁定；后端落在标准库；中间件用经典模式。
- 前端客户端生成的阻塞点消失。
- 附带 OpenAPI 文档 / Swagger UI / 任意语言客户端的能力。

**什么时候该重新考虑**：若将来出现**内部服务间高性能 RPC**或**双向流式**需求，那一块可以单独引入 **gRPC/protobuf**（对外 REST 仍用 OpenAPI），两者并存、各管一段。

---

## 7. 对现有代码的影响（迁移范围）

- **不动**：`domain/ application/ infra/`（DDD 核心，框架无关）。
- **改 api 层**：
  - `cmd/server/main.go`：Hertz `server` → 标准库 `http.Server` + `ServeMux`。
  - `api/middleware`：Hertz 中间件 → 经典 `func(http.Handler) http.Handler`（recover / 请求 ID / CORS / 访问日志 / JWT 鉴权）。
  - `api/handler`：handler 签名 `(ctx, *app.RequestContext)` → `(http.ResponseWriter, *http.Request)`；实现 oapi-codegen 生成的 `ServerInterface`。
  - **JWT auth 的签发/校验逻辑是纯 Go，保留不动**，只改"从请求取 token、塞进 context"的 wiring 层。
- **删除**：Hertz / hz / Thrift / apache-thrift 依赖；之前 thriftgo/hz 生成的 `api/model/workflow`、`api/router/register.go`、`api/router/workflow`、`api/handler/workflow`；`idl/workflow.thrift`。
- **新增依赖**：`oapi-codegen`（开发期 codegen 工具）；运行时仅标准库 + 少量（如 JWT 库）。

---

## 8. 信息来源与核实状态

**已核实/业界事实**：
- OpenAPI 是 REST API 的事实契约标准；spec-first 是契约优先工作流。
- Go 1.22+ `net/http.ServeMux` 支持方法+路径模式路由与路径参数（标准库特性）。
- 经典中间件模式 `func(http.Handler) http.Handler` 是 Go Web 通用写法。

**待核实（执行时确认）**：
- `oapi-codegen` 的 `std-http` 生成目标的确切版本与命令行 flag（以其 README 为准；本项目执行迁移时验证）。
- `openapi-typescript` + `openapi-fetch` 的当前版本与用法（执行时锁定）。

**结论**：对"面向浏览器的 REST API + 单一契约多端统一 + 经典可迁移底层"这组目标，**OpenAPI(spec-first) + net/http + oapi-codegen/openapi-typescript** 是最自然、最标准、锁定最少的组合，取代原先对齐 coze 的 Thrift+hz+Hertz。
