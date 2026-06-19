# 05 - MinIO：对象存储选型理由

> 主题：`backend/infra/storage` 抽象层 + 3 种实现（MinIO / AWS S3 / 火山 TOS），靠环境变量 `STORAGE_TYPE` 切换，开箱默认 **MinIO**
> 对标对象：coze-studio（字节扣子开源版）后端
> 本地源码核实路径：`/tmp/coze-studio`，版本以 `backend/go.mod` 为准；接口看 `backend/infra/storage/storage.go`，实现看 `backend/infra/storage/impl/`，部署看 `docker/docker-compose.yml` 与 `docker/.env.example`
> 全文区分三类动机：**纯技术动机**、**生态/dogfooding 动机**（字节自家生态）、**平台/私有化动机**（coze 是可私有化部署给企业的平台，金融/医疗等场景要求数据不出域）。凡"确定事实"给源码佐证，凡"合理推断的动机"明确标注，不确定的数字标"待核实"。

---

## 0. 一句话结论（先给答案再讲为什么）

coze 没有"钦定"一家对象存储，而是**自研了一层薄抽象 `storage.Storage`（8 个方法的接口），底下挂 3 种实现——MinIO / AWS S3 / 火山 TOS，靠一个环境变量 `STORAGE_TYPE` 切换，开箱默认用 MinIO**。

这件事的本质**不是"MinIO 性能天下第一"，而是两条约束叠加的结果**：

1. **私有化约束**：coze 是要部署进客户机房的平台，金融/医疗/政企客户要求"数据不出域"，对象存储必须能**自托管**——这就直接排除了"只能跑在公有云上的 OSS/S3 服务"作为**默认值**的可能。
2. **生态约束**：自托管对象存储里，只要选了**说 S3 协议**的那一家，coze 的存储代码就能一套对接 MinIO（自托管）/ AWS S3（出海客户用 AWS）/ 任何 S3 兼容服务，切换零改码。MinIO 恰好是"自托管 + S3 兼容 + 单二进制易部署"三者交集里最成熟的那个。

所以默认选 MinIO 是 **"私有化必须自托管" × "S3 协议统一生态"** 两个约束的交点，TOS 实现则是字节自家生态的 dogfooding（公有云形态下用火山引擎自己的对象存储）。下面逐层拆。

---

## 1. 对象存储是什么（先把三种存储模型分清）

要理解为什么是"对象存储"而不是"往磁盘写文件"，得先把存储的三种模型摆开。它们不是谁取代谁，而是**面向不同访问模式**。

### 1.1 三种存储模型对比

| 维度 | 块存储（Block） | 文件存储（File） | 对象存储（Object） |
|---|---|---|---|
| 抽象单位 | 定长块（sector/block） | 文件 + 目录树 | 对象（key → bytes + metadata） |
| 访问接口 | SCSI / NVMe / iSCSI（设备级） | POSIX / NFS / SMB（路径级） | HTTP REST API（`GET/PUT/DELETE`） |
| 命名空间 | 无（裸设备） | 树状层级（`/a/b/c.txt`） | **扁平 key 空间**（"目录"只是 key 前缀） |
| 能否原地改 | 能（随机读写任意偏移） | 能（`seek`+`write`） | **不能**（对象不可变，改 = 整体覆盖） |
| 元数据 | 几乎没有 | 文件系统级（权限/时间戳） | **丰富、可自定义**（Content-Type、Tagging、用户元数据） |
| 扩展性 | 单卷有上限 | 受单机/集群文件系统限制 | **水平扩展到 PB/EB，对象数近乎无限** |
| 典型产品 | EBS、云硬盘、Ceph RBD | NFS、CephFS、NAS | S3、OSS、TOS、MinIO、Ceph RGW |
| 适合场景 | 数据库、虚拟机磁盘 | 共享目录、传统应用 | 海量非结构化数据：图片/视频/日志/备份/AI 产物 |

一句话辨析：

- **块存储**给你"一块裸盘"，你爱在上面装文件系统还是跑数据库随意，最底层、最灵活，但**没有应用语义**。
- **文件存储**给你"一棵目录树 + POSIX 语义"，进程能像本地文件一样 `open/seek/write`，适合需要**就地随机读写 + 多机共享**的传统应用。
- **对象存储**把每个文件当成一个**不可变的对象**，用一个全局唯一的 **key** 寻址，通过 **HTTP API** 存取，自带**丰富元数据**和**近乎无限的水平扩展**。代价是**不能原地改**（改一个字节也要整体覆盖）、**不保证 POSIX 语义**（不能 `seek` 到对象中间写）。

### 1.2 对象存储为什么适合 coze 这类平台

coze 要存的东西是什么？用户上传的头像/图片、知识库原始文档、插件图标、工作流运行产物、（在 vibe-studio 里将来还有）沙箱产物和视频资产。这些数据的共同特征：

- **非结构化、体积大、数量多**（典型海量小文件 + 偶发大文件）；
- **写一次读多次、几乎不就地改**（上传完就是只读，要改就是重新上传一份新 key）；
- **要能直接给浏览器**（前端拿一个 URL 就能下载/预览，最好不经过后端转发）。

这正好命中对象存储的能力面：扁平 key 寻址 + 不可变对象 + HTTP 直接可达 + 海量水平扩展。如果用文件系统硬扛，你要**自己造一遍对象存储的轮子**——自己生成可访问 URL、自己做鉴权、自己做多副本、自己做扩容——这就是第 4 节"为什么不选本地文件系统"的核心论点。

### 1.3 S3 协议是什么（为什么它是事实标准）

**S3（Simple Storage Service）** 是 AWS 2006 年推出的对象存储服务，是云上对象存储的开山鼻祖。它定义了一套基于 **HTTP REST** 的 API：

- 资源模型：`Bucket`（桶，命名空间）→ `Object`（对象，key + 数据 + 元数据）；
- 核心操作：`PutObject` / `GetObject` / `DeleteObject` / `HeadObject` / `ListObjects`（这正是 coze `storage.Storage` 接口里那几个方法的来源）；
- 鉴权：**AWS Signature V4**（请求签名算法）；
- 高级能力：分片上传（multipart upload）、**预签名 URL（presigned URL）**、对象标签（tagging）、版本控制、生命周期策略等。

**关键点：S3 不只是 AWS 的产品名，它的 API 已经成为对象存储的事实标准协议。** 后来者——阿里云 OSS、火山 TOS、MinIO、Ceph 的 RGW 网关——几乎都提供"**S3 兼容 API**"，意思是：你用为 AWS S3 写的客户端代码、SDK、工具（如 `aws-cli`、`s3fs`），**改个 endpoint 地址就能连它们**。这是整个对象存储生态能"一套代码打通"的根基，也是 coze 能用一层薄抽象兼容三家的前提。

> 类比理解：S3 之于对象存储，约等于 SQL 之于关系数据库、POSIX 之于文件系统——一个被广泛实现的"接口标准"，让上层代码不被单一厂商锁死。

---

## 2. 在 coze 里怎么用（源码举证）

### 2.1 一层接口：`storage.Storage`

抽象定义在 `backend/infra/storage/storage.go`。接口本身就是一份"对象存储能力清单"，且**命名直接对齐 S3 语义**：

```go
type Storage interface {
    PutObject(ctx, objectKey, content []byte, opts ...PutOptFn) error
    PutObjectWithReader(ctx, objectKey, content io.Reader, opts ...PutOptFn) error
    GetObject(ctx, objectKey) ([]byte, error)
    DeleteObject(ctx, objectKey) error
    GetObjectUrl(ctx, objectKey, opts ...GetOptFn) (string, error)  // 预签名 URL
    HeadObject(ctx, objectKey, opts ...GetOptFn) (*FileInfo, error)
    ListAllObjects(ctx, prefix, opts ...GetOptFn) ([]*FileInfo, error)
    ListObjectsPaginated(ctx, input, opts ...GetOptFn) (*ListObjectsPaginatedOutput, error)
}
```

注意 `GetObjectUrl` 返回**预签名 URL**——这正是 1.2 里"前端拿个 URL 直接下载、不经后端转发"的落地点。`option.go` 里还把 `ContentType`、`ContentDisposition`、`Tagging`、`Expires`、分页 `Cursor` 等都抽象成了 functional options，**这些恰好是 S3 协议的字段**，反过来印证整套抽象是按 S3 模型设计的。

### 2.2 多实现：MinIO / S3 / TOS 三套，目录就摆在那

```
backend/infra/storage/impl/
├── storage.go          # 工厂：按 STORAGE_TYPE 选实现
├── minio/minio.go      # MinIO 实现，用 minio-go/v7
├── s3/s3.go            # AWS S3 / 通用 S3 兼容实现，用 aws-sdk-go-v2
├── tos/tos.go          # 火山 TOS 实现，用 volcengine/ve-tos-golang-sdk
└── internal/fileutil   # 公共 URL 拼装等工具
```

`go.mod` 里三套 SDK 都在场，互相印证：

```
github.com/minio/minio-go/v7 v7.0.90
github.com/aws/aws-sdk-go-v2/service/s3 v1.84.1
github.com/volcengine/ve-tos-golang-sdk/v2 v2.7.17
```

### 2.3 切换机制：一个环境变量 `STORAGE_TYPE`

工厂在 `backend/infra/storage/impl/storage.go`：

```go
func New(ctx context.Context) (Storage, error) {
    storageType := os.Getenv(consts.StorageType) // "STORAGE_TYPE"
    switch storageType {
    case "minio":
        return minio.New(ctx, os.Getenv(MinIOEndpoint), os.Getenv(MinIOAK),
            os.Getenv(MinIOSK), os.Getenv(StorageBucket), envkey.GetBoolD("MINIO_USE_SSL", false))
    case "tos":
        return tos.New(ctx, ak, sk, bucket, endpoint, region)
    case "s3":
        return s3.New(ctx, ak, sk, bucket, endpoint, region)
    }
    return nil, fmt.Errorf("unknown storage type: %s", storageType)
}
```

业务层只依赖 `storage.Storage` 接口，**完全不知道底下是 MinIO 还是 S3 还是 TOS**——这是标准的"依赖倒置 + 工厂选实现"。同文件里还有个 `NewImagex`，给图片处理（ImageX）走同样的多实现切换，结构一致。

### 2.4 默认值：MinIO（不是 S3，也不是 TOS）

`docker/.env.example` 与 `docker/.env.debug.example` 都明确：

```
export STORAGE_TYPE="minio"   # minio / tos / s3
export STORAGE_BUCKET="opencoze"
export MINIO_AK=$MINIO_ROOT_USER
export MINIO_SK=$MINIO_ROOT_PASSWORD
export MINIO_ENDPOINT="minio:9000"
```

且 `docker/docker-compose.yml`（约 203 行起）**真的起了一个 minio 容器**：

```yaml
minio:
  image: minio/minio:RELEASE.2025-06-13T11-33-47Z-cpuv1
  container_name: coze-minio
  # entrypoint 里用 mc 自动建 bucket（opencoze）、灌入默认图标，
  # 然后 exec minio server /data --console-address ":9001"
```

也就是说**开箱即用的形态就是"附带一个自托管 MinIO"**，S3/TOS 是给"已经在用公有云对象存储的部署者"留的切换口。这是判断"官方默认选谁"的硬证据。

### 2.5 一个值得注意的实现细节：S3 实现其实是"通用 S3 兼容客户端"

读 `impl/s3/s3.go` 会发现，它并没有写死连 AWS 官方 endpoint，而是用了**自定义 EndpointResolver**：

```go
customResolver := aws.EndpointResolverWithOptionsFunc(func(service, region string, ...) (aws.Endpoint, error) {
    return aws.Endpoint{URL: endpoint, SigningRegion: region, Source: aws.EndpointSourceCustom}, nil
})
cfg, _ := config.LoadDefaultConfig(ctx,
    config.WithCredentialsProvider(creds),
    config.WithEndpointResolverWithOptions(customResolver),
    config.WithRegion("auto"))
```

含义：这个"s3"实现的 endpoint 是从环境变量读的，所以它**不光能连 AWS S3，也能连任何 S3 兼容服务**（阿里云 OSS 的 S3 兼容端点、Cloudflare R2、甚至另一套 MinIO）。这恰好把第 1.3 节"S3 是事实标准协议"落到了代码上——一套 `aws-sdk-go-v2` 代码就是整个 S3 生态的通用客户端。（MinIO 实现单独用 `minio-go` 而非走这条，属于"原生 SDK 体验更顺手"的选择，二者都说 S3 协议。）

---

## 3. ★ 为什么官方默认选 MinIO（技术 + 生态 + 平台三类理由）

这是全文重点。把动机拆成三类，并诚实标注哪些是源码/常识能确证的事实、哪些是合理推断。

### 3.1 纯技术动机

**(A) S3 兼容 API —— 一套抽象对接整个生态，切换不改码**（确定事实，可由源码佐证）

MinIO 原生说 S3 协议。coze 的 `storage.Storage` 接口完全按 S3 语义设计（见 2.1），于是：

- 默认自托管用 MinIO；
- 出海客户用 AWS，把 `STORAGE_TYPE` 改 `s3`、填 AWS 凭证即可；
- 想接其它 S3 兼容服务，复用 `s3` 实现改 endpoint 即可（见 2.5）。

**业务代码一行不动**。这就是"S3 兼容"的真正价值：它不是某个 feature，而是**让存储后端变成可替换的插件**。如果 MinIO 不说 S3 协议，coze 就得为它单独写一套非标准 API 适配，多实现抽象的意义会被大幅削弱。

**(B) 可自托管 / 部署轻量 —— single binary，Go 写**（确定事实）

MinIO 是一个用 **Go 写的单二进制**对象存储服务器，`minio server /data` 一行就能起一个。它没有外部依赖（不需要单独的元数据数据库、不需要 ZooKeeper 之类协调组件），直接把对象和元数据落到给定的数据目录/磁盘。这一点在 compose 里看得很直白：一个 `minio/minio` 镜像、一个 `/data` 卷、一条 `exec minio server` 就跑起来了。

对比要上 Ceph：Ceph 至少要 MON（监视器）+ OSD（对象存储守护进程）+ MGR（管理器），再加 RGW 网关才能提供 S3 接口，节点和组件数量是另一个量级。MinIO 的"轻"对**开箱体验**和**中小规模私有化**是决定性的。

**(C) 性能**（合理推断，具体数字待核实）

MinIO 官方宣称面向高吞吐场景做了优化（纠删码、并行 IO 等），社区 benchmark 也常把它作为高性能自托管对象存储的代表。但**具体的吞吐/延迟数字依赖硬件、网络、对象大小分布，本文不引用未经本环境验证的数据，标注「待核实」**。对 coze 当前规模而言，性能并非选型的第一约束——可自托管和 S3 兼容才是。

### 3.2 生态 / dogfooding 动机（字节自家生态）

**TOS 实现的存在 = 字节自家生态的 dogfooding**（事实：实现存在；推断：动机是自用火山引擎）

`go.mod` 里有 `volcengine/ve-tos-golang-sdk`，`impl/tos/tos.go` 里用 `tos.NewClientV2` 接火山引擎对象存储（TOS, Tinder Object Storage）。**这是确定事实**：coze 把字节自家云（火山引擎）的对象存储做成了一等公民实现。

**合理推断**：字节内部/火山引擎托管形态下，自然倾向于用自家的 TOS——既是吃自家狗粮（dogfooding），也能复用火山的运维、计费、合规体系。但**默认值仍是 MinIO 而非 TOS**，说明开源版的首要目标是"任何人都能自托管跑起来"，而不是"绑死火山引擎"。TOS 是给"部署在火山引擎上的用户"准备的最优路径，不是强制项。

> 这里和本仓库 04 篇（消息队列 eventbus 多实现）是同一种设计哲学：字节开源项目普遍做"抽象层 + 多实现 + env 切换 + 默认轻量自托管 + 保留自家云实现"。MinIO/S3/TOS 对应 NSQ/Kafka/RocketMQ/Pulsar/NATS 那套思路。

### 3.3 平台 / 私有化动机（最关键的那条约束）

**coze 是可私有化部署的平台，对象存储必须能自托管**（事实：coze 是私有化平台 + compose 自带 minio；推断:默认选 MinIO 主要由此驱动）

这是默认值选 MinIO 的**根本原因**，也是它压过"纯技术性能"的地方：

- coze 定位是部署给企业的 Agent 平台。金融、医疗、政企等行业有**数据主权 / 数据不出域**的强合规要求——用户上传的文档、知识库内容、对话产物**不能进公有云**。
- 这就要求对象存储**必须能跑在客户自己的机房/私有网络里**。AWS S3、阿里云 OSS 这类"只作为公有云服务存在"的产品，**天然不满足**这个约束，因此**不能当默认值**（它们只能作为"客户自愿用公有云"时的可选项）。
- 在"能自托管 + 说 S3 协议 + 部署够轻"的候选里，MinIO 是最成熟、社区最大的那个。于是它成了默认。

**一句话**：默认选 MinIO，技术层是因为"S3 兼容让生态统一"，但拍板的那一票是**平台的私有化/合规约束**——必须有一个开箱即用、能装进客户机房的自托管对象存储。

---

## 4. ★ 同类替代逐个对比 + 为什么不选（含选错的技术代价）

把候选分成三档看：**公有云托管服务**（S3 / OSS / TOS）、**自托管分布式存储**（Ceph / SeaweedFS / MinIO）、**不用对象存储**（本地文件系统）。

### 4.1 总览表

| 候选 | 形态 | S3 兼容 | 能自托管 | 部署/运维成本 | 成熟度/生态 | 作为 coze 默认的结论 |
|---|---|---|---|---|---|---|
| **MinIO** | 自托管对象存储 | ✅ 原生 | ✅ single binary | 低 | 高（自托管对象存储事实标准） | ✅ **选作默认** |
| AWS S3 | 公有云服务 | ✅（它就是标准） | ❌ | 无（托管） | 极高 | ⚠️ 仅作可选实现（私有化不接受） |
| 阿里云 OSS | 公有云服务 | ✅ 兼容端点 | ❌ | 无（托管） | 高（国内） | ❌ 不进默认（同 S3，且更绑国内云） |
| 火山 TOS | 公有云服务 | ✅ 兼容 | ❌ | 无（托管） | 中高 | ⚠️ 作可选实现（字节自家生态） |
| Ceph (RGW) | 自托管分布式存储 | ✅（RGW 网关） | ✅ | **极高** | 高（但偏 IaaS） | ❌ 运维过重，场景过度 |
| SeaweedFS | 自托管分布式存储 | ✅ 部分兼容 | ✅ | 低 | 中（不如 MinIO 成熟） | ❌ 成熟度/生态不足 |
| 本地文件系统 | 单机目录 | ❌ | ✅（但不是对象存储） | 极低 | —— | ❌ 单机、不可水平扩展、要自造一切 |

### 4.2 为什么不选 AWS S3 / 阿里云 OSS（作为默认）

它们本身没问题——**S3 就是协议标准本身，OSS 是国内成熟方案**。问题在于**形态**：它们是**只存在于公有云的托管服务，无法装进客户私有机房**。

- 把它当默认 = 强制所有私有化客户把数据交给公有云 = 直接违反金融/医疗的数据不出域要求。
- **选错的代价**：如果 coze 默认绑死某家公有云 OSS，那么(1) 数据主权敏感的客户根本无法采用，平台丢失整个 toB 高合规市场；(2) 形成厂商锁定，迁移成本极高。

所以它们的正确定位是"**可选实现**"——客户**自愿**用公有云时，改 `STORAGE_TYPE` 即可（S3 实现还顺带兼容 OSS 的 S3 端点，见 2.5）。这正是 coze 的做法。

### 4.3 为什么不选 Ceph（运维过重、场景过度）

Ceph 是"对象 + 块 + 文件"三合一的统一分布式存储，能力上是真正的重型武器（其 RGW 网关也提供 S3 兼容接口）。但对 coze 这个场景：

- **运维成本不在一个量级**：Ceph 要部署 MON / OSD / MGR，再加 RGW 才有 S3 接口，节点规划、CRUSH map、故障恢复都需要专业 SRE。MinIO 是 `minio server /data` 一行。
- **能力严重过剩**：coze 只需要"对象存储"这一种能力，根本不需要块存储和文件存储。为了一个对象存储桶引入一整套 IaaS 级存储系统，是典型的 over-engineering。
- **选错的代价**：默认就上 Ceph，会把"开箱即跑"变成"先招个存储工程师"。绝大多数私有化客户的对象存储需求（几 TB ~ 几十 TB、海量小文件）根本用不到 Ceph 的规模。**Ceph 适合的是"我本来就在自建私有云/IaaS"的客户**——那是第 5 节的适用边界。

### 4.4 为什么不选 SeaweedFS（轻但不够成熟/生态弱）

SeaweedFS 也是 Go 写的自托管存储，主打海量小文件、架构轻量，部分兼容 S3——和 MinIO 是最接近的对手。不选它主要是**成熟度与生态**：

- **S3 兼容完整度**：MinIO 的 S3 兼容覆盖更全、行为更贴近 AWS；SeaweedFS 的 S3 网关兼容度相对弱一些（部分高级特性支持有限，**具体差异待核实**）。
- **生态与认知度**：MinIO 已是"自托管 S3"的事实代名词，文档、社区、第三方集成、客户认知都更广；SeaweedFS 社区规模较小。
- **客户接受度**：私有化交付时，"我们用 MinIO"比"我们用 SeaweedFS"更容易被客户运维团队接受——这本身就是降低交付摩擦的因素。

**选错的代价**：选一个 S3 兼容不够完整的后端，意味着 coze 的存储抽象（预签名 URL、tagging、分片上传等）可能在某些实现上行为不一致，需要为它写特例适配，**侵蚀了"多实现共用一套抽象"的整洁性**。在没有强烈理由（如极端小文件规模）时，选成熟度更高的 MinIO 是更稳的工程决策。

### 4.5 为什么不选本地文件系统（看似最简单，实则要重造一切）

"直接 `os.WriteFile` 写到磁盘不就行了？" —— 这是最常见、也最危险的诱惑。代价是你要**把对象存储的核心能力一个个手搓出来**：

| 对象存储免费给你的 | 用本地文件系统你得自己造 |
|---|---|
| 水平扩展（加机器加容量） | 单机磁盘满了就完，多机要自己分片/路由 |
| 多副本 / 纠删码（坏盘不丢数据） | 自己做副本同步、自己处理一致性 |
| HTTP 直达 + 预签名 URL | 自己起文件服务、自己签 URL、自己做过期 |
| 访问鉴权 / 对象标签 / 元数据 | 自己设计权限模型、自己存元数据 |
| 标准 S3 API（生态工具直接用） | 没有标准接口，所有工具都得定制 |

- **架构硬伤**：本地文件系统**绑死单机**。coze 后端一旦多副本/水平扩展部署，"文件写在哪台机器、另一台怎么读"立刻就是问题——要么挂 NFS（又回到文件存储的共享与一致性难题），要么自己造分布式。
- **选错的代价**：本地文件系统在 demo 阶段最省事，但它是**技术债的典型来源**——等你需要扩容、需要给前端发 URL、需要鉴权时，会发现自己在重新实现一个简陋且不可靠的 MinIO。**正确做法是直接用对象存储**，本地开发也用 MinIO 容器，保持"开发=生产"的存储语义一致。

---

## 5. 适用边界（什么时候不该用 MinIO，该用别的）

选型没有银弹。MinIO 是 coze 默认场景下的最优解，但有明确的边界。

**该直接用公有云 OSS / S3 的情况：**

- 部署形态本身就在公有云上，且**没有数据不出域的合规约束**。这时托管服务运维成本为零、弹性近乎无限、还自带 CDN/生命周期/跨区复制等成熟能力，自己跑 MinIO 反而是给自己加运维负担。coze 对此的支持就是 `STORAGE_TYPE=s3`（或火山形态用 `tos`）。
- 数据量极大且增长不可预测、又不想自己买盘扩容——公有云的"按量付费 + 无限扩容"更划算。

**该上 Ceph（或其它重型分布式存储）的情况：**

- 客户**本来就在自建私有云 / IaaS**，已经有 Ceph 集群和专职存储团队。这时让 coze 的对象存储复用现有 Ceph RGW（S3 兼容）比单独再起一套 MinIO 更合理——因为 Ceph 已经被运维起来了，边际成本低。
- 需要在同一套存储里同时提供对象 + 块 + 文件三种能力（例如对象存储只是更大平台的一部分）。

**MinIO 自身规模上的注意（待核实）：**

- MinIO 的纠删码部署有节点/磁盘数量的最佳实践（如单 set 内磁盘数有推荐范围），超大规模（数百 PB）下的运维与扩容策略需要专门规划——**具体阈值与版本相关，标注「待核实」**。但对 coze 主流私有化规模（几 TB ~ 几百 TB 级别），MinIO 完全够用。

**判断口诀**：要私有化/数据不出域 → MinIO（默认）；纯公有云无合规约束 → 用该云的 OSS/S3；已有自建 IaaS 存储团队 → 复用 Ceph。

---

## 6. 对我们项目（vibe-studio）的取舍

vibe-studio 对齐 coze 架构，对象存储**也用 MinIO**。这个决定纯从技术约束推导，不是"照抄":

1. **开发期就要"开发=生产"的存储语义**。如果本地开发用文件系统、生产用对象存储，预签名 URL、tagging、分片上传这些行为在两套后端上不一致，会埋下"本地能跑线上挂"的坑。本地直接起一个 MinIO 容器，开发和生产说同一套 S3 协议，行为一致。

2. **保留切换公有云的能力，零成本**。沿用 coze 的 `storage.Storage` 抽象 + 多实现 + env 切换，意味着 vibe-studio 现在用自托管 MinIO，将来某个部署要落到公有云（AWS / 阿里云 / 火山），改个 `STORAGE_TYPE` + 凭证即可，**业务代码不动**。这是"现在选 MinIO"几乎没有锁定风险的根本原因——因为我们绑的是 **S3 协议**，不是 MinIO 这个具体产品。

3. **将来的资产形态正好命中对象存储**。vibe-studio 规划里要存**沙箱产物**（代码执行/构建产物，写一次读多次、体积不定）和**视频资产**（大文件、要能直接给前端流式拉取/预览）。这两类都是"非结构化 + 大 + 不就地改 + 要 HTTP 直达"的对象存储典型负载：
   - 视频资产用**预签名 URL** 直接发给前端播放器，不经后端转发，省后端带宽；
   - 大文件用**分片上传**（multipart）抗网络抖动；
   - 沙箱产物用 **key 前缀**按任务/用户组织，用 `ListObjectsPaginated` 分页列举，用 **tagging/生命周期** 做自动清理。
   这些能力 MinIO 原生支持，自己用文件系统造一遍既不划算也不可靠。

4. **轻量、好部署**。vibe-studio 当前规模下，MinIO 单二进制/单容器即可，不需要 Ceph 那种重型方案；等真到了需要重型分布式存储或纯公有云的规模，第 5 节的边界判断同样适用，且因为绑的是 S3 协议，迁移代价可控。

**结论**：vibe-studio 用 MinIO 是"S3 协议统一生态（不锁定）+ 开发生产一致 + 对象存储天然匹配未来资产形态 + 部署轻"四条共同推出的，而非路径依赖。

---

## 7. 来源与核实状态（区分事实 / 推断）

### 7.1 确定事实（有源码/配置佐证）

| 结论 | 证据位置 |
|---|---|
| coze 自研 `storage.Storage` 接口抽象对象存储能力 | `backend/infra/storage/storage.go`、`option.go` |
| 接口方法直接对齐 S3 语义（Put/Get/Delete/Head/List/预签名 URL） | `storage.go` 接口定义 |
| 支持 3 种实现：MinIO / AWS S3 / 火山 TOS | `backend/infra/storage/impl/{minio,s3,tos}/` |
| 靠环境变量 `STORAGE_TYPE` 工厂选实现 | `backend/infra/storage/impl/storage.go` 的 `New()` |
| 默认值是 `minio` | `docker/.env.example`、`.env.debug.example`：`STORAGE_TYPE="minio"` |
| 开箱 compose 真的起一个自托管 MinIO 并自动建桶 | `docker/docker-compose.yml` 约 203 行 `minio:` 服务 |
| MinIO 用 `minio-go/v7 v7.0.90` | `go.mod` + `impl/minio/minio.go` |
| S3 用 `aws-sdk-go-v2/service/s3 v1.84.1`，且用自定义 EndpointResolver（=通用 S3 兼容客户端，可连非 AWS 端点） | `go.mod` + `impl/s3/s3.go` |
| TOS 用 `volcengine/ve-tos-golang-sdk/v2 v2.7.17` | `go.mod` + `impl/tos/tos.go` |
| `GetObjectUrl` 返回预签名 URL，默认有效期 7 天 | `impl/minio/minio.go` 的 `GetObjectUrl`（`3600*24*7`） |

### 7.2 合理推断（逻辑/常识，非源码直证）

| 推断 | 依据与不确定度 |
|---|---|
| 默认选 MinIO 的主因是"私有化必须自托管 + S3 兼容统一生态" | 由"coze 是私有化平台 + compose 自带 minio + 接口按 S3 设计"反推，逻辑链清晰但官方未给文字说明 |
| TOS 实现的动机是字节自家生态 dogfooding（火山引擎） | TOS 实现存在是事实；"动机是吃自家狗粮"是合理推断 |
| 不选 Ceph 是因运维过重/场景过度 | 通用工程常识，非 coze 官方表态 |
| 不选 SeaweedFS 是因成熟度/生态不如 MinIO | 通用生态认知，非 coze 官方表态 |

### 7.3 待核实（不引未验证数字）

- MinIO 的具体性能数字（吞吐/延迟）——依赖硬件与对象大小分布，本环境未跑 benchmark，**待核实**。
- SeaweedFS 与 MinIO 的 S3 兼容完整度具体差异清单——**待核实**。
- MinIO 纠删码部署的节点/磁盘数量最佳实践阈值与超大规模运维边界——版本相关，**待核实**。
- 阿里云 OSS / 火山 TOS 的 S3 兼容覆盖度细节——**待核实**（不影响"它们是公有云托管、不能自托管"这一定性结论）。

---

> 全文方法论：先讲清三种存储模型与 S3 协议（是什么）→ 源码举证 coze 怎么用（接口 + 多实现 + 默认 MinIO）→ 三类动机拆解为什么选（技术/生态/平台，重点平台私有化约束）→ 逐个替代对比 + 选错代价 → 适用边界 → 落到本项目取舍 → 区分事实/推断/待核实。结论收敛到一句话：**默认 MinIO = "私有化必须自托管" × "S3 协议统一生态" 的交点，绑的是协议不是产品，所以零锁定。**
