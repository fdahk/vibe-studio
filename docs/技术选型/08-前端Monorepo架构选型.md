# 前端 Monorepo 架构选型（web + 原生 iOS/Android 三端）

> 目标：三端（**web=React/TS、iOS=原生 Swift、Android=原生 Kotlin**）+ 后端 Go，如何组织代码、如何共享。
> 核心结论：**"monorepo"在这里要分两层看**——① 仓库级 polyglot monorepo（一个 git repo 装所有端，各自工具链）；② JS 级 workspace（`frontend/` 内部用 pnpm workspaces + Turborepo 管多个 TS 包）。**跨语言共享不是靠代码 import，而是靠 OpenAPI 契约当中枢**（一份 `openapi.yaml` → 各语言各自生成客户端）。

---

## 1. 先厘清一个关键误区：原生端进不了 JS monorepo

很容易以为"monorepo = 把 web/iOS/Android 三端代码放进一个 pnpm 工作区一起管"。**这对原生端不成立**：

- pnpm/npm/yarn workspaces、Turborepo、Rush 这些都是 **JS/TS 生态的工具**，只认 `package.json`，只能管 TS/JS 包。
- iOS（Swift，Xcode/SPM）、Android（Kotlin，Gradle）是**完全不同的语言和构建系统**，没有 `package.json`，无法被 pnpm 工作区纳管，也无法 `import` 一个 TS 包。

**所以原生端"不需要额外处理"——准确说不是"特殊处理"，而是它们压根不参与 JS 工作区**：各用各的工具链（iOS=Xcode/SPM、Android=Gradle）管自己的依赖与构建，与 `frontend/` 完全隔离。三端唯一的连接点是 **OpenAPI 契约**（见第 3 节）——原生端从同一份 `openapi.yaml` 生成自己语言的客户端，仅此而已。所以三端的"共享"必须换一种方式实现。

---

## 2. 两层 monorepo

### 第一层：仓库级 polyglot（多语言）monorepo

一个 git 仓库 `vibe-studio/` 同时容纳所有端，每个端用各自的工具链：

```
vibe-studio/                 ← 一个 git 仓库（polyglot monorepo）
├── backend/                 Go（go.mod）
├── frontend/                JS/TS 工作区（pnpm workspaces + Turborepo）
├── ios/                     原生 Swift（Xcode 工程 / SPM）        ← 待建
├── android/                 原生 Kotlin（Gradle 工程）            ← 待建
└── docs/
```

"monorepo"在这一层的意义 = **一个仓库、统一管理、共享同一份契约**，而不是"所有代码用同一个构建工具"。

### 第二层：JS workspace（仅 frontend 内部）

`frontend/` 内部才是真正意义上的 JS workspace，用 pnpm workspaces 管多个 TS 包（web app + 共享包）。

---

## 3. 跨语言怎么共享？——OpenAPI 契约当中枢

三端不能互相 import 代码，但它们**调用同一个后端**。所以共享点是**接口契约**：

```
backend/api/openapi/openapi.yaml   ← 唯一契约源（single source of truth）
   ├─ openapi-typescript ───→ web(TS) 的类型 + 客户端
   ├─ openapi-generator (swift5) ─→ iOS(Swift) 的 model + 客户端
   ├─ openapi-generator (kotlin) ─→ Android(Kotlin) 的 model + 客户端
   └─ oapi-codegen ─────────→ 后端(Go) 的类型
```

**这才是"三端统一"的真正落地**：不是把代码塞进一个工作区，而是**一份 OpenAPI 契约，各语言各自生成各自的客户端**。改一处 `openapi.yaml`，四端（Go/TS/Swift/Kotlin）重新生成即同步。

> 对照本仓库已有结论：见 `[07-OpenAPI契约与net-http框架决策.md](07-OpenAPI契约与net-http框架决策.md)` 与 `[../KB/IDL与数据模型分层.md](../KB/IDL与数据模型分层.md)`。OpenAPI 作为跨语言中枢，正是选它（而非 Thrift）的核心收益之一。

---

## 4. pnpm workspaces 与 Turborepo：各是什么、为什么两个都要

这是最容易混的点。先记住一句话：**pnpm workspaces 管「依赖怎么装、包之间怎么互相找到」，Turborepo 管「任务怎么高效地跑」。两者在不同层、解决不同问题，是叠加关系，不是二选一。**

### 4.1 pnpm workspaces —— 包管理层（「装」）

pnpm 是个包管理器（与 npm / yarn 同类）；workspaces 是它的一个能力：允许**一个仓库里有多个 `package.json`（多个"包"）**，pnpm 把它们当成一个整体来安装依赖。

它解决两件事：

- **依赖只装一份**：所有包的依赖装进同一个内容寻址 store，再用硬链接连到各包。重复依赖去重 → 省磁盘、装得快；还顺带防"幽灵依赖"（用了却没在自己 `package.json` 声明的包）。
- **本地包直接互引**：web app 在 `package.json` 里写 `"@vibe/api-client": "workspace:*"`，就能像引 npm 包一样引本地源码——改了 `@vibe/api-client` 立刻在 web 生效，**不用发布到 npm 再装回来**。

它**不管**"怎么跑 build / test、按什么顺序跑、要不要缓存"——那不是包管理器的职责。

### 4.2 Turborepo —— 任务编排层（「跑」）

Turborepo 是个**任务运行器（task runner）**。你的每个包里都有 `build` / `test` / `lint` 等脚本，turbo 帮你**一条命令、按依赖顺序、并行、带缓存**地把它们跑完。

它解决四件事：

- **一条命令跑全仓**：`turbo run build` 跑所有包的 build，不用手动 `cd` 进每个包逐个跑。
- **按依赖图排顺序**：`@vibe/web` 依赖 `@vibe/shared`，turbo 自动先 build `shared` 再 build `web`。
- **缓存（核心价值）**：某个包的输入没变就直接命中缓存秒过，只重跑受影响的包；CI 上尤其省时间。
- **并行**：互不依赖的包同时跑。

它**不管**"依赖怎么装、版本怎么去重"——那是包管理器的事。

### 4.3 为什么两个都要（职责正交，叠加使用）

| 缺谁 | 会怎样 |
|---|---|
| 缺 pnpm workspaces | 多个包各装各的依赖，重复臃肿；本地包互引只能靠 `npm link` / 发版，繁琐易错 |
| 缺 Turborepo | 依赖能装好，但跑构建要手动逐个包跑，没顺序保证、没缓存、不能并行；包一多就难受 |

打个比方：**pnpm workspaces 像"统一仓储"**——把所有零件入库、贴标签、彼此能找到；**Turborepo 像"流水线调度"**——决定先装哪个零件、哪些能并行、做过的不重做。仓储 ≠ 调度，所以两个都要。这也是 2026 年 JS monorepo 的主流默认组合。

> 也因此回答了第 1 节的问题：这两个工具都只认 JS 的 `package.json`，原生 iOS/Android 没有 `package.json`，自然两个都用不上，也就"无需额外处理"。

---

## 5. JS 工具横向对比

monorepo 工具分两层职责：**包管理/依赖**（workspaces）与**任务编排/缓存**（task runner）。


| 工具                    | 职责层            | 定位                                  | 适用规模    | 备注                       |
| --------------------- | -------------- | ----------------------------------- | ------- | ------------------------ |
| **pnpm workspaces**   | 包管理            | 硬链接 + 内容寻址 store，去重快、防 phantom deps | 任意      | 我们选的包管理层                 |
| npm / yarn workspaces | 包管理            | 同类，但 pnpm 更省空间、依赖隔离更严               | 任意      | pnpm 更优                  |
| **Turborepo**         | 任务编排           | 任务管线 + 远程/本地缓存 + 只跑受影响的包，零侵入        | 中小~中大   | 我们选的编排层，配置简单             |
| Nx                    | 任务编排+插件生态      | 功能强大（代码生成器/依赖图/插件），但概念多、侵入性强        | 中大~大    | 对我们偏重                    |
| Rush                  | 一体化（Microsoft） | 企业级严格依赖策略 + 发布流程，重                  | 超大（几百包） | **coze 用它**（259 包），对我们过重 |
| Lerna                 | 发布工具（老牌）       | 早期主流，现多被 pnpm+Turborepo/Nx 取代       | —       | 不推荐新项目                   |


---

## 6. 选型：pnpm workspaces + Turborepo

理由：

- **轻、主流、可迁移**：2026 年 JS monorepo 的主流默认组合；概念少、配置简单（`pnpm-workspace.yaml` + `turbo.json`）。
- **够 3 端用**：我们 JS 侧就 web + 几个共享包，不需要 Rush 那套企业级依赖策略。
- **比 Rush 轻、比 Nx 简单**：Rush 是 coze 的 259 包规模才需要；Nx 概念/插件偏重。
- pnpm 防 phantom dependency（幽灵依赖）、装包快、省磁盘。

> coze 用 Rush 是因为它有 **259 个前端包 + 多团队**；我们的规模用 pnpm+Turborepo 才是合适的"右size"。

---

## 7. 目录结构

仓库级：

```
vibe-studio/
├── backend/        Go
├── frontend/       ← JS workspace（本文重点）
├── ios/            原生 Swift（待建，吃 openapi.yaml 生成的 Swift 客户端）
├── android/        原生 Kotlin（待建，吃 openapi.yaml 生成的 Kotlin 客户端）
└── docs/
```

`frontend/` JS workspace：

```
frontend/
├── pnpm-workspace.yaml        packages: ['apps/*', 'packages/*']
├── turbo.json                 build/dev/lint/test/typecheck 管线
├── package.json               根（private，turbo 脚本 + 共享 devDeps）
├── apps/
│   └── web/                   React + Vite 应用
└── packages/
    ├── api-client/            从 openapi.yaml 生成的 TS 类型 + typed 客户端（@vibe/api-client）
    └── shared/                跨包共享的类型/工具（@vibe/shared）
```

---

## 8. 原生端如何生成客户端（待建 iOS/Android 用）

原生端不进 JS workspace，但从同一份契约生成客户端（示意命



令，落地时按工具版本核实）：

```bash
# iOS（Swift）
openapi-generator generate -i backend/api/openapi/openapi.yaml -g swift5 -o ios/Generated/APIClient
# Android（Kotlin）
openapi-generator generate -i backend/api/openapi/openapi.yaml -g kotlin  -o android/app/src/generated
```

生成物（Swift/Kotlin 的 model + API 调用类）作为各自工程的一部分。**契约变更 → 各端重新生成**，保持四端类型一致。

---

## 9. 落地命令 / 工作流（JS 侧）

```bash
cd frontend
pnpm install                 # 安装所有 workspace 包依赖
pnpm turbo run build         # 构建所有包（带缓存，只跑受影响的）
pnpm turbo run dev           # 起 dev（web）
pnpm turbo run test          # 跑所有包测试
pnpm --filter @vibe/web dev  # 只起 web
# 改了后端契约后，重新生成各端客户端：
pnpm --filter @vibe/api-client gen   # web 端 TS（openapi-typescript）
```

---

## 10. 踩坑 / 注意

- **phantom dependency（幽灵依赖）**：用了没在自己 package.json 声明的包 → pnpm 默认严格隔离会暴露这类问题（好事），但迁移时可能报"找不到模块"，按提示补声明。
- **依赖提升/版本一致**：多包用不同版本的 React 会出诡异问题；共享依赖（react/typescript）版本要统一（可在根或共享配置约束）。
- **循环依赖**：`@vibe/web → @vibe/api-client → @vibe/shared` 单向，别让 shared 反向依赖 app。
- **契约同步**：四端客户端都从 openapi.yaml 生成，但生成是"按需手动/CI 触发"的——契约改了要记得重新生成各端（建议 CI 加校验：生成物与 spec 不一致则失败）。
- **原生与 TS 不共享运行时代码**：别幻想 Swift/Kotlin 复用 TS 逻辑；跨语言只共享"契约"，不共享"实现"。业务逻辑各端各写（或下沉到后端）。

---

## 11. 信息来源与核实状态

- **已核实**：coze 前端为 Rush monorepo（~259 包，见本目录 `06-Rspack-Rsbuild-前端构建选型.md` 与官方架构剖析）；pnpm workspaces / Turborepo / Nx / Rush 的定位为业界通识。
- **待核实（落地时确认）**：`openapi-generator` 的 swift5 / kotlin 生成器确切参数与产物结构；Turborepo 当前版本配置 schema。
- **结论**：JS 侧用 **pnpm workspaces + Turborepo**；原生 iOS/Android 在仓库内独立工程、**经 OpenAPI 契约共享**而非纳入 JS 工作区。

