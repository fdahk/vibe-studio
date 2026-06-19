# IDL 契约与多端代码生成方案

> 本文回答一个核心问题：**前后端如何从同一份 IDL 自动生成代码，做到"改一处契约、多端类型同步"**；并讲清一个容易踩的认知坑——"Apache Thrift 官方支持 TS，为什么不能直接拿来用"。
>
> 背景：后端已用 `idl/workflow.thrift` + `thriftgo`/`hz` 生成 Go（REST），详见 `backend/coze-studio-代码级架构分析.md` 与 `官方架构剖析与我们的工程决策.md`。本文聚焦"前端这一端怎么从同一份 IDL 生成"。

---

## 1. IDL 是什么、为什么要它

**IDL（Interface Definition Language，接口定义语言）** = 一份**语言无关**的接口契约，描述：数据结构（struct）、方法（service）、以及（在 hz 里）HTTP 路由（`api.post` 等注解）。

**它解决什么问题**：多端各写一套类型 → 必然**漂移**。后端把字段 `created_at` 改成 int64，前端还当 string，联调才炸；新增一个字段，前端漏改。手写多端模型 = 不一致的温床。

**IDL 的核心价值 = 单一事实源（single source of truth）+ 代码生成**：

```
            ┌──(thriftgo/hz)──→ 后端 Go: model + router + handler 骨架
idl/*.thrift ┤
            └──(idl2ts / OpenAPI 桥)──→ 前端 TS: 类型 + 请求函数
```

改 `.thrift` → 重新生成 → **两端类型/客户端一起变**。这就是"多端接口统一"的本质，也是 coze 的做法（后端 Go 用 thriftgo/hz，前端 TS 用自研 idl2ts，喂的是**同一份** `idl/`）。

> **推论**：前端**手写** TS 模型 = 放弃 IDL 的全部意义（又回到漂移）。所以前端也必须**从 IDL 生成**。本文就是定"怎么生成"。

---

## 2. 现状（后端已做完的部分）

- `backend/idl/workflow.thrift`：契约，带 hz 注解（`api.post="/api/v1/workflows"`、`api.body="name"` 等）。
- `thriftgo` → 生成 `api/model/workflow`（结构体 + json/form/query tag）。
- `hz` → 生成 `api/router`（路由）+ handler stub，接进我们的 DDD application service。
- **后端对外形态 = REST/JSON**：`POST /api/v1/workflows`，body 普通 JSON。

缺口：**前端 `packages/api-client` 怎么也从这份 IDL 自动生成**。

---

## 3. 核心：Thrift-RPC 与 REST 到底差在哪（决定前端怎么生成）

同一份 thrift `service`，能以两种**完全不同的形态**对外，这是全篇命门。先建心智模型 → 逐维度拆 → 看字节级真实例子。

### 3.1 心智模型：动词 vs 名词

- **RPC（Remote Procedure Call，远程过程调用）**：心智是**"调一个远程函数"**。API = 一堆**方法/动词**（`CreateWorkflow(req)`、`ListWorkflow(req)`）。客户端像调本地函数一样调它，框架负责把参数序列化、发出去、收回来。
- **REST（Representational State Transfer，表述性状态转移）**：心智是**"对资源做标准操作"**。API = 一堆**资源/名词**（`/workflows`、`/workflows/{id}`）+ **统一动词**（HTTP 的 GET/POST/PUT/DELETE）。你不"调方法"，你"对资源 GET / POST"。

> 一句话：**RPC 把"方法名"塞进 body；REST 把"操作语义"摊在 URL 路径 + HTTP 方法上。**

### 3.2 多维度对比

| 维度 | Thrift-RPC | REST/JSON |
|---|---|---|
| 抽象单位 | 方法（procedure） | 资源（resource）+ HTTP 动词 |
| 寻址 | 通常**单一 endpoint**，方法名在 body 里 | **每个资源一个 URL**，动词用 HTTP method |
| 数据编码 | thrift 协议（TBinary 二进制 / TCompact / TJSON），带 **message envelope**（方法名 + 消息类型 + seqid + 参数 struct），字段按**数字 id** 编码 | 普通 **JSON**，字段按**名字**，无 envelope |
| 传输 | TSocket（TCP 长连）或 THttpClient（HTTP 仅当管道） | 标准 HTTP（无状态请求/响应） |
| 客户端写法 | 生成 stub：`client.CreateWorkflow(req)` 像本地函数调用 | 构造请求：`fetch('/api/v1/workflows',{method:'POST',body})` |
| 错误表达 | thrift **exception**（在 IDL 里类型化定义） | **HTTP 状态码**（400/404/500）+ body |
| 可调试性 | ❌ 二进制/envelope，抓包看不懂，要专用工具 | ✅ 浏览器 / curl / Postman 直接看明文 JSON |
| 缓存/网关/CDN | ❌ 对中间件不透明（全是 POST 同一 URL，无法按方法缓存/路由） | ✅ HTTP 语义友好（GET 可缓存、幂等、路径可路由） |
| 性能 | ✅ 二进制更小更快（适合高并发服务间） | JSON 略大，但浏览器原生、足够用 |
| 典型场景 | **服务间内部 RPC**（东西向） | **对前端 / 对外 API**（南北向、浏览器） |

### 3.3 字节级真实例子（同一个"创建工作流"）

**REST/JSON（我们后端 hz 暴露的形态）**：
```
POST /api/v1/workflows HTTP/1.1        ← 路径本身就表达了"对 workflows 资源 POST = 创建"
Content-Type: application/json

{"name":"我的工作流"}                     ← 纯资源数据，字段按名字
---------- 响应 ----------
HTTP/1.1 200 OK
{"workflow":{"id":"db3d...","name":"我的工作流","status":"draft","created_at":1781872525}}
```

**Thrift-RPC（TJSONProtocol over HTTP）**：
```
POST /thrift HTTP/1.1                   ← 单一 endpoint，"创建"这个语义全在 body 里
Content-Type: application/x-thrift

[1,"CreateWorkflow",1,0,{"1":{"rec":{"1":{"str":"我的工作流"}}}}]
 │  │              │ │  └─ 参数：第1个arg(请求struct)→它的第1个字段(name)=string("str")
 │  │              │ └──── seqid (0)
 │  │              └────── 消息类型 (1 = CALL)
 │  └───────────────────── 方法名 "CreateWorkflow"
 └──────────────────────── 协议版本
```

**Thrift-RPC（TBinaryProtocol）**：body 是**紧凑二进制字节流**（形如 `80 01 00 01 00 00 00 0e 43 72 65 61 74 65 ...`），肉眼完全不可读。

**一眼看清差别**：
- REST → **路径有语义**（`/workflows` + POST），**body 是干净的资源 JSON**（按字段名）。
- Thrift-RPC → **单一 endpoint**，body 是 **envelope（方法名/类型/seqid）+ 按字段数字 id 编码的参数**，甚至是二进制。
- 结论：REST 用浏览器/curl 就能调、能看；Thrift-RPC 必须有对应的 thrift runtime 做编解码，否则连请求都拼不出来。

### 3.4 南北向 vs 东西向（为什么两者都存在、不是谁取代谁）

- **东西向（服务 ↔ 服务，内部）**：要高性能、强类型、内部可控 → 常用 **Thrift-RPC / kitex**。
- **南北向（浏览器/客户端 ↔ 后端，对外）**：要可调试、网关/CDN 友好、浏览器原生 → 常用 **REST/JSON**。
- **coze 正是这么分的**：内部微服务间可能用 kitex（thrift-RPC），对前端的 Web API 用 hz（REST）。**同一份 IDL，两种形态各取所需。**

### 3.5 回到本文

**hz 用 `api.*` 注解把 thrift service 映射成 ② REST/JSON**——这也是 coze 给前端的选择。所以：

> **前端要的是"REST 客户端"，不是"thrift-RPC 客户端"。** 这一句决定了第 5 节所有方案的取舍——为什么官方 `js:ts`（生成的是 RPC 客户端）不直接适用、为什么要走 thrift→OpenAPI→TS。

---

## 4. 澄清："Apache Thrift 官方支持 TS"——支持，但生成的是 RPC 客户端

**先承认事实：Apache Thrift 官方确实支持 TS**。`thrift --gen js:ts`（js 生成器的 ts 选项）会生成 JavaScript + `.d.ts` 类型定义；社区还有更现代的 `@creditkarma/thrift-typescript`。所以"thrift 不支持 TS"是错的。

**但它生成的是【形态① Thrift-RPC】客户端**：

```
// 官方 js:ts 生成的客户端大致这样用（形态①）：
const transport = new Thrift.THttpClient("/thrift");   // 单一 thrift endpoint
const protocol  = new Thrift.TJSONProtocol(transport); // thrift 协议
const client    = new WorkflowServiceClient(protocol);
client.CreateWorkflow(req);   // 把"方法名 CreateWorkflow + 参数"按 thrift message 编码后 POST
```
它发出的 body 是 **thrift message envelope**（`[methodName, msgType, seqId, struct...]`），打到**一个** thrift endpoint。

**而我们后端是【形态② REST】**：
```
POST /api/v1/workflows      body = {"name":"..."}      # 普通资源 JSON，按路径分发
```

两者对不上：
- **URL 结构不同**：单 endpoint vs RESTful 路径。
- **body 编码不同**：thrift message envelope vs 纯资源 JSON。
- **路由分发不同**：靠 thrift 方法名 vs 靠 HTTP 方法+路径。
- 而且 `api.*` 是 **hz 的扩展注解**，Apache 标准生成器**根本不认**，不会把它变成对 `/api/v1/workflows` 的 REST 调用。

> **一句话**：不是"thrift 不支持 TS"，而是**"官方 TS 生成器产出的是 RPC 客户端，与 hz 暴露的 REST API 不是同一种协议"**。要么换前端的生成方式（让它产 REST 客户端，Path A），要么换后端的暴露形态（改成真 thrift-RPC，Path B）。

---

## 5. 三条可行路径

### Path A —— REST/JSON（对齐 coze 的选择）
后端保持 hz REST，前端从同一份 IDL 生成**REST 客户端**。两种做法：

#### A1（推荐）：thrift → OpenAPI → TS
```
idl/workflow.thrift
  │ thrift-gen-http-swagger (CloudWeGo 官方插件, 认 hz 注解)
  ▼
openapi.yaml (OpenAPI 3.0)
  │ openapi-typescript / orval / openapi-generator
  ▼
前端 TS 类型 + 请求函数 (REST)
```
- **原理**：先把 hz-thrift 转成业界标准 **OpenAPI**，再用成熟的 OpenAPI→TS 工具产客户端。
- **优点**：单一 IDL 源；两段工具都成熟、社区大、维护好；**OpenAPI 本身是多语言中枢**（顺带白嫖 Swagger UI 调试、可生成任意语言客户端）；与 coze 的 REST 选择一致。
- **代价**：多一个中间格式（OpenAPI）；thrift→OpenAPI 表达力极少数边角有损耗（基本不影响）。
- **选错的代价**：几乎没有——这是最稳的工业级管线。

#### A2：直接用 coze 的 idl2ts
- **原理**：`@coze-arch/idl2ts-cli`（**就在 coze 开源仓库 `frontend/infra/idl/` 里**，带 CLI）直接把 hz-thrift → TS REST 客户端，专为 hz 注解设计。
- **优点**：最贴官方、一步到位、无中间格式。
- **代价**：bespoke 工具，要拖 coze 整套 `@coze-arch/*` 依赖与构建；文档少；维护性依赖上游。
- **选错的代价**：和一个内部工具链强绑定，将来升级/脱钩成本高。

### Path B —— 全 Thrift-RPC（最"纯官方 Thrift"）
- **原理**：后端**不走 hz REST**，改暴露**真正的 Thrift-RPC**端点（Apache thrift Go server 或 kitex）；前端直接用**官方 `thrift --gen js:ts`** 的 RPC 客户端（`THttpClient` + `TJSONProtocol` over HTTP）。
- **优点**：一个官方编译器、两端全包、**零 bespoke 工具**；最纯粹的"官方 TS"用法。
- **代价**：thrift-RPC over HTTP 在浏览器里**不友好**——难调试、抓包看不懂、无 REST 语义、对 CORS/缓存/网关/CDN 不友好；前端要引 thrift JS runtime；**偏离 coze**（coze 的 Web API 故意选 REST）。
- **选错的代价**：Web 端体验和可观测性差，且和"对齐 coze"的目标背离。

### 对比表

| 维度 | A1 thrift→OpenAPI→TS | A2 coze idl2ts | B 全 Thrift-RPC |
|---|---|---|---|
| 单一 IDL 源 | ✅ | ✅ | ✅ |
| 工具成熟度/社区 | ★★★★★ | ★★☆（bespoke） | ★★★★（apache 官方） |
| 与 coze 一致 | ✅（REST） | ✅✅（同款工具） | ❌（coze 用 REST） |
| Web 前端友好 | ★★★★★ | ★★★★★ | ★★（RPC 不适合浏览器） |
| 维护成本 | 低 | 中高（拖上游） | 中 |
| 附带收益 | OpenAPI/Swagger UI/多语言 | — | — |

---

## 6. 决断

**推荐 A1（thrift → OpenAPI → TS）**，排序理由：
1. **真·单一源 + 多端统一**：满足 IDL 的本来目的，且 OpenAPI 让"多端"扩展到任意语言。
2. **工具成熟、可长期维护**：两段都是社区主力工具，不绑死任何 bespoke 内部库。
3. **与 coze 的 REST 选择一致**，又比 coze 的 idl2ts 更标准、更通用。
4. **附带白嫖** Swagger UI（调试）+ OpenAPI 文档。

- **A2** 作为"最大程度复刻官方"的备选（想原样体验 coze 工具链时）。
- **B** 仅当你刻意要做"纯 Apache Thrift RPC 全栈"实验时；否则不选（Web 体验差 + 偏离 coze）。

---

## 7. A1 落地步骤（选定后执行）

1. **装 thrift→OpenAPI 插件**（CloudWeGo 官方）：
   ```bash
   go install github.com/hertz-contrib/swagger-generate/thrift-gen-http-swagger@latest
   ```
2. **从 IDL 生成 OpenAPI**（具体 flag 以插件 README 为准，模式如下）：
   ```bash
   thriftgo -g go ... # 已有
   # 用 http-swagger 插件产出 openapi.yaml（按 swagger-generate README 调用）
   ```
   产物建议放 `backend/idl/openapi/openapi.yaml`，作为前后端共享契约产物。
3. **前端 `packages/api-client` 从 OpenAPI 生成 TS**：
   ```bash
   pnpm add -D openapi-typescript
   npx openapi-typescript ../../backend/idl/openapi/openapi.yaml -o src/schema.d.ts
   # 运行时用 openapi-fetch 做"类型安全的 fetch 客户端"
   pnpm add openapi-fetch
   ```
4. **接入**：`api-client` 导出 typed client；`apps/web` 通过它调后端；vite/rsbuild 配 `/api` 代理到 :8888。
5. **一键生成**：在 `Makefile` 加 `gen` target，把"thrift→Go"+"thrift→OpenAPI→TS"串成一条命令，改完 IDL 一键同步两端。

---

## 8. 注意事项 / 踩坑

- **注解要写全**：`api.body/api.query/api.path` 不写全，生成的 OpenAPI 会缺字段/绑定错。
- **版本錨定**：thriftgo 0.4.5 生成的 model 用旧版 apache/thrift API，需 pin `apache/thrift v0.13.0`（已踩过，见后端分析）。
- **生成物要 commit、不手改**：生成代码当产物提交，禁止手改（`DO NOT EDIT`），改契约只改 `.thrift`。
- **统一响应包络**：若后端响应有统一 `code/msg/data` 包络，要在 IDL/OpenAPI 里体现，否则前端类型对不上真实响应。
- **OpenAPI 表达力边界**：thrift 的少数特性（如 typedef、复杂 union）转 OpenAPI 可能需手动校准，遇到再说。

---

## 9. 信息来源与核实状态

**已核实**：
- Apache Thrift 官方 TS：`thrift --gen js:ts` 生成 JS + `.d.ts`（生成 Thrift-RPC 客户端）——官方生成器行为。
- thrift→OpenAPI：CloudWeGo 官方 [hertz-contrib/swagger-generate（thrift-gen-http-swagger）](https://github.com/hertz-contrib/swagger-generate) · [Hertz Swagger 文档](https://www.cloudwego.io/docs/hertz/tutorials/third-party/middleware/swagger/)
- coze idl2ts：开源仓库内 `frontend/infra/idl/`（含 `@coze-arch/idl2ts-cli`，本地查证有 CLI bin）。
- OpenAPI→TS：`openapi-typescript` / `openapi-generator` / `orval`（业界主流，本文按通用用法描述）。

**待核实（落地前确认）**：
- `thrift-gen-http-swagger` 的**确切命令行调用方式**（作为 thriftgo 插件的 invoke flag）以其 README 为准。
- thrift→OpenAPI 对 hz 全部注解的覆盖度（个别注解可能需手动补 OpenAPI 字段）。

**结论**：前端必须**从 IDL 生成**（手写即放弃 IDL 意义）；推荐 **A1：thrift → OpenAPI → TS**；A2(coze idl2ts) 为对齐官方的备选；B(全 thrift-RPC) 仅特殊场景。
