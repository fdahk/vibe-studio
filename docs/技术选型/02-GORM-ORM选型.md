# 02 - GORM 全家桶：Go ORM 选型理由

> 主题：`gorm.io/gorm` + `gorm.io/gen` + `gorm.io/plugin/dbresolver` + `gorm.io/driver/mysql` + `gorm.io/driver/sqlite`
> 对标对象：coze-studio（字节扣子开源版）后端
> 本地源码核实路径：`/tmp/coze-studio/backend`，版本以 `backend/go.mod` 为准
> 面向想深入理解 Go ORM 选型本质的读者。

---

## 0. 一句话结论（先给答案再讲为什么）

coze 后端的数据访问层是 **「GORM 做运行时引擎 + gorm/gen 编译期生成类型安全 DAO」** 的组合拳，外加 dbresolver 预留读写分离、sqlite 兜测试。

这套组合的本质是：**用 GORM 的生态/灵活性打底，用 gen 的代码生成把 GORM 最大的短板（运行时弱类型）补上**。它既不是"无脑选最流行"，也不是"为了类型安全上 ent"，而是一个在「生态成熟度」和「类型安全」之间做了精确权衡的中间解。

后面 7 段会把这个结论拆开讲清楚，并诚实标注哪些是源码证据（事实），哪些是合理推断（动机）。

---

## 1. 是什么（扫盲：ORM 是什么 / 为什么要）

### 1.1 ORM 解决的根本问题：阻抗失配（impedance mismatch）

数据库里是「表 + 行 + 列」，代码里是「对象 + 字段」。两种模型天生对不上：

- 数据库没有"对象"概念，只有扁平的行；代码里却是嵌套的 struct、有指针、有切片
- 关系在数据库里靠外键 + JOIN 表达；在代码里靠对象引用表达
- 数据库的 NULL ≠ 代码的零值，需要额外处理

这个鸿沟叫 **对象-关系阻抗失配**。每次手写 SQL，你都在手动跨越这个鸿沟：写 `SELECT`、把每一列 `rows.Scan(&x.Name, &x.Age, ...)` 塞进 struct、处理 NULL、拼 WHERE 条件。重复、易错、改一列要改一片。

ORM（Object-Relational Mapping，对象关系映射）就是把这层映射自动化的库：你定义一个 struct，ORM 负责帮你生成 SQL、扫描结果、维护关联。

### 1.2 ORM 的能力光谱（不是非黑即白）

Go 生态里"操作数据库"的方案是一条光谱，不是只有"用 ORM"和"不用 ORM"两档：

| 抽象层级 | 代表 | 你写什么 | 谁帮你做什么 |
|---|---|---|---|
| 裸驱动 | `database/sql` | 全部 SQL + 手动 Scan | 只给连接池和 `*sql.Rows` |
| SQL 薄封装 | `sqlx` | 全部 SQL | 帮你把行 Scan 进 struct |
| SQL builder | `squirrel` | 用链式调用拼 SQL 字符串 | 帮你安全拼接，不帮 Scan |
| SQL-first 生成 | `sqlc` | 写 `.sql` 文件 | 编译期生成类型安全的 Go 函数 |
| schema-first 生成 | `ent` | 写 schema 代码 | 生成全套类型安全 API + 迁移 |
| 运行时全功能 ORM | `GORM` | 写 struct + 链式调用 | 运行时反射生成 SQL、关联、钩子、迁移 |

GORM 在最右端：**功能最全、抽象最高、上手最快，代价是运行时反射 + 弱类型**。这个"代价"正是后面 gen 要解决的问题，先记住。

### 1.3 GORM 全家桶各模块是什么

coze 用的不是单个 GORM，是一套组合，先把每个零件是什么讲清楚：

- **`gorm.io/gorm`**：核心库。提供 `db.Where().Find()` 这种链式 API、关联（has-one/has-many/many-to-many）、钩子（BeforeCreate 等）、自动迁移（AutoMigrate）、事务、软删除。靠运行时反射工作。
- **`gorm.io/gen`**：代码生成器。读数据库表结构（或你给的 model），**编译期生成**类型安全的 DAO 代码。把 GORM 运行时才报错的弱类型查询，变成编译期就能检查的强类型调用。这是全家桶的灵魂。
- **`gorm.io/plugin/dbresolver`**：读写分离 / 多数据源插件。配置主从后，读走从库、写走主库；也支持按表分库。
- **`gorm.io/driver/mysql`**：MySQL 方言驱动。GORM 核心不绑定具体数据库，方言（怎么拼 MySQL 特有 SQL、怎么转类型）由 driver 提供。
- **`gorm.io/driver/sqlite`**：SQLite 方言驱动。主要价值是 **零外部依赖**——`:memory:` 模式直接在进程内开一个数据库，单测不用起 MySQL 容器。

---

## 2. 在 coze 里干什么（源码举证）

> 以下每条都标注了核实路径，是从 `/tmp/coze-studio/backend` 实际读出来的，不是脑补。

### 2.1 版本事实（`backend/go.mod`）

```
go 1.24.0
gorm.io/gorm                 v1.25.11
gorm.io/gen                  v0.3.26
gorm.io/plugin/dbresolver    v1.5.2
gorm.io/driver/mysql         v1.5.7
gorm.io/driver/sqlite        v1.4.3   // indirect / 仅测试用
gorm.io/driver/postgres      v1.5.11  // indirect，全家桶带的，未见业务直接用
github.com/go-sql-driver/mysql v1.9.0 // indirect，MySQL 底层驱动
github.com/mattn/go-sqlite3    v1.14.15 // indirect，SQLite 底层驱动（cgo）
```

**事实**：五个目标依赖全部存在且是直接依赖（除 sqlite 是 indirect）。说明这套组合是真用的，不是 go.mod 里的历史残留。

### 2.2 连接初始化极薄（`backend/infra/orm/`）

这个目录只有两个文件，薄到出乎意料：

- `infra/orm/database.go`：**整个文件就一行有效代码** `type DB = gorm.DB`。coze 没有自己包一层 ORM 抽象，而是直接把 `gorm.DB` 当成项目的数据库类型别名。
- `infra/orm/impl/mysql/mysql.go`：`New()` 函数，`gorm.Open(mysql.Open(dsn))` 打开连接，然后从环境变量读连接池参数（`SetMaxIdleConns` 默认 10、`SetMaxOpenConns` 默认 100、`SetConnMaxLifetime` 默认 3600s、`SetConnMaxIdleTime` 默认 600s）。

**解读**：coze 不做"自己再抽象一层 Repository 接口包住 ORM"这种过度设计。`gorm.DB` 直接透传，连接池参数走环境变量配置。这是 right-size 的体现——大厂开源项目反而比想象中朴素。

### 2.3 gen 是真正的主力（136 个文件用到 gen）

**事实**：`grep -rl "gorm.io/gen"` 命中 **136 个 `.go` 文件**。几乎每个 domain 都有 `internal/dal/query/*.gen.go` + `internal/dal/model/*.gen.go`。

生成器本体在 `backend/types/ddl/gen_orm_query.go`（一个 `package main` 脚本）。它的关键配置：

```go
g := gen.NewGenerator(gen.Config{
    OutPath: filepath.Join(rootPath, path),
    Mode:    gen.WithoutContext | gen.WithDefaultQuery | gen.WithQueryInterface,
    FieldNullable: fieldNullablePath[path],
})
g.UseDB(gormDB)
g.WithOpts(gen.FieldType("deleted_at", "gorm.DeletedAt"))
```

几个值得讲的细节（都是源码里读到的）：

- **`path2Table2Columns2Model` 是一张大映射表**：手工列出"哪个 domain 的哪张表，哪些 JSON 列要映射成哪个 Go struct"。比如 `single_agent_draft` 表的 `model_info` 列映射成 `*bot_common.ModelInfo`，并打上 `serializer:json` 标签。这解决了 GORM 原生对"列里存 JSON、读出来要反序列化成具体类型"的支持不够友好的问题。
- **时间字段特殊处理**：`created_at`/`updated_at` 被改成 `autoCreateTime:milli`/`autoUpdateTime:milli`，即用毫秒时间戳存而非 `DATETIME`。这是性能/跨时区取舍。
- **生成的 model 长这样**（`app_draft.gen.go`）：每个字段是 `field.Int64`/`field.String` 这种**字段表达式对象**，不是裸的 `interface{}`。所以你写查询时是 `q.AppDraft.ID.Eq(123)` 这种编译期类型安全的调用，而不是 GORM 原生的 `Where("id = ?", 123)` 字符串。

**这就是 coze 选 gen 的核心收益**：把 GORM 的弱类型字符串查询，升级成编译期检查的强类型链式调用。改了列名，编译直接报错，而不是上线后运行时炸。

### 2.4 业务层怎么用生成的 DAO

**事实**（`grep query.Use`）：业务层通过 `query.Use(db)` 或 `query.SetDefault(db)` 拿到生成的 Query 对象。例如：

- `domain/app/repository/app_impl.go:59`：`query: query.Use(components.DB)`
- `domain/memory/database/service/database_impl.go:124`：`tx := query.Use(d.db).Begin()`——事务也是生成代码提供的。

生成的 `Query` 结构体（`app/internal/dal/query/gen.go`）提供了 `Transaction`、`Begin/Commit/Rollback`、`WithContext`、`ReadDB()/WriteDB()` 全套方法。

### 2.5 dbresolver：代码里有，但运行时没启用（重要的诚实区分）

这是本文最需要讲清楚的一个点，因为它直接关系"事实 vs 推断"。

**事实 A**：`grep dbresolver` 命中 **70 个文件**，全是 `.gen.go`。每个生成的 query 里都有：

```go
func (q *Query) ReadDB() *Query  { return q.ReplaceDB(q.db.Clauses(dbresolver.Read)) }
func (q *Query) WriteDB() *Query { return q.ReplaceDB(q.db.Clauses(dbresolver.Write)) }
```

**事实 B**：搜索运行时的物理注册（`dbresolver.Register` / `db.Use(dbresolver...)` / `Replicas` / `Sources`），**在非生成、非测试代码里一个都没有**。

**事实 C**：搜索业务层主动调用 `.ReadDB()` / `.WriteDB()`，**结果为空**——没有任何业务代码主动走读写分离路径。

**结论（事实层面）**：dbresolver 当前是 **"gen 模板自带、代码层面预留、但部署上没启用"** 的状态。`ReadDB()/WriteDB()` 是 gorm/gen 生成代码的标准产物，不是 coze 团队手写的读写分离逻辑。当前 coze 是单库部署；要启用读写分离，运维只需在 `gorm.Open` 后加一段 `db.Use(dbresolver.Register(...))` 配置主从，业务层再按需调 `.ReadDB()`，**不用改任何业务代码**。

这个区分很重要：很多文章会说"coze 用了读写分离"，但源码证据是"coze 预留了读写分离能力，当前未启用"。**这正是选 GORM+gen 全家桶的隐性收益之一——扩展点是免费送的。**

### 2.6 sqlite 只服务测试（1 个文件）

**事实**：`gorm.io/driver/sqlite` 只在 `internal/mock/infra/orm/sqlitedb.go` 一处用到。它用 `:memory:` 或 `file:xxx?mode=memory&cache=shared` 开内存库，配合 `AutoMigrate` 建表、塞测试数据、跑完 `DropTable` 清理。

**意义**：DAL 层单测不依赖外部 MySQL，直接进程内跑。CI 快、本地无环境负担。

### 2.7 生产迁移走 Atlas，不是 GORM AutoMigrate（推翻常见假设）

**事实**：`grep AutoMigrate` 在非测试代码里 **一个都没有**。AutoMigrate 只出现在上面那个 sqlite mock 文件里（建测试表用）。

**事实**：真正的 schema 迁移在 `docker/atlas/` 下：
- `migrations/` 里有 **17 个版本化 SQL 迁移文件**（`20250703095335_initial.sql` 到 `20251028085526_update.sql`）
- `opencoze_latest_schema.hcl`（声明式 schema 全貌）
- `atlas.hcl`（Atlas 配置）+ `atlas.sum`（校验和）

**解读**：这是个反直觉但非常专业的取舍。GORM 自带 AutoMigrate，新手都爱用——但 **生产环境 coze 明确不用 AutoMigrate**，而是用 [Atlas](https://atlasgo.io/) 做版本化、可审查、可回滚的迁移。原因见第 3 段。这说明 coze 把 GORM 的定位收窄成了"运行时查询引擎"，迁移这件危险的事交给专业工具。

---

## 3. ★ 为什么官方选它（三类动机，重点讲 gen / dbresolver 解决了什么）

按规格要求，分三类动机，**并诚实标注哪些是源码可证的事实、哪些是合理推断**。

### 3.1 纯技术动机

**① GORM 是 Go 最成熟、生态最大的 ORM（事实层面的行业共识）**

- 文档最全（官方中英文文档齐全）、Star 最多、Stack Overflow 答案最多、第三方插件最多。
- 对团队招人友好：随便招个 Go 后端大概率用过 GORM，学习成本几乎为零。
- 这一条本身不是 coze 源码能证明的，是 Go 社区的客观生态事实。**推断**：coze 选它有"团队熟悉度 + 招人友好"的考量，但官方没明说。

**② gen 补上了 GORM 最痛的短板：运行时弱类型（核心技术动机，源码强证据）**

GORM 原生 API 的致命伤是 **字符串 + interface{}**：

```go
// GORM 原生：列名是字符串，值是 interface{}，编译期完全不检查
db.Where("naem = ?", name).Find(&users)  // 拼错 naem，编译通过，运行时才炸
```

gen 把它变成：

```go
// gen 生成：字段是强类型对象，拼错列名 / 类型不匹配 = 编译报错
q.User.Where(q.User.Name.Eq(name)).Find()
```

**源码证据**：136 个文件用 gen、每个 model 的字段都是 `field.Int64`/`field.String`（见 2.3）。这是 coze 真金白银在用的核心能力，**是事实不是推断**。

为什么这一点对 coze 这种大项目特别重要：表多（17 个迁移文件演进出几十张表）、字段多、多人协作、长期维护。靠运行时字符串查询，重构时（改列名、改类型）是地狱；有了 gen，编译器就是你的回归测试。

**③ gen 同时解决了 GORM 对「JSON 列 ↔ Go struct」支持不友好的问题（源码强证据）**

见 2.3 的 `path2Table2Columns2Model`。coze 大量字段是把复杂结构（`ModelInfo`、`PluginInfo[]`、`OpenAPI doc`）以 JSON 存进单列。gen 配置 `serializer:json` + 指定 Go 类型，让这些列读出来直接是强类型 struct，省掉手动 Marshal/Unmarshal。**事实**。

**④ dbresolver 提供"零成本预留"的水平扩展能力（事实：预留；推断：动机）**

见 2.5。**事实**是代码预留了读写分离、当前未启用。**推断**的动机是：作为一个要被无数人私有化部署的开源项目，coze 需要"小公司单库能跑、大公司加从库就能扩"的弹性，而 gen + dbresolver 让这个扩展点 **零业务代码改动** 就能开启。这是个很聪明的"为未来留门、但不为未来付现在的成本"的设计。

**⑤ sqlite 给测试零依赖（源码证据）**

见 2.6。**事实**。让 DAL 单测进程内可跑，CI 友好。

### 3.2 生态 / dogfooding 动机

**① CloudWeGo 生态本身不含 ORM，GORM 是自然补位（推断，但合理）**

coze 后端用字节自家的 CloudWeGo 全家桶（Hertz/Kitex/Eino）。但 CloudWeGo **没有官方 ORM**——字节内部数据访问层并不强推某个自研 ORM。所以 ORM 这块是"生态空白，按社区主流选"，GORM 作为 Go 事实标准被选中。**这一条是推断**：字节没公开说"我们就用 GORM"，但从"CloudWeGo 不提供 ORM + GORM 是社区标准"两个事实可以合理推出。

**② 没有 dogfooding 自研 ORM 的动机（推断）**

不像 RPC 框架（Kitex/Hertz 是字节核心基建、有强 dogfooding 动机），ORM 不是字节想对外秀肌肉的领域。所以这里看不到"为了推广自研轮子而选某库"的痕迹——选 GORM 是务实而非政治。**推断**。

### 3.3 平台 / 私有化动机

**① 开源 + 私有化部署，要求"最低环境门槛"（事实支撑推断）**

coze-studio 是要让外部用户在自己机器上 `docker compose up` 就能跑起来的。这带来两个对 ORM 的硬约束：

- **数据库要主流、易得**：选 MySQL（`driver/mysql`），不绑字节内部数据库。**事实**。
- **测试 / 轻量场景零外部依赖**：sqlite 内存库（**事实**）让开发者不装 MySQL 也能跑 DAL 测试。

**② 用 Atlas 而非 AutoMigrate，是私有化场景的负责任选择（事实 + 推断）**

**事实**：生产迁移走 Atlas 版本化 SQL（见 2.7），AutoMigrate 只在测试用。**推断动机**：AutoMigrate 在生产是危险的——它会"尽力让表结构匹配 struct"，但不可控（可能加列不可能删列、改类型行为模糊、无法 review、无法回滚）。对一个要在 **别人生产环境** 跑、还要支持版本升级的开源平台，必须用可审查、可回滚、版本化的迁移。Atlas 正是干这个的。这体现了 coze 把 GORM 的能力**有意识地收窄**——只用它擅长且安全的部分（运行时查询），危险的部分（迁移）交给专业工具。

### 3.4 小结：这套组合到底在解决什么

| 模块 | 解决的问题 | coze 里的证据等级 |
|---|---|---|
| gorm 核心 | 对象关系映射、链式查询、事务、关联 | 事实（136 文件 + infra/orm） |
| gen | GORM 弱类型 → 编译期类型安全；JSON 列 → 强类型 | 事实（136 文件，生成器在 types/ddl） |
| dbresolver | 读写分离能力（预留，未启用） | 事实（预留）；动机为推断 |
| driver/mysql | 主流数据库、私有化易得 | 事实 |
| driver/sqlite | 测试零依赖 | 事实（1 文件，仅测试） |
| Atlas（非 GORM） | 安全的版本化迁移（替代 AutoMigrate） | 事实 |

---

## 4. ★ 同类替代逐个对比 + 为什么不选（含选错代价）

下面每个替代都给：**它是什么 → 对比维度 → 为什么 coze 不选 → 如果当初选错的代价**。

### 4.1 ent（Facebook / Meta 出品）

**它是什么**：schema-as-code 的 ORM。你用 Go 代码定义 schema（字段、边/关系、索引），ent 代码生成出**全套类型安全 API**，包括图遍历式的关联查询。是 Go 生态里类型安全做得最彻底的 ORM。

**对比维度**：

| 维度 | ent | GORM + gen |
|---|---|---|
| 类型安全 | 极强（schema 即真相，全 API 生成） | 强（gen 生成查询 DAO，但 schema 仍可绕过） |
| 关联查询 | 图模型，遍历优雅 | 传统 ORM 关联，够用 |
| 学习曲线 | **陡**（schema DSL、Builder、代码生成心智模型都得学） | 平缓（会 GORM 就会大半） |
| 侵入性 | **高**（整个数据层都得按 ent 的 schema-first 范式组织） | 低（gen 是可选增强，底层还是普通 GORM） |
| 生态/社区 | 小于 GORM（虽然 Meta 背书，但中文资料、第三方插件远少） | 最大 |
| 灵活性（动态/原生 SQL） | 较弱（强范式，逃逸到原生 SQL 不顺手） | 强（随时 `db.Raw()` / `Where(string)`） |

**为什么 coze 不选（推断 + 行业共识）**：

- **侵入性 + 学习曲线**：ent 要求整个数据层 buy-in 它的 schema-first 范式。coze 几十个 domain、多人协作，强制全员学 ent 的 DSL 和图模型，迁移和招人成本都高。GORM 是 Go 后端的"普通话"，ent 是"方言"。
- **灵活性**：coze 有大量 JSON 列、动态条件查询、跨表场景。GORM 留了大量逃生通道（原生 SQL、字符串条件），ent 的强范式在这些场景反而别扭。
- **生态厚度**：ent 虽好，社区和第三方支持仍明显小于 GORM。对要被外部私有化、可能遇到各种环境的开源项目，"踩坑有人答"很重要。

**选错代价**：如果 coze 当初选了 ent——① 招来的 Go 工程师大概率没用过 ent，上手期拉长；② 那些 JSON 列 + 动态查询场景会写得很拧巴，可能大量逃逸到原生 SQL，反而丧失了 ent 的类型安全优势；③ 社区遇到的边角问题更难搜到答案。**但要公平说**：如果 coze 是个关系极其复杂、schema 高度稳定的强一致系统，ent 的图模型 + 极致类型安全反而会是更优解。coze 的"JSON 重、关系中等、迭代快"特征，让天平倒向 GORM+gen。

### 4.2 sqlc

**它是什么**：SQL-first 代码生成器。你手写 `.sql` 文件（带特定注释），sqlc 解析 SQL + 表结构，**编译期生成类型安全的 Go 函数**。哲学是"SQL 才是真相，代码是 SQL 的产物"。

**对比维度**：

| 维度 | sqlc | GORM + gen |
|---|---|---|
| 类型安全 | 极强（从 SQL 推导，编译期检查） | 强 |
| 对 SQL 的掌控 | **完全掌控**（你写的就是最终 SQL，无隐藏行为） | 部分（ORM 生成 SQL，偶有惊喜） |
| 动态查询 | **弱**（SQL 是静态的，`WHERE` 条件数量可变时很难写） | 强（链式按条件追加） |
| 学习曲线 | 中（要会写 SQL + 学 sqlc 注释规范） | 平缓 |
| 适合场景 | 查询固定、追求极致可控的系统 | 查询多变、快速迭代的系统 |

**为什么 coze 不选（推断 + 行业共识）**：

- **动态查询是硬伤**：coze 的后台管理、列表筛选场景充满"按 N 个可选条件组合查询"（有的传 spaceID、有的传 ownerID、有的传时间范围）。sqlc 的 SQL 是静态的，这种动态组合要么写一堆 SQL 变体，要么靠丑陋的 `WHERE (? IS NULL OR col = ?)` hack。GORM 的链式 `if cond { db = db.Where(...) }` 是这类场景的天然解。
- **JSON 列处理**：coze 重度用 JSON 列 + 序列化成 struct，sqlc 对此的支持不如 gen 配置友好。

**选错代价**：如果选了 sqlc——简单 CRUD 和报表类查询会很爽（SQL 完全可控、性能透明），但一旦遇到动态筛选，开发体验断崖下跌，最后很可能在 sqlc 之外又引入一个 SQL builder 来拼动态条件，技术栈反而更碎。**公平说**：如果 coze 是个查询模式高度固定、对 SQL 性能极度敏感的系统（比如金融对账），sqlc 的"零 ORM 开销 + SQL 完全可控"会很香。

### 4.3 sqlx

**它是什么**：`database/sql` 的薄封装。核心增强就一件事——把查询结果自动 Scan 进 struct（`StructScan`），省掉手动一列列 `Scan`。**它不是 ORM**，不帮你生成 SQL、不管关联、不管迁移。

**对比维度**：你还是得**手写每一条 SQL**。sqlx 只省了"把行塞进 struct"这一步。

**为什么 coze 不选**：抽象层级太低。coze 几十张表、CRUD 量巨大，用 sqlx 意味着手写海量 SQL + 维护海量 SQL 字符串，且 SQL 里的列名/类型**编译期完全不检查**（sqlx 的 Scan 是运行时反射匹配）。这恰恰是 coze 用 gen 要消灭的痛点。

**选错代价**：开发效率断崖式下降，且失去编译期类型安全——sqlx 的"类型安全"只到 struct Scan 这层，SQL 字符串本身全是运行时风险。对大型快速迭代项目是负担。**公平说**：sqlx 适合"SQL 不多、追求零魔法、团队 SQL 功底强"的中小项目，它的透明和轻量是优点。

### 4.4 原生 `database/sql`

**它是什么**：Go 标准库的数据库接口。只提供连接池、`Query`/`Exec`、`*sql.Rows`。**所有事都得自己做**：拼 SQL、`rows.Next()` 循环、手动 `Scan` 每一列、处理 NULL（用 `sql.NullString` 等）。

**为什么 coze 不选**：样板代码（boilerplate）爆炸。一个 CRUD 实体动辄几十行重复的 Scan 代码。在 coze 这种规模下完全不可维护。

**选错代价**：开发速度极慢、出错率极高（漏 Scan 一列、NULL 处理错），几乎不可能用它支撑 coze 的体量。**公平说**：`database/sql` 是一切的基座（GORM/sqlx 底层都是它），且在"只有几条 SQL、追求零依赖零抽象"的微型工具里它最干净。但它不是应用层数据访问的答案。

### 4.5 squirrel

**它是什么**：SQL builder（构建器）。用链式 Go 调用拼出 SQL **字符串**（`sq.Select("*").From("users").Where(sq.Eq{"id": 1})`），帮你安全拼接、防注入。但它**只负责拼 SQL，不负责 Scan**——拼出来的 SQL 还得交给 `database/sql`/sqlx 执行和扫描。

**为什么 coze 不选**：它只解决了"动态拼 SQL"这一半问题，另一半（Scan、关联、类型安全的字段引用）还得自己接别的库。GORM 的链式 API 把"拼 SQL + 执行 + Scan"一站式解决了，gen 还顺带给了类型安全。squirrel 相比之下是"半成品"。

**选错代价**：技术栈拼凑感强（squirrel + sqlx + 手动管理），且字段名仍是字符串、无编译期检查。**公平说**：squirrel 在"已经用裸 SQL，但被动态条件拼接折磨"的场景是个好补丁，它定位是工具不是框架。

### 4.6 xorm

**它是什么**：另一个 Go 全功能 ORM，曾经是 GORM 的主要竞争对手，功能也很全。

**为什么 coze 不选（行业共识）**：纯粹是**社区/生态强弱之争**。GORM 在过去几年的社区活跃度、文档质量、Star 增长、第三方插件丰富度上全面胜出，xorm 相对沉寂。在功能差不多的前提下，选生态更大的那个是理性选择。

**选错代价**：选了 xorm，未来遇到问题时社区答案更少、招的人更可能没用过、长期维护风险更高。**公平说**：xorm 本身是成熟可用的 ORM，技术上没大问题，输的是势能不是能力。

### 4.7 一张总表收口

| 方案 | 抽象层级 | 类型安全 | 动态查询 | 学习/招人成本 | 生态 | coze 不选的主因 |
|---|---|---|---|---|---|---|
| **GORM + gen** | 高（+生成增强） | 强（gen） | 强 | 低 | 最大 | （选了） |
| ent | 高 | 极强 | 较弱 | 高 | 中 | 侵入性高、学习曲线陡、JSON/动态场景别扭 |
| sqlc | 中（生成） | 极强 | 弱 | 中 | 中 | 动态查询是硬伤 |
| sqlx | 低 | 弱（仅Scan） | 需手写 | 低 | 中 | 抽象太低、要写海量 SQL、无列名检查 |
| database/sql | 最低 | 无 | 全手写 | 低 | 标准库 | 样板代码爆炸 |
| squirrel | 中（仅builder） | 弱 | 强 | 低 | 中 | 只解决拼 SQL 一半问题 |
| xorm | 高 | 中 | 强 | 低 | 弱于 GORM | 社区/生态势能输给 GORM |

---

## 5. 适用边界（何时该用 ent / sqlc / 原生，而不是 GORM）

**诚实地说，GORM+gen 不是万能解。** 给学生一个可操作的决策框架：

**该坚持 GORM + gen 的场景（coze 属于这类）**：
- 表多、迭代快、多人协作、需要编译期类型安全防回归
- 查询模式多变（大量动态条件筛选）
- 有 JSON 列存复杂结构
- 团队 Go 工程师多但不一定都是 SQL 专家
- 需要"小公司单库能跑、大公司加从库能扩"的弹性

**反而该选 ent 的场景**：
- 数据模型是**复杂的图/关系网**（社交关系、权限图、组织架构），需要优雅的图遍历
- schema 高度稳定、对类型安全要求到极致（一点运行时数据库错误都不能接受）
- 团队愿意 buy-in schema-first 范式、不在乎学习曲线
- 不太需要逃逸到原生 SQL

**反而该选 sqlc 的场景**：
- 查询模式**高度固定**（没什么动态条件）
- 对 SQL 性能/执行计划极度敏感，要求"我写的就是最终 SQL，零 ORM 开销和惊喜"
- 团队 SQL 功底强、乐意以 SQL 为真相
- 典型：报表系统、对账系统、数据密集型分析后端

**反而该用 sqlx / database/sql 的场景**：
- 项目极小、SQL 就那么几条
- 追求零依赖、零魔法、完全可控
- 写工具脚本、一次性数据迁移程序

**该额外引入 Atlas（而非用 GORM AutoMigrate）的场景**：
- **任何生产环境**。AutoMigrate 只适合本地开发/测试快速建表。一旦上生产、要支持版本升级和回滚，就该上 Atlas 这类版本化迁移工具——coze 正是这么做的。

一句话边界：**GORM+gen 是"通用快速迭代型业务后端"的最优默认值；ent/sqlc 是在「关系极复杂」或「查询极固定」两个极端场景的更优特化解。**

---

## 6. 对我们 vibe-studio 的启示

vibe-studio 对齐 coze 但 right-size，后端同样 GORM + MySQL。基于上面的拆解，给出可执行建议：

**直接照搬的（coze 已验证的好实践）**：
1. **GORM 做运行时引擎是稳的默认选择**——生态、招人、文档都最优，对一个常规全栈系统，没理由偏离主流。
2. **生产迁移别用 AutoMigrate**——哪怕项目小，养成用版本化 SQL 迁移的习惯（Atlas 或更轻的 golang-migrate）。原因：AutoMigrate 无法回滚、无版本记录、并发 DDL 有风险，版本化迁移才可控。
3. **sqlite 内存库跑 DAL 单测**——零依赖、CI 快，直接抄 coze 的 `MockDB` 思路。

**要 right-size / 慎重的（coze 的体量未必适合我们）**：
4. **gen 是否值得引入，看表的数量**。gen 的收益（编译期类型安全）随表数量和迭代频率放大。coze 几十张表用 gen 完全值得；vibe-studio 如果只有十几张表、且你一个人开发，gen 带来的"每次改 schema 要重新生成"的工作流成本可能不划算。**建议**：表少时先用 GORM 原生 + 谨慎的代码 review，表多到重构开始痛了再上 gen。这本身就是 right-size 的体现——**不为还没遇到的规模付现在的复杂度成本**。
5. **dbresolver 不要现在配**。coze 自己都只是预留没启用。vibe-studio 单库足够，读写分离是"等真有读压力了 5 分钟就能加"的东西，现在配纯属过度设计。

**值得深入理解的点**：
- 能讲清"GORM 弱类型痛点 → gen 如何用代码生成补上 → 编译期 vs 运行时检查的差异"，就比"我用过 GORM"深一层。
- 能讲"coze 为什么生产不用 AutoMigrate 而用 Atlas"，展示你懂迁移的工程风险，而不是只会调库。
- 能讲"dbresolver 是预留未启用"这种**源码级的诚实观察**，体现你真的读了源码、能区分"号称用了"和"实际用了"。
- 能横向对比 ent/sqlc 并说清各自适用边界，体现你不是只会一个工具，而是理解整条选型光谱。

---

## 7. 来源与核实状态（区分事实 / 推断）

### 7.1 事实（源码 / go.mod 直接可证，已核实）

| 结论 | 核实方式 | 路径 |
|---|---|---|
| go 1.24.0；gorm v1.25.11 / gen v0.3.26 / dbresolver v1.5.2 / driver/mysql v1.5.7 / driver/sqlite v1.4.3 | 读 go.mod | `backend/go.mod` |
| ORM 初始化极薄，`type DB = gorm.DB` 直接别名、无额外抽象层 | Read | `backend/infra/orm/database.go`、`infra/orm/impl/mysql/mysql.go` |
| 连接池参数走环境变量（idle 10 / open 100 / lifetime 3600s / idletime 600s 默认） | Read | `infra/orm/impl/mysql/mysql.go:45-48` |
| gen 是主力，136 个 .go 文件使用 | grep | 全 backend |
| gen 生成器配置 + 表/JSON 列映射 + 毫秒时间戳 | Read | `backend/types/ddl/gen_orm_query.go` |
| 生成的 model 字段是 `field.Int64/String` 强类型表达式 | Read | `domain/app/internal/dal/query/app_draft.gen.go` |
| 业务层用 `query.Use(db)` / `query.SetDefault(db)` 拿 DAO | grep | `domain/app/repository/app_impl.go` 等 |
| dbresolver 出现在 70 个文件，全是 .gen.go 模板 | grep | 全 backend |
| 运行时无 `dbresolver.Register`/`db.Use(dbresolver)`，业务层无 `.ReadDB()/.WriteDB()` 调用 | grep | 全 backend（结果为空） |
| sqlite 仅 1 个文件用，且仅测试（`:memory:`） | grep + Read | `internal/mock/infra/orm/sqlitedb.go` |
| 生产迁移走 Atlas（17 个版本化 SQL + HCL schema），AutoMigrate 仅测试用 | grep + ls | `docker/atlas/migrations/`、`docker/atlas/*.hcl` |

### 7.2 推断（基于事实的合理推理，coze 官方未明说）

| 推断 | 推理依据 | 置信度 |
|---|---|---|
| 选 GORM 含"团队熟悉度 / 招人友好"考量 | GORM 是 Go 社区事实标准（生态事实）+ 大厂理性选型惯例 | 高 |
| dbresolver 预留是为"小公司单库 / 大公司加从库"的私有化弹性 | 事实=代码预留未启用 + coze 是开源私有化项目 | 中高 |
| 不 dogfooding 自研 ORM 是因为 ORM 非字节对外秀肌肉的领域 | CloudWeGo 无官方 ORM（事实）+ 对比 Kitex/Hertz 的强 dogfooding | 中 |
| 用 Atlas 而非 AutoMigrate 是出于私有化场景"可审查可回滚"的负责任考量 | 事实=生产用 Atlas + AutoMigrate 生产的已知风险 | 高 |
| 不选 ent/sqlc 的具体权衡（侵入性、动态查询硬伤等） | 行业共识 + 各库设计特性，非 coze 文档明示 | 中高 |

### 7.3 待核实 / 本文未深入

- coze 是否在某些 domain 有手写原生 SQL（`db.Raw`）的逃逸用法——本文未系统统计，只确认了 gen 是主路径。
- `driver/postgres`（go.mod 里 indirect 存在）是否有任何代码路径真的用到 PostgreSQL，还是纯属全家桶依赖带入——**待核实**，初步判断未用。
- Atlas 迁移与 gen 的 model 定义之间如何保持一致（是 schema 先行还是 model 先行）——本文未深挖两者的同步工作流，**待核实**。
- gen 的 `Mode: WithoutContext` 选择背后的具体考量（性能 / 风格）——**待核实**，本文仅记录事实未展开动机。
