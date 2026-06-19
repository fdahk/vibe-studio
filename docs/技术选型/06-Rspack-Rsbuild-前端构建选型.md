# Rspack / Rsbuild —— 前端构建工具选型

> 本文是一篇"技术选型理由"文档，目标读者是想搞懂"巨型 monorepo 为什么选 Rspack/Rsbuild、为什么不选别的"这件事本质的人。
> 全文区分三类动机：**纯技术动机**（性能/兼容）、**生态/dogfooding 动机**（Rspack/Rsbuild 是字节 web infra 团队自研，内部大规模验证后开源）、**规模动机**（coze-studio 前端是 ~259 包的 Rush monorepo，构建性能是硬约束）。
> 凡是"确定事实"都给出 coze-studio 源码/版本佐证；凡是"合理推断的动机"明确标注；构建速度的具体数字标"待核实"。

---

## 1. 是什么（打包器 / 构建工具到底干嘛）

### 1.1 一个前端构建工具要解决的三件事

你写的源码（TS/JSX/Less/SVG/CSS Module……）浏览器一个都不认。浏览器只认 JS、CSS、能解析的资源。构建工具干的就是把"开发期的源码形态"翻译成"浏览器能跑的产物形态"，具体拆成三层：

- **转译（transform）**：单文件级别的语法降级。TS 去类型、JSX 变 `React.createElement`、ES2022 私有方法降到目标浏览器支持的语法、Less/Sass 编译成 CSS。负责这层的工具历史上是 Babel/ts-loader，现在是 esbuild/SWC，Rspack 内置的是 **SWC**（Rust 写的转译器）。
- **打包（bundle）**：模块级别的图构建。从入口出发解析 `import` 依赖，构建一张**模块依赖图**，做 **tree-shaking**（删没用到的导出）、**code splitting**（按路由/体积拆 chunk）、**作用域提升**，最后把成百上千个模块拼成有限个 bundle 文件。这是 webpack/Rspack/Rollup 的核心活儿。
- **开发服务（dev server）**：开发期不需要产出最终产物，需要的是**快**。起一个本地服务器，做模块热替换（**HMR**，改一个文件只更新那个模块、不刷整页），按需编译、增量编译。

### 1.2 bundler vs dev server：两个常被混为一谈的概念

这是理解 Rspack 和 Vite 之争的关键，必须先掰清楚。

- **bundler（打包器）**：webpack、Rspack、Rollup、esbuild、Parcel。职责是"把模块图打成产物"。它既能服务生产构建，也能驱动开发服务——但开发期它仍然在做"打包"这件事（只是增量化了）。
- **dev server（开发服务器）**：Vite 是这条路线的代表。Vite 的 dev 模式**根本不打包**：它利用浏览器原生 ESM（`<script type="module">`），把你的源码文件几乎一对一地通过 HTTP 暴露给浏览器，浏览器请求哪个模块、它就**按需转译哪个模块**（用 esbuild 做单文件转译，极快）。第三方依赖才用 esbuild 预打包成一个 bundle（pre-bundling）。所以 Vite dev 启动飞快——它把"打包"这个最贵的步骤在开发期**整个跳过了**。

> 一句话区分：**bundler 路线（webpack/Rspack）开发期也在打包，靠 Rust/增量把打包做快；no-bundle 路线（Vite dev）开发期不打包，靠浏览器原生 ESM 绕开打包。** 两条路线在 dev 体验上都能很快，但快的原理完全不同，代价也不同（见第 4 节 Vite 部分）。

### 1.3 Rspack 与 Rsbuild 各是什么

- **Rspack**：字节 web infra 团队用 **Rust** 重写的 **webpack 兼容打包器**。两个卖点叠加——① **API/生态对齐 webpack**：配置结构（`module.rules`、`resolve`、`plugins`、`optimization.splitChunks`……）和 webpack 几乎一致，大量 webpack loader/plugin 可直接复用；② **Rust 实现**：把 webpack 用 JS 跑的那些 CPU 密集型工作（解析、转译、打包、tree-shaking）换成 Rust 多线程，冷启动、全量构建、HMR 都大幅提速。它是"**底层引擎**"。
- **Rsbuild**：架在 Rspack 之上的**开箱即用上层框架**。Rspack 本身像 webpack 一样"什么都能配，但什么都得自己配"；Rsbuild 把 React/TS/Less/Sass/SVGR、PostCSS、产物优化、dev server、代理等做成**预设 + 插件**，给你一套合理默认值，再用 `defineConfig` 暴露可覆盖的口子。类比：**Rspack 之于 Rsbuild ≈ webpack 之于 Create React App / Vue CLI**——一个是引擎，一个是开箱即用的脚手架层。

---

## 2. 在 coze-studio 里怎么用（读源码举证）

以下均以本地源码 `/tmp/coze-studio` 为准。前端是 **Rush monorepo**，`rush.json` 里 `"packageName"` 出现 **259** 次（即 259 个工程包），`rushVersion: 5.147.1`、`pnpmVersion: 8.15.8`。

### 2.1 版本（确定事实）

app `frontend/apps/coze-studio/package.json` 的 `devDependencies`：

- `@rsbuild/core: ~1.1.0`
- `@rspack/core: >=0.7`
- `@rsdoctor/rspack-plugin: 1.0.0-rc.0`（Rsdoctor 是配套的构建分析工具）
- 同时还留着 `webpack: ~5.91.0`、`@coze-arch/pkg-root-webpack-plugin`（下面 2.3 会讲为什么 webpack 还在）

共享配置包 `frontend/config/rsbuild-config/package.json` 把 Rsbuild 插件全家桶钉死在 `~1.1.0` / `~1.0.6`：`@rsbuild/plugin-react`、`@rsbuild/plugin-less`、`@rsbuild/plugin-sass`、`@rsbuild/plugin-svgr`，外加字节自家的 `@douyinfe/semi-rspack-plugin: 2.61.0`（Semi 设计系统的 Rspack 插件）。

### 2.2 两层结构：共享 config 包 + app 级 rsbuild.config.ts

coze 没有让每个 app 各写一份完整 Rsbuild 配置，而是抽了一个 **`@coze-arch/rsbuild-config`** 工程包统一约定，再让 app 在它基础上 merge 自己的差异。这正是 monorepo 的标准做法——**配置即代码、统一收口**。

**共享层** `frontend/config/rsbuild-config/src/index.ts` 导出一个 `defineConfig(options)`，里面预置了整条插件链与默认产物策略（节选）：

```ts
plugins: [
  pluginReact(),
  pluginSvgr({ mixedImport: true, svgrOptions: { exportType: 'named' } }),
  pluginLess({ lessLoaderOptions: { additionalData: `@import "…variables.less";` } }),
  pluginSass({ sassLoaderOptions: { /* silenceDeprecations… */ } }),
],
output: {
  filenameHash: true,
  assetPrefix: cdnPrefix,      // 支持 CDN 前缀注入
  injectStyles: true,
  cssModules: { auto: true },  // 自动识别 *.module.css
  sourceMap: { js: 'source-map' },
  overrideBrowserslist,        // chrome>=51 / safari>=10 / ios_saf>=10 …
},
source: {
  define: getDefine(),         // 把 GLOBAL_ENVS 注入成编译期常量
  include: [/\/node_modules\/(marked|@dagrejs|@tanstack)\//], // 强制编译这几个含 ES2022 语法的依赖
},
tools: {
  postcss: (opts, { addPlugins }) => addPlugins([require('tailwindcss/nesting')(require('postcss-nesting'))]),
  rspack: (_, { appendPlugins }) => appendPlugins([
    new PkgRootWebpackPlugin(),                       // ← 复用的 webpack 生态插件
    new SemiRspackPlugin({ theme: '@coze-arch/semi-theme-hand01' }),
  ]),
},
```

注意 `defineConfig` 最后是 `mergeRsbuildConfig(config, options)`——共享层给基线，app 层传差异，标准的"基线 + 覆盖"两层模型。

**app 层** `frontend/apps/coze-studio/rsbuild.config.ts` 引入 `import { defineConfig } from '@coze-arch/rsbuild-config'`，只写本 app 特有的东西：

- **dev server 代理**：`/api`、`/v1` 代理到 `http://localhost:${WEB_SERVER_PORT||8888}/`，`strictPort: true`。这与 vibe-studio 的 Vite 代理是同一类需求（见第 6 节）。
- **HTML**：标题"扣子 Studio"、favicon、模板。
- **自定义 loader**：通过 `tools.rspack` 的 `addRules` 给 `.css/.less/.tsx/.ts/.js` 挂一个 `@coze-arch/import-watch-loader`（排除 `node_modules` 和 i18n 包）。
- **resolve.fallback**：把 Node 的 `path` 模块 polyfill 成 `path-browserify`（浏览器没有 Node 内置模块，这是 webpack 时代就有的经典做法，Rspack 直接照搬这套字段）。
- **装饰器**：`source.decorators.version: 'legacy'`——因为代码里用了 `inversify` 的 `@injectable()`/`@inject` 依赖注入装饰器。
- **chunk 拆分策略**：`performance.chunkSplit` 用 `split-by-size`，`minSize: 3MB`、`maxSize: 6MB`。**这是 259 包巨型应用的典型烦恼**——模块太多，必须按体积切 chunk，否则单个 bundle 会爆炸。

### 2.3 一个关键证据：webpack 生态插件被原样复用进 Rspack

上面 `appendPlugins([new PkgRootWebpackPlugin(), …])` 里的 `PkgRootWebpackPlugin` 来自 `frontend/infra/plugins/pkg-root-webpack-plugin`。看它的 `package.json`：它包了一个上游 npm 包 `@coze-arch/pkg-root-webpack-plugin@1.0.0-alpha…`，**`devDependencies` 里赫然写着 `webpack: ~5.89.0`**——这就是一个为 webpack 写的插件。它能不改一行代码塞进 Rspack 的 `appendPlugins`，正是 Rspack "webpack 兼容" 卖点的**实锤证据**：旧的 webpack plugin 资产可以直接迁移过来跑。

全仓 `frontend/packages` 下还有 **13** 个包的依赖里列着 `webpack`——说明这套迁移不是一刀切重写，而是"换引擎、留生态"。这正是选 Rspack 而非 Vite 的核心理由之一（见第 3、4 节）。

---

## 3. ★ 为什么官方选 Rspack/Rsbuild（三类理由）

按"事实 / 推断"分级，逐条拆。

### 3.1 纯技术动机：webpack 兼容 + Rust 提速（事实层面可佐证）

**(a) webpack 兼容 = 迁移成本低、生态可复用。**
这是 Rspack 区别于其他"快打包器"的根本设计取向。Rspack 不是发明一套新配置哲学，而是**有意对齐 webpack 的 API 表面**：`module.rules`、`resolve.fallback`、`optimization.splitChunks`、`plugins` 的生命周期钩子……都按 webpack 的样子来。带来的直接好处：

- 团队过去积累的 webpack loader/plugin、对 `splitChunks`/`resolve` 的调参经验**不作废**。2.3 节那个 `PkgRootWebpackPlugin` 就是活的例子。
- 一个有 webpack 历史包袱的大仓库，**迁移到 Rspack 的成本远低于迁移到 Vite**——后者是 Rollup 插件体系，配置哲学和生态都得重学重写。

对于"内部已有海量 webpack 配置/插件"的字节来说，这条几乎是决定性的：换引擎提速，但**不推翻已有工程资产**。

**(b) Rust 实现 = 冷启动 / 构建 / HMR 大幅提速。**
webpack 的瓶颈是它本身用 JS 写、单线程为主，解析+转译+打包都在 V8 里跑。Rspack 把这些 CPU 密集环节换成 **Rust 多线程**，转译用内置 **SWC**（也是 Rust）。对于模块数巨大的应用，构建耗时近似随模块数线性增长，引擎从 JS 换到 Rust 多线程的收益在大仓库上被放大。

> 提速倍数（官方常宣传"数倍于 webpack")属于**待核实**——具体倍数高度依赖项目规模、缓存命中、机器核数，本文不引用未经本仓实测的数字。可佐证的是**方向**：259 包规模下，"JS 单线程打包器"是真实痛点，换 Rust 引擎是对症下药。

### 3.2 生态 / dogfooding 动机：自研 + 内部大规模验证（部分事实 + 部分推断）

- **事实**：Rspack 与 Rsbuild 由**字节 web infra 团队自研开源**；coze 用的 Semi 插件 `@douyinfe/semi-rspack-plugin`（`@douyinfe` 是字节抖音前端的 npm scope）、Semi 设计系统、Rsdoctor 分析工具，全是同一生态的拼图。配置包里这套"Rspack + Rsbuild 插件 + Semi 主题 + Rsdoctor"是**同源工具链**，集成顺滑度天然高。
- **推断**：选自研栈通常还有"**掌控力**"考量——构建工具是工程流水线的命脉，用自家维护的栈，遇到 bug 能直接推动修复、能定制内部需求（私有化部署、CDN 前缀、内部 monorepo 工具链对接），不受制于外部社区节奏。这条对 coze 是合理推断，源码不能直接证明，标**推断**。

### 3.3 规模动机：259 包巨型 monorepo 的构建性能是硬约束（事实层面成立）

这是最该被强调、也最容易被忽略的一条。**前端构建工具的选型，规模是第一性的。**

- coze 前端是 **Rush** 管理的 259 包 monorepo（`rush.json` 实测）。Rush 是微软为"超大规模 monorepo"设计的工具，本身就预设了"包多到一定程度"的场景。
- 在这个规模下，构建工具的常数因子被**直接乘上几百倍**：每次全量构建、每次 dev 冷启动、每次改动触发的增量编译，慢一点都是以分钟计的体感差异。2.2 节那个 `chunkSplit: split-by-size, minSize 3MB / maxSize 6MB` 的配置，就是被规模逼出来的——模块多到必须主动按体积切 chunk。
- 这种规模下，**dev 冷启动和全量构建的绝对耗时**才是主要矛盾。Rspack 的 Rust 引擎正是冲着这个矛盾去的。

> 小结三类理由的权重（推断排序）：**规模动机 ≈ 技术动机（webpack 兼容 + Rust）> dogfooding**。规模和技术是"不得不"，dogfooding 是"正好顺手又可控"。对一个 259 包、带 webpack 历史、要私有化交付的前端，Rspack 几乎是同类工具里**唯一同时满足"兼容旧生态 + 显著提速"**的选项。

---

## 4. ★ 同类替代逐个对比 + 为什么不选

对比维度统一用：**实现语言/引擎**、**dev 启动模型**、**生产打包能力**、**webpack 生态兼容**、**大 monorepo 适配**、**定制能力**。

### 4.1 Webpack —— 生态最大，但 JS 引擎在大仓库上慢

- **引擎**：JS（V8），核心单线程为主。**生态**：全前端最大，loader/plugin 无所不包，调参经验最厚。
- **为什么不选**：纯粹是**性能**。259 包规模下，JS 单线程打包器的冷启动/全量构建慢到影响日常迭代。Rspack 的全部价值就是"**保留 webpack 的生态与 API，换掉它的 JS 引擎**"——所以选 Rspack 本质上是"选了一个更快的 webpack"，webpack 的生态优势 Rspack 几乎全继承了，慢的缺点被去掉了。**技术代价**：几乎为零，这是 webpack → Rspack 迁移成本最低的根本原因。

### 4.2 Vite —— 中小项目最佳 DX，但双引擎 + 巨型 monorepo 是软肋

这是最需要展开的对比，因为它正是 vibe-studio 的选择（第 6 节）。

- **引擎/模型**：**dev 用 esbuild（no-bundle，浏览器原生 ESM 按需编译）**，**prod 用 Rollup 打包**。这是 Vite 的标志，也是它的两难。
- **dev 体验**：中小项目里 Vite dev 冷启动几乎瞬时（因为不打包），DX 业界顶级。
- **为什么 coze 不选**，三层技术代价：
  1. **dev / prod 双引擎不一致**：开发期走 esbuild + 原生 ESM，生产期走 Rollup 打包。两套代码路径意味着"dev 跑得好 ≠ prod 没问题"——产物差异、tree-shaking 行为差异、某些只在打包后才暴露的问题，会在生产构建才现形。项目越大、依赖越杂，这种"dev/prod 行为漂移"的排查成本越高。Rspack 是**单引擎**（dev 和 prod 都是 Rspack 打包），不存在这个裂缝。
  2. **超大 monorepo 下 no-bundle 的反噬**：Vite dev "不打包、按需请求模块"在中小项目是优势，但在 259 包、模块成千上万的应用里会退化——首屏要请求海量模块，浏览器侧的请求瀑布、依赖预打包（pre-bundling）的失效与重建、深层 monorepo 依赖图的解析，都会让 dev 体验从"瞬时"掉下来。no-bundle 的红利随规模递减。
  3. **重度 webpack 生态依赖的迁移成本**：coze 有现成的 webpack loader/plugin 资产（2.3 的 `PkgRootWebpackPlugin`、13 个还挂着 webpack 的包）。迁到 Vite 要全部用 Rollup 插件体系**重写**；迁到 Rspack 基本能复用。对存量大仓，这是巨大的成本差。
- **额外（推断）**：字节有自研 Rspack/Rsbuild，要掌控自有构建栈，自然不会把旗舰产品的构建押在外部社区的 Vite 上。标**推断**。

> 一句话：**Vite 赢在"中小项目 dev DX"，coze 的痛点是"巨型 monorepo 的生产构建 + 旧生态迁移"，两者矛盾点不重合，所以不选。** 这不是 Vite 不好，是场景不匹配（见第 5 节边界）。

### 4.3 esbuild —— 极快，但是"零件"不是"整机"

- **引擎**：Go 写的，转译/打包都极快。
- **为什么不选**：esbuild 的定位是**底层超快转译器/简单打包器**，不是全功能 bundler。它的 **code splitting、CSS 处理、HMR、插件能力都偏弱**，复杂应用需要的产物精细控制（chunk 策略、按需加载、复杂 CSS 流水线、丰富插件钩子）它给不全。**事实上 Rspack 自己内部就用 SWC（同类角色）做转译，Vite 用 esbuild 做 dev 转译**——esbuild/SWC 是"被别人当零件用"的层级，而不是直接拿来当应用主构建。直接用 esbuild 当主力，等于要自己补齐一个全功能 bundler 缺的所有东西。

### 4.4 Turbopack —— 当时绑 Next.js、不成熟

- **引擎**：Rust（Vercel 出品，技术路线和 Rspack 类似）。
- **为什么不选**：① **生态绑定 Next.js**——Turbopack 早期深度耦合 Next 框架，独立用于任意 React monorepo 不现实；coze 不是 Next 应用。② **当时成熟度不足**——长期处于 beta、能力覆盖和稳定性不足以支撑 259 包的生产级 monorepo。对一个要私有化交付的旗舰产品，构建工具的稳定性是底线，不会押在未成熟的工具上。

### 4.5 Rollup —— 库打包之王，不适合大型应用工程

- **引擎**：JS。产物**最干净**（tree-shaking 强、输出可读），是**打库（library）**的事实标准，Vite 的 prod 引擎就是它。
- **为什么不选**：Rollup 面向"打一个干净的库产物"，对**应用级**工程需要的 dev server、HMR、海量资源/CSS 处理、复杂 code splitting，要么靠一堆插件硬拼、要么力不从心。它和 esbuild 一样更像"被集成的引擎"（被 Vite 集成），不适合直接当大型应用的主构建。

### 4.6 Parcel —— 零配好用，大型定制能力弱

- **引擎**：早期 JS、后用 Rust（SWC）做了部分提速。卖点是**零配置**，开箱即用体验好。
- **为什么不选**：零配的另一面是**深度定制能力弱、生态/社区体量小**。259 包、要 CDN 前缀注入、要自定义 loader、要 inversify 装饰器、要按体积切 chunk、要复用 webpack 插件——这种重度定制场景，Parcel 的"约定优先、少给口子"会处处掣肘。Parcel 适合"快速起一个不需要太多定制的项目"，不适合工程复杂度拉满的巨型 monorepo。

### 4.7 一句话对比表

| 工具 | 引擎 | dev 模型 | webpack 生态兼容 | 巨型 monorepo | 不选的核心原因 |
|---|---|---|---|---|---|
| **Rspack/Rsbuild** | Rust | 打包(增量) | **强(API 对齐)** | **强** | —（选它） |
| Webpack | JS | 打包(增量) | 原生 | 弱(慢) | JS 引擎大仓库慢 |
| Vite | esbuild/Rollup 双引擎 | no-bundle | 弱(Rollup 体系) | 中→弱 | 双引擎不一致 + 超大仓 no-bundle 反噬 + 旧生态迁移贵 |
| esbuild | Go | — | 弱 | 中 | 非全功能、HMR/插件弱(被当零件用) |
| Turbopack | Rust | 打包 | 弱 | 中 | 绑 Next、当时不成熟 |
| Rollup | JS | — | 弱 | 弱 | 面向打库、非应用工程 |
| Parcel | JS/Rust | 打包 | 中 | 弱 | 零配但大型定制/生态弱 |

---

## 5. 适用边界：什么规模 Vite 就够，什么时候才需要 Rspack

选型最忌讳"大厂用什么我用什么"。这一节给出**按规模划线**的判断框架。

**用 Vite 就够（甚至更好）的场景：**
- 单 app 或少量包的中小项目，模块数在"几百到一两千"量级；
- 没有沉重的 webpack 历史包袱（不需要复用一堆 webpack loader/plugin）；
- 团队最看重的是**dev 启动速度和 DX**，能接受 dev/prod 双引擎的理论裂缝（绝大多数中小项目根本碰不到这个裂缝）。
- 这类项目里 Vite 的 no-bundle dev 是纯收益，Rspack 的 Rust 引擎优势体现不出来（项目还没大到让 JS 打包慢成瓶颈）。

**才需要上 Rspack/Rsbuild 的场景：**
- **模块数/包数巨大**（几十到几百个工程包的 monorepo，模块上万），JS 单线程打包已成日常迭代瓶颈；
- **有 webpack 存量资产要平滑迁移**（loader/plugin、复杂 `splitChunks` 调参），迁 Vite 等于重写；
- **要求 dev/prod 单引擎一致性**，无法接受双引擎行为漂移的排查成本；
- **重度定制**（CDN 前缀、自定义 loader、装饰器、按体积切 chunk……）需要 webpack 式的全功能配置能力。

**判断的第一性问题永远是**：你的瓶颈到底是"dev 启动慢"还是"巨型 monorepo 的构建慢 + 旧生态迁移"？前者选 Vite，后者选 Rspack。**规模不到，提前上 Rspack 就是过度工程**——它的全部红利都来自"大到 JS 引擎扛不住"这个前提。

---

## 6. 对我们项目（vibe-studio）的取舍

**结论先行：vibe-studio 单 app 用 Vite 是更合适的选择，不应照搬 coze 的 Rspack。理由是规模与场景完全不在一个量级。**

### 6.1 我们的现状（源码为准）

`/Users/mac/vibe-studio/frontend` 的实际配置极简：

- `package.json`：`dev: "vite"`，`build: "tsc --noEmit && vite build"`，依赖只有 `react/react-dom` + `@vitejs/plugin-react` + `vite ^5.4.10` + `typescript`。**纯单 app，无 monorepo**。
- `vite.config.ts` 全文就 13 行：一个 `react()` 插件、dev server 端口 5173、`/api` 代理到后端 Hertz `:8888`。

### 6.2 为什么不照搬 Rspack（逐条对应第 3 节的理由）

把 coze 选 Rspack 的三类理由拿到 vibe-studio 上**逐条失效**：

1. **规模动机失效**：coze 是 259 包的 Rush monorepo，vibe-studio 是**单 app**。Rspack 的 Rust 引擎红利全部来自"巨型 monorepo 的构建/冷启动绝对耗时"这个前提——这个前提我们**根本不存在**。单 app 下 Vite dev 的 no-bundle 冷启动几乎瞬时，体验只会比 Rspack 更好。
2. **webpack 兼容动机失效**：Rspack 兼容 webpack 的最大价值是"复用旧 loader/plugin、迁移成本低"。vibe-studio 是**全新项目，零 webpack 历史包袱**，没有任何旧生态要迁。这条价值对我们等于零。
3. **dogfooding / 掌控力动机失效**：我们不是字节、没有自研构建栈要掌控，反而应该用社区最成熟、文档最全、生态最大的方案——这恰恰是 Vite。

而 coze 不选 Vite 的两个核心顾虑（双引擎不一致、超大仓 no-bundle 反噬），在 vibe-studio 这个规模下**都不会触发**：单 app 模块少，no-bundle 是纯收益；dev/prod 双引擎的理论裂缝在小项目里基本碰不到。

### 6.3 同一类需求、两种实现：dev 代理

值得注意的是，coze 和 vibe-studio 有一个**完全同构的需求**——前端 dev server 把 `/api` 代理到后端，避免开发期 CORS。两边都用同名后端端口 `8888`（Hertz）：

- coze（Rsbuild）：`server.proxy: [{ context: ['/api'], target: 'http://localhost:8888/', changeOrigin: true }, …]`
- vibe-studio（Vite）：`server.proxy: { '/api': 'http://localhost:8888' }`

这说明**构建工具不同不代表工程需求不同**——dev 代理这类需求是通用的，Vite 用更简洁的写法就解决了。我们没必要为了一个三行能搞定的代理，背上 Rsbuild 整套配置体系的复杂度。

### 6.4 什么时候 vibe-studio 才该重新考虑 Rspack

按第 5 节的边界：当且仅当 vibe-studio **演进成多包 monorepo、模块数膨胀到 JS 打包成为日常瓶颈**，或**积累了需要复用的复杂 webpack 式构建逻辑**时，才值得重新评估 Rspack/Rsbuild。在那之前，**照搬大厂的 Rspack 就是典型的过度工程**——用一套为 259 包设计的重型工具链，去解决一个单 app 根本不存在的问题。

> 一句话：**coze 选 Rspack 是被 259 包的规模和 webpack 历史逼出来的正确选择；vibe-studio 选 Vite 是被单 app、零包袱、追求 DX 的现实决定的正确选择。同一个问题域，不同的规模，得出相反的最优解——这恰恰是"按第一性原理而非按品牌"做选型的范例。**

---

## 7. 来源与核实状态

**确定事实（有 coze-studio 源码/版本佐证）：**

- coze 前端是 Rush monorepo，**259 个工程包**（`/tmp/coze-studio/rush.json` 中 `"packageName"` 计数 = 259；`rushVersion 5.147.1` / `pnpmVersion 8.15.8`）。
- 用 Rspack + Rsbuild：`@rsbuild/core ~1.1.0`、`@rspack/core >=0.7`、`@rsdoctor/rspack-plugin 1.0.0-rc.0`（`frontend/apps/coze-studio/package.json`）。
- 共享 config 包 `@coze-arch/rsbuild-config` 预置 React/SVGR/Less/Sass 插件链 + Semi 主题插件 + PostCSS nesting（`frontend/config/rsbuild-config/src/index.ts`）。
- app 层插件链/配置：dev 代理 `/api`+`/v1`→`:8888`、`import-watch-loader`、`path-browserify` fallback、legacy 装饰器、`chunkSplit split-by-size 3MB/6MB`（`frontend/apps/coze-studio/rsbuild.config.ts`）。
- **webpack 生态插件被原样复用进 Rspack**：`PkgRootWebpackPlugin` 来自 `frontend/infra/plugins/pkg-root-webpack-plugin`，其依赖里包了上游 webpack 插件且 `devDependencies` 含 `webpack ~5.89.0`；全仓 `frontend/packages` 下仍有 **13** 个包列 `webpack` 依赖。
- vibe-studio 是**单 app + Vite ^5.4.10**，`vite.config.ts` 仅 `react()` 插件 + 5173 端口 + `/api`→`:8888` 代理（`/Users/mac/vibe-studio/frontend/`）。
- Rspack 内置 SWC 做转译、Vite dev 用 esbuild / prod 用 Rollup、Rspack API 对齐 webpack——均为这些工具的公开设计事实。

**合理推断（源码不能直接证明，已在正文标注）：**

- dogfooding / 自有构建栈"掌控力"是 coze 选 Rspack 的动机之一（推断，3.2 / 4.2）。
- 三类动机的权重排序"规模 ≈ 技术 > dogfooding"（推断，3.3 末）。
- 字节"要掌控自有构建栈、不押注外部 Vite"（推断，4.2）。

**待核实（不引用未实测数字）：**

- Rspack 相对 webpack 的**具体提速倍数**（官方常宣传"数倍"，但高度依赖项目规模/缓存/核数，本仓未实测）——标**待核实**，正文只用"方向正确"而不引用具体倍数。
- Turbopack 早期"绑 Next.js + 不成熟"的具体程度，随版本快速变化，本文以"当时"为限定词，时效性需按当前版本重新核实。
