# IDL 与数据模型分层（DTO / Entity / PO）

> 解决一个普遍误解：**"用 IDL 维护一份模型，就能生成所有用到模型的地方"**——这句话对一半、错一半。
> 核心结论：**IDL 统一的是"网络边界上的契约模型"，跨的是"多端（多语言/多调用方）"，不是"一个服务内部的多层（DB/领域/API）"。** 三层模型是有意的解耦，和 IDL 正交。

---

## 1. 常见误解

很多人（包括第一次接触时）以为：
> 写一份 IDL（Thrift / OpenAPI / Protobuf）当"唯一模型源" → 自动生成数据库模型、领域模型、API 模型、前端类型……所有地方。

**这是错的。** IDL 只生成**一层**：API 契约（请求/响应在网络上的形状）。数据库模型、领域模型都不归它管，也不该归它管。

---

## 2. IDL 到底生成什么

IDL 生成的是 **契约模型 / 传输模型（DTO，Data Transfer Object）**——即数据在**网络边界**上的形状（HTTP 请求体、响应体的字段）。

它的真正价值在于**跨"端"统一**：

```
openapi.yaml  (一份契约源)
   ├─ oapi-codegen ──────→ 后端 Go: 请求/响应类型（api/openapi/openapi.gen.go）
   └─ openapi-typescript ─→ 前端 TS: 请求/响应类型 + 客户端
```

**没有 IDL 的痛点**：后端用 Go 写一遍 `RegisterRequest{Username,Password}`，前端用 TS **再手写一遍**同样的结构。两份手写副本**必然漂移**——后端改个字段，前端忘了同步 → 联调炸。

**有 IDL**：一份 `openapi.yaml` 同时生成 Go 类型和 TS 类型，前后端**共享同一契约源，永不漂移**。

---

## 3. 关键澄清："多端" ≠ "多层"

| 概念 | 含义 | IDL 是否覆盖 |
|---|---|---|
| **多端（multi-end）** | 同一网络边界的**不同侧/不同语言**：后端 Go、前端 TS、移动端、其它微服务 | ✅ IDL 正是为此 |
| **多层（multi-layer）** | 一个服务**内部**的纵向分层：API 层 / 领域层 / 持久化层 | ❌ IDL 只管 API 这一层 |

所以"一份模型生成所有用到的地方"——**对的范围是**："API 契约这一层，生成到所有语言/端"；**不包括**领域模型和 DB 模型（那是服务内部、不同关注点的东西）。

---

## 4. 三层数据模型

一个后端领域（以 user 为例）通常有三种"模型"，各管一段：

| 层 | 名称 | 本文项目中的位置 | 谁维护 |
|---|---|---|---|
| API 边界 | **DTO（传输模型）** | `api/openapi/openapi.gen.go`（`openapi.User` 等） | **改 `openapi.yaml` → 重新生成**（不手改生成物） |
| 业务核心 | **Entity（领域实体）** | `domain/user/entity.go`（`domain.User`） | **手写** |
| 持久化 | **PO（持久化对象）** | `infra/persistence/user_repo.go`（`userPO`） | **手写** |

三者通过**映射**连接：

```
HTTP JSON ──json.Decode──▶ DTO(openapi.RegisterRequest)
                                  │ handler 取字段
                                  ▼
                          Entity(domain.User)  ◀──── 业务逻辑在这层
                                  │ repo
                          ┌───────┴────────┐
                  Create: Entity→PO     Get: PO→Entity
                                  │
                                  ▼
                            PO(userPO) ──gorm──▶ users 表

返回方向：Entity ──toModel──▶ DTO(openapi.User) ──json──▶ HTTP JSON
```

本项目里的真实映射点：
- `userPO ↔ domain.User`：在 `infra/persistence/user_repo.go`（Create 时 Entity→PO，查询时 PO→Entity）。
- `domain.User → openapi.User`：在 `api/handler/user/deps.go` 的 `toModel()`。

---

## 5. 为什么不能"一个 struct 通吃三层"

三层是**不同关注点**，强行合并会出问题：

| 维度 | DTO（API） | Entity（领域） | PO（DB） |
|---|---|---|---|
| 关注 | 客户端看到什么 | 业务规则/不变量 | 表结构/索引/列类型 |
| `PasswordHash` | **绝不能有**（泄漏！） | 有 | 有 |
| 时间字段 | `created_at` unix int（前端好用） | `time.Time` | DB `datetime` |
| 字段命名 | 对外 API 风格 | 业务语义 | 列名/下划线 |
| 变更原因 | 接口演进 | 业务规则变化 | 加索引/改列/迁移 |

两个最硬的理由：

1. **安全/防泄漏**：`PasswordHash` 必须存在于 Entity 和 PO，但**绝不能进 API 响应**。如果三层共用一个 struct，密码哈希就会顺着 API 返回给前端。分层 + `toModel` 显式只挑该暴露的字段，从结构上杜绝泄漏。
2. **解耦/独立演进**：DB 改列名不应波及 API 契约（否则破坏所有客户端）；API 加字段不应逼你迁移数据库；领域模型可在两端都不变时保持稳定。一个共享 struct 会让这三件事互相牵连。

代价是要写几个映射函数（`toModel` 等），换来的是**安全 + 解耦**。

---

## 6. 官方 coze 也是三层——而且用了**两套** codegen

coze-studio 同样没有"一份模型 everywhere"：

| 层 | coze 里 | 谁生成/维护 |
|---|---|---|
| API DTO | `api/model/*` | **thrift 生成**（IDL） |
| 领域实体 | `domain/*` | **手写** |
| DB 模型 | `infra` + `gorm/gen` | **gorm/gen 生成**（源与 thrift 无关） |

关键观察：**coze 用 thrift 生成 API 模型、用 gorm/gen 生成 DB 模型——两套独立的代码生成、两个不同的源。** 这本身就证明：连官方也是**按层各用各的工具**，IDL（thrift）只负责 API 契约那一层。

---

## 7. 哪些手维护、哪些生成（本项目）

- **唯一手维护的契约源**：`api/openapi/openapi.yaml`。它生成**前后端的 API 类型**（多语言、多端），生成物不手改。
- **手写**：`domain/user/entity.go`（业务核心）、`infra/persistence/user_repo.go` 的 `userPO`（持久化）。

**给用户加一个字段（例：`nickname`）的完整流程**：
1. 改 `openapi.yaml`（加 `nickname`）→ `oapi-codegen` 重新生成 → DTO 有了，前端类型也会有。
2. 改 `domain/user/entity.go`（加 `Nickname`）。
3. 改 `userPO`（加 gorm 字段）+ repo 的 PO↔Entity 映射。
4. 改 `toModel`（Entity→DTO 带上 nickname）。

> 只有第 1 步是"改一份生成多处"，第 2~4 步是手写——因为它们是不同关注点。

---

## 8. 什么时候可以简化分层

层不是越多越好，是按需：

- **trivial CRUD、无敏感字段、不会长大**：可以合并领域和 PO（直接拿 gorm struct 当领域模型），甚至直接拿 DTO 当一切用。省事，但 API↔DB 耦合、字段易泄漏。
- **会长大的平台 / 有敏感字段（如密码）/ 前后端独立演进**：保持三层。coze 和本项目属于这类。

判断标准：**这层模型会因为不同原因而变化吗？会泄漏不该暴露的字段吗？** 是 → 分开；否 → 可合并。

---

## 9. 小结

1. **IDL 生成的是 API 契约（DTO）这一层，跨的是"多端/多语言"**——让前后端不用各手写一份 API 类型、永不漂移。
2. **IDL 不生成、也不该生成领域模型和 DB 模型**——那是服务内部不同关注点。
3. **三层（DTO/Entity/PO）+ 映射是有意的解耦**，和 IDL 正交；coze 同样三层，且 API 与 DB 用了两套不同的 codegen。
4. **"维护一份就生成所有地方"**：成立于"API 契约 × 所有端"，不成立于"一个 struct × 所有层"。

**关联**：`../技术选型/07-OpenAPI契约与net-http框架决策.md`（为什么用 OpenAPI 做契约）、`../IDL契约与多端代码生成方案.md`（thrift→OpenAPI 的演进与工具链）。
