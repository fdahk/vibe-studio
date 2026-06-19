# 03 - MySQL：关系数据库选型理由

> 主题：MySQL 8.4（关系库主库）+ OceanBase 变体（向量存储，非关系库替代）+ sqlite（单测）
> 对标对象：coze-studio（字节扣子开源版）后端
> 本地源码核实路径：`/tmp/coze-studio`，DB 配置以 `docker/docker-compose.yml`、`docker/.env.example`、`docker/volumes/mysql/schema.sql`、`backend/infra/orm`、`backend/infra/oceanbase` 为准
> 面向想深入理解关系数据库选型本质的读者。

---

## 0. 一句话结论（先给答案，并先纠一个常见误解）

coze 的关系数据库主库是 **MySQL 8.4.5**，全部 55 张业务表（用户、Agent、工作流、插件、会话、权限……）都落在 MySQL；sqlite 只在单元测试里做内存 mock；schema 由 **Atlas** 这个迁移工具管理并 dump 成 `schema.sql`。

**这里要先纠正一个广泛流传、但与源码不符的说法**：很多人（包括本文写作前拿到的需求假设）以为 coze 提供的 **OceanBase 变体 = 把关系库 MySQL 换成 OceanBase**。**这是错的。** 源码证据明确显示：OceanBase 在 coze 里是 **向量存储（VectorStore）的三个可选实现之一**（`milvus` / `vikingdb` / `oceanbase`），跟 Milvus 是同一层、互为替代；它替换的是"向量检索引擎"，**不是关系库**。在 `docker-compose-oceanbase.yml` 里，`mysql:8.4.5` 服务依然原样保留并继续当关系库主库。

所以本文的真实结论是：**关系库选型上，coze 只押了 MySQL 一家，没有给关系库做多引擎抽象；OceanBase 出现在另一条赛道（向量库），是另一篇文档（向量数据库选型）的主角，本文只在第 3 节澄清它、不把它当关系库替代来讲。**

后面 7 段会把这个结论拆开，并诚实标注哪些是源码证据（事实），哪些是合理推断（动机）。

---

## 1. 是什么（扫盲：关系库 / 事务 / SQL）

### 1.1 关系数据库（RDBMS）解决什么

关系数据库的核心模型是「二维表 + 行 + 列 + 表间关系」。它的三个根本卖点：

- **结构化 + 强约束**：每张表有固定 schema（列名、类型、是否可空、唯一约束、外键）。脏数据在写入时就被挡住，而不是等读出来才发现。
- **关系 + JOIN**：数据拆成多张表（范式化），靠外键关联，查询时用 JOIN 拼回来。避免一份数据存多份导致的不一致。
- **声明式查询语言 SQL**：你描述"要什么"（`SELECT ... WHERE ...`），不描述"怎么拿"，由查询优化器决定走哪个索引、用什么 JOIN 算法。

### 1.2 事务与 ACID（关系库最值钱的东西）

事务（Transaction）是"一组要么全成功、要么全失败"的操作。它的保证缩写为 **ACID**：

| 字母 | 含义 | 通俗解释 | 没有它会怎样 |
|---|---|---|---|
| A | Atomicity 原子性 | 一组操作不可分割，中途失败全回滚 | 转账扣了钱没到账 |
| C | Consistency 一致性 | 事务前后数据满足所有约束 | 外键悬空、余额变负 |
| I | Isolation 隔离性 | 并发事务互不干扰（靠隔离级别 + 锁/MVCC） | 读到别人没提交的脏数据 |
| D | Durability 持久性 | 提交后断电也不丢（靠 redo log + 刷盘） | 提交成功了重启后没了 |

对 coze 这种平台型产品，**强事务和强关系是刚需**：发布一个工作流要同时改 `workflow`、`workflow_version`、`connector_workflow_version` 等多张表，必须原子提交；权限/空间/成员关系全靠表间关联表达。这类数据天然属于关系库，不是 KV 或文档库的舒适区。

### 1.3 为什么是 MySQL 而不是泛泛的"关系库"

MySQL 是开源关系库里装机量最大的一个，默认存储引擎 InnoDB 提供完整的 ACID 事务、行级锁、MVCC（多版本并发控制，读不阻塞写）、外键。8.x 之后还补齐了窗口函数、CTE（公共表表达式）、原生 JSON 类型等过去被诟病缺失的能力——这也是为什么 coze 敢在 schema 里大量用 JSON 列（见下节）。

---

## 2. 在 coze 里怎么用（源码举证）

### 2.1 主库：MySQL 8.4.5（事实）

`docker/docker-compose.yml`（也包括 oceanbase 变体）：

```yaml
mysql:
  image: mysql:8.4.5
  container_name: coze-mysql
  environment:
    MYSQL_DATABASE: ${MYSQL_DATABASE:-opencoze}
  command:
    - --character-set-server=utf8mb4
    - --collation-server=utf8mb4_unicode_ci
  volumes:
    - ./volumes/mysql/schema.sql:/docker-entrypoint-initdb.d/init.sql
```

`docker/.env.example` 里的连接配置（事实）：

```bash
export MYSQL_DSN="${MYSQL_USER}:${MYSQL_PASSWORD}@tcp(${MYSQL_HOST}:${MYSQL_PORT})/${MYSQL_DATABASE}?charset=utf8mb4&parseTime=True"
export MYSQL_MAX_IDLE_CONNS=10
export MYSQL_MAX_OPEN_CONNS=100
export MYSQL_CONN_MAX_LIFETIME=3600   # seconds
export MYSQL_CONN_MAX_IDLE_TIME=600   # seconds
```

几个值得注意的细节：
- **字符集统一 utf8mb4**：utf8mb4 才是 MySQL "真正的 UTF-8"（能存 emoji、4 字节字符），老的 `utf8` 是阉割版（最多 3 字节）。大厂规范几乎都强制 utf8mb4。
- **连接池参数显式配置**：`MaxOpenConns=100 / MaxIdleConns=10 / ConnMaxLifetime=3600s`。这是生产级的连接池治理——限制并发连接上限防打爆 DB，回收长连接避免被 MySQL 的 `wait_timeout` 单方面断开导致"已失效连接"。
- 连接由 GORM 的 mysql driver 建立：`backend/infra/orm/impl/mysql/mysql.go` 里 `gorm.Open(mysql.Open(dsn))`。

### 2.2 表规模与 JSON 列用法（事实）

`docker/volumes/mysql/schema.sql`：共 **55 张表**，其中有 **59 处 `json` 类型列**。例如 `agent_tool_draft.operation`（存 OpenAPI Operation schema）、`app_release_record.connector_ids` / `extra_info`。

这点很关键，是第 4 节"为什么不选 PostgreSQL"的直接反驳证据：**coze 确实需要半结构化字段，但 MySQL 8 的原生 JSON 类型已经够用**，没有非 PG 的 JSONB 不可的硬需求。

### 2.3 schema 由 Atlas 管理（事实）

`docker/atlas/` 目录：`atlas.hcl` + `opencoze_latest_schema.hcl` + `migrations/`（一串带时间戳的增量 SQL）+ `atlas.sum`。`docker/atlas/README.md` 显示工作流是 `atlas migrate diff` 生成增量、`dev-url "docker://mysql/8/"`（注意：连真 MySQL 8 容器做 diff，不是 sqlite）。最终 dump 出的全量 schema 就是 compose 启动时灌进去的 `schema.sql`。

含义：coze 的 DDL 是**版本化、可回溯、声明式**管理的，不是手写 SQL 拍上去。Atlas 把"目标 schema"和"当前 schema"做 diff 自动算迁移，属于现代 schema-as-code 实践。

### 2.4 OceanBase 变体 = 向量存储替代，不是关系库替代（事实，重点澄清）

`docker/.env.example`：

```bash
# VectorStore type: milvus / vikingdb / oceanbase
export VECTOR_STORE_TYPE="milvus"
```

`docker/docker-compose-oceanbase.yml` 里那段注释直说了：`# OceanBase for vector storage`。而该 compose 里 `mysql:8.4.5` 服务**依旧存在**。

代码层面（`backend/infra/oceanbase/oceanbase.go`）OceanBaseClient 的方法全是向量接口：

```go
func (c *OceanBaseClient) CreateCollection(ctx, collectionName, dimension int) error
func (c *OceanBaseClient) InsertVectors(ctx, collectionName, vectors []VectorResult) error
func (c *OceanBaseClient) SearchVectors(ctx, ..., topK int, threshold float64) ([]VectorResult, error)
```

它被注册在 `backend/infra/document/searchstore/impl/oceanbase/`（搜索/向量检索层），跟 Milvus 实现并排。**结论再强调一次：OceanBase 在 coze 里和 Milvus 同层，是知识库 RAG 的向量检索引擎选项，关系库始终是 MySQL。**

### 2.5 sqlite 只用于单测 mock（事实）

`backend/internal/mock/infra/orm/sqlitedb.go`：`newSQLiteDB(":memory:")`，配 `gorm.io/driver/sqlite`。即跑单测时用内存 sqlite 顶替 MySQL，零依赖、进程内、跑完即焚。

`backend/go.mod` 里的依赖定位也印证了边界：
- `gorm.io/driver/mysql`、`gorm.io/driver/sqlite` 是**直接依赖**（业务/测试在用）。
- `gorm.io/driver/postgres` 是 **indirect（间接依赖）**——被 Atlas 等工具链间接拉进来，**业务代码并不用 PG**。
- `github.com/pingcap/tidb/pkg/parser` 也在依赖里，但它是 **TiDB 的 SQL parser 库**（被拿来解析/校验用户在工作流"数据库节点"里写的 SQL），**不代表 coze 用 TiDB 当存储**。这是个容易被误读成"coze 用了 TiDB"的坑，特此标注。

---

## 3. 为什么官方选 MySQL（三类理由）

按"技术 / 生态-dogfooding / 平台-私有化"三类动机拆，并诚实区分事实与推断。

### 3.1 技术理由（部分事实 + 部分推断）

- **数据形态天然是关系型 + 强事务**（事实，由 schema 推得）：55 张表大量互相关联（draft/online/version 三态、各种关联表如 `agent_to_database`、`connector_workflow_version`），发布、版本化、权限都需要多表原子写。这是关系库的主场，MySDQL InnoDB 的行锁 + MVCC + 完整事务正好覆盖。
- **JSON 需求 MySQL 8 已满足**（事实）：59 处 JSON 列证明确有半结构化需求，但没强到要 PG 的 JSONB 索引/操作符。MySQL 8 原生 JSON 够用。
- **成熟度与确定性**（推断）：MySQL/InnoDB 二十年生产验证，行为可预期、踩坑点都被趟平、出问题能搜到答案。对一个要给上万企业私有化部署的产品，"无惊喜"本身就是巨大价值。

### 3.2 生态 / dogfooding 理由（强推断，符合常识）

- **国内 MySQL 生态最厚**（业界事实）：国内互联网后端事实标准就是 MySQL。DBA、运维、监控、备份、分库分表中间件（如 ShardingSphere）、慢查询分析工具、人才储备……全都围绕 MySQL 最成熟。换句话说，选 MySQL 的生态与运维成本最低。
- **字节内部体系大概率以 MySQL 为主**（推断，待核实具体内部组件）：字节内部数据库平台、DBA 工具链、容灾备份方案高度概率是 MySQL 系。开源版沿用团队最熟的栈，迁移/运维心智负担最小。这属于"用自己最顺手的工具"式 dogfooding。
- 注：本文不掌握字节内部具体 DB 平台命名/形态，上述标 **待核实**。

### 3.3 平台 / 私有化理由（强推断，且是本案最关键的一类）

coze-studio 是要**私有化部署给上万企业**的开源产品。这类产品的 DB 选型，"客户能不能运维得起"权重极高：

- **客户侧 MySQL 运维能力最普及**（推断，符合常识）：随便一家有 IT 团队的企业都能运维 MySQL，能找到会 MySQL 的人、有现成备份/监控方案。换成小众库，等于给每个客户的运维都加负担，拖累私有化落地。
- **已有体系兼容**（推断）：很多目标客户内部本就有 MySQL 集群/规范，coze 用 MySQL 能直接接入他们既有的备份、审计、容灾体系。
- **降低支持成本**（推断）：客户出 DB 问题时，MySQL 的问题官方和社区都更容易远程支持。

> 一句话：私有化产品的 DB 选型，是在"我自己技术上想要什么"和"我的几千个客户运维得动什么"之间做权衡，后者往往压倒前者。MySQL 是这道题的"最大公约数"。

### 3.4 那为什么还要做 OceanBase 变体？（澄清动机，非关系库）

再次强调：OceanBase 变体动的是**向量库**那一层，不是关系库。它存在的动机（推断）：

- **国产化 / 信创需求**：部分政企、金融客户有"用国产数据库"的合规要求，OceanBase（蚂蚁）是头部国产分布式数据库，提供它能接这类客户。
- **超大规模向量场景**：OceanBase 支持向量检索且能水平扩展，面向知识库数据量极大、又不想额外引入 Milvus 这类专用向量库的客户，可以用一套 OceanBase 同时承载（向量）。
- 关系库这一层 coze 并没有因此动摇——OceanBase 兼容 MySQL 协议，但 coze 没把关系库主库切到它。

---

## 4. 同类替代逐个对比 + 为什么不选（含选错代价）

对比维度固定为：**事务/一致性、国内生态&团队熟悉度、私有化运维门槛、特性增量、对 coze 的净收益**。

| 候选 | 类型 | 相对 MySQL 的核心差异 | coze 为什么不选（净收益判断） |
|---|---|---|---|
| **PostgreSQL** | 关系库 | 功能更强：JSONB（带索引/操作符）、pgvector、更严谨的类型与约束、更强的标准 SQL 兼容 | coze 的 JSON 需求 MySQL 8 已覆盖；向量另有专门方案（Milvus/OceanBase）不靠 pgvector；国内生态/团队熟悉度/字节既有体系都偏 MySQL → PG 的特性增量对 coze **用不上**，却要付生态切换成本 |
| **OceanBase（当关系库）** | 分布式关系库 | MySQL 兼容、金融级、超大规模水平扩展 | coze 关系库根本没到需要分布式的量级（55 张表、单实例 MySQL 足够）；上分布式库 = 运维复杂度暴涨，私有化中小客户扛不住。coze 只在**向量层**按需提供它 |
| **TiDB** | NewSQL | MySQL 兼容、自动水平扩展、HTAP | 水平扩展强但**运维重、资源占用大**（PD/TiKV/TiDB 多组件，起步就要一票节点）；中小私有化客户部署不起。"为了将来可能用得上的扩展性，现在让所有客户多扛几台机器"不划算 |
| **MariaDB** | MySQL 分支 | 协议/语法高度兼容 MySQL，部分特性领先 | 与 MySQL 差异小、收益不明显，却放弃了 MySQL 官方版的确定性和最广生态 → **零增量、负迁移收益** |
| **MongoDB** | 文档型 NoSQL | schema-free、文档模型、水平扩展易 | coze 数据是**强关系 + 强事务**（多表关联、原子发布、权限）。文档库做这类一致性要绕路、且关系/JOIN 是弱项 → 模型错配 |
| **SQLite** | 嵌入式关系库 | 零依赖、单文件、进程内 | 不支持高并发写、无网络访问、单写者锁 → 当生产主库不可能。coze 正确地只把它用在**单测内存 mock** |

### 4.1 选错的代价（反事实推演）

- **错选 PG**：得重做团队/客户对 PG 的运维能力建设，国内招 PG DBA 比 MySQL 难，私有化客户里会 PG 的更少；换来的 JSONB/pgvector coze 又不需要 → 纯负收益。
- **错选 TiDB/OceanBase 当主库**：每个私有化客户的部署门槛从"装一个 MySQL"变成"装一套分布式集群"，中小客户直接劝退，私有化铺量目标受挫。这是平台型产品最致命的代价——**不是技术跑不动，是客户装不起、运维不动**。
- **错选 MongoDB**：强一致写要在应用层补偿（手写两阶段/补偿事务），权限和关系查询写起来扭曲，长期维护成本高，且数据一致性 bug 难查。

---

## 5. 适用边界（什么时候反而该上 PG / TiDB / Mongo）

诚实地说，MySQL 不是万能答案。换个场景结论会反转：

- **该上 PostgreSQL**：重度依赖复杂查询/GIS（PostGIS）/JSONB 深度索引/想用 pgvector 一站式做向量、或对 SQL 标准/类型严谨性要求高的团队。如果是从零起步、团队对两者都熟，PG 在"功能上限"和"严谨性"上更优。
- **该上 TiDB**：单表/单库数据量真的撑爆单机 MySQL（TB 级、写入吞吐极高），且团队有能力运维分布式集群、机器预算充足，需要弹性水平扩展和 HTAP（同库跑 OLTP+OLAP）。
- **该上 OceanBase**：超大规模 + 金融级一致性 + 国产化合规三者叠加的场景（典型是大型政企/金融）。
- **该上 MongoDB**：数据天生是松散文档、schema 频繁变、几乎没有跨文档强事务需求（如日志、CMS 内容、设备上报）。
- **该用 SQLite**：单测、桌面/移动端本地存储、嵌入式、CLI 工具的轻量持久化。

判断口诀：**先看数据形态（关系 vs 文档）、再看一致性需求（强事务 vs 最终一致）、再看规模（单机够不够）、最后看谁来运维（团队/客户能力）。** coze 这四问的答案都指向单机 MySQL。

---

## 6. 对我们 vibe-studio 的启示

vibe-studio 对齐 coze、也用 MySQL，这个方向是对的。具体落地建议：

1. **关系库就用 MySQL 8，别折腾**。我们的数据形态（项目、画布、组件、用户、权限）和 coze 同构，强关系 + 需要事务，单机 MySQL 完全够，不要为"将来可能扩展"提前上 TiDB/分布式。
2. **学 coze 把连接池参数显式写进配置**（MaxOpen/MaxIdle/ConnMaxLifetime），别用 driver 默认值裸跑——默认无上限连接数能在压测时打爆 DB。
3. **字符集统一 utf8mb4 + utf8mb4_unicode_ci**，从建库就定死，避免后期存 emoji/特殊字符踩坑、避免不同表 collation 不一致导致 JOIN 报错。
4. **schema 用迁移工具版本化管理**（Atlas 或 golang-migrate/Prisma migrate 视我们的栈而定），不要手写 SQL 直接拍生产。DDL 要可回溯、可 review。
5. **半结构化字段优先用 MySQL 8 原生 JSON 列**，不要因为"听说 PG 的 JSONB 更强"就切库——除非真的需要对 JSON 内部字段建索引并高频查询。
6. **向量/RAG 是另一条赛道**：如果 vibe-studio 要做知识库检索，向量存储单独选型（Milvus 等），不要试图让 MySQL 兼任，更不要把"OceanBase 变体"误解成关系库方案。
7. **测试用内存 sqlite mock**（配合 GORM 的 sqlite driver）可借鉴，但要注意 sqlite 与 MySQL 的 SQL 方言差异（如某些函数、JSON 行为不同），强依赖 MySQL 特性的逻辑还是得跑集成测试连真 MySQL。

---

## 7. 来源与核实状态

### 7.1 事实（已在本地源码核实）

| 结论 | 证据路径 |
|---|---|
| 关系库主库是 MySQL 8.4.5 | `docker/docker-compose.yml`、`docker/docker-compose-oceanbase.yml`（均含 `image: mysql:8.4.5`） |
| 连接 DSN + 连接池参数显式配置 | `docker/.env.example`（MYSQL_DSN、MaxOpen/Idle/Lifetime） |
| 共 55 张表、59 处 JSON 列 | `docker/volumes/mysql/schema.sql`（grep 统计） |
| schema 由 Atlas 管理并 dump | `docker/atlas/`（atlas.hcl、migrations/、atlas.sum、README） |
| OceanBase = 向量存储选项（非关系库） | `docker/.env.example`（`VECTOR_STORE_TYPE: milvus/vikingdb/oceanbase`）、`backend/infra/oceanbase/oceanbase.go`（全是向量接口）、`backend/infra/document/searchstore/impl/oceanbase/` |
| oceanbase 变体里 MySQL 仍在 | `docker/docker-compose-oceanbase.yml` 仍有 mysql 服务 |
| sqlite 仅用于单测内存 mock | `backend/internal/mock/infra/orm/sqlitedb.go`（`:memory:`） |
| postgres 是 indirect 依赖、业务不用 | `backend/go.mod`（`gorm.io/driver/postgres ... // indirect`） |
| TiDB 仅作 SQL parser 库、非存储 | `backend/go.mod`（`github.com/pingcap/tidb/pkg/parser`） |
| GORM mysql driver 建连 | `backend/infra/orm/impl/mysql/mysql.go`（`gorm.Open(mysql.Open(dsn))`） |

### 7.2 推断（合理但未在源码中明文写出动机）

- "选 MySQL 是因为国内生态最厚 / 私有化客户运维门槛低 / 字节内部体系以 MySQL 为主"——**推断**，符合行业常识与私有化产品规律，但源码不会写动机。
- "OceanBase 变体是为国产化合规 + 超大规模向量客户"——**推断**。

### 7.3 待核实（涉及外部产品具体能力/数字，本文未独立验证）

- 字节内部数据库平台/DBA 工具链的具体形态与命名 —— **待核实**。
- OceanBase 的具体向量能力指标（支持的索引类型、维度上限、性能数字）—— **待核实**。
- TiDB 集群最小部署规模、资源占用的具体数字 —— **待核实**。
- MySQL 8 JSON 与 PostgreSQL JSONB 在索引能力上的精确差异边界 —— **待核实**（本文只用到"MySQL 8 原生 JSON 已满足 coze 需求"这一较弱论断，该论断由 schema 证据支撑）。
