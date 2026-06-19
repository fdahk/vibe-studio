# coze-studio 官方前端 · 代码级架构分析

> 基于 2026-06 对 `coze-dev/coze-studio` main 分支的**本地实测**（稀疏 clone 了 `frontend/config`、`apps`、`infra`、`packages/{workflow,arch,components}`，逐文件 grep + 阅读 + 两路深读）。
> 文件路径真实；个别版本号/计数为快照近似，标注处为推断。
> 配套：后端分析见 `../backend/coze-studio-代码级架构分析.md`；取舍见 `../../myReact/docs/项目规划/coze/官方架构剖析与我们的工程决策.md`。

---

## 0. 速览

| 指标 | 实测 |
|---|---|
| 形态 | **大型 monorepo**（不是单 app），`rush.json` 注册 **~259 个 project**（按 level-1~4 + team 标签分层） |
| 包管理/编排 | **Rush 5.147.1 + pnpm 8.15.8**（Node ≥21） |
| 构建 | **Rsbuild 1.1.0**（基于 Rspack）+ Semi Design rspack 插件 |
| 框架/栈 | React 18.2 + **react-router v6** + **zustand 4.4.7**（+ immer）+ ahooks + @tanstack/react-query + axios |
| 样式 | **Tailwind 3.3.3** + PostCSS + Less + Stylelint |
| UI 库 | **Semi Design**（`@douyinfe/semi-ui`）→ `@coze-arch/bot-semi` 封装 → `@coze-arch/coze-design` 设计系统 |
| 画布引擎 | **FlowGram 自由布局**（`@flowgram-adapter/free-layout-editor`）——**不是 React Flow** |
| 前后端契约 | **同一套 Thrift IDL**：后端 thriftgo/hz 生成 Go，前端 **idl2ts** 生成 TS 客户端 |
| 可观测 | Slardar(APM) + Tea(埋点) + bot-flags(特性开关) |

**一句话**：一个 **Rush 管理的 259 包 monorepo**，用**字节自研工具链**（Rsbuild/Semi/FlowGram/idl2ts/Slardar/Tea），靠 **IDL 与后端共享契约**，工作流编辑器是 **FlowGram 自由布局 + schema 驱动表单 + 命令模式撤销重做**。

---

## 1. Monorepo 组织

```
frontend/
├── apps/coze-studio/     唯一应用（入口）
├── packages/             业务/领域包（@coze-workflow/* @coze-studio/* @coze-data/* @coze-agent-ide/* ...）
├── config/               ★ 共享工具链配置（被各包 workspace:* 复用）
├── infra/                基础设施包（idl 代码生成器、eslint-plugin 等）
├── rush.json             Rush 编排（~259 projects，level/team 标签）
├── rushx-config.json     rushx 命令配置
└── disallowed_3rd_libraries.json  禁用三方库清单（治理）
```

`packages` 按命名空间分域（实测）：`@coze-arch/*`（架构基础）、`@coze-workflow/*`（工作流编辑器）、`@coze-studio/*`、`@coze-data/*`、`@coze-agent-ide/*`、`@coze-common/*`、`@coze-foundation/*`、`@coze-project-ide/*`、`@coze-devops/*`。

> 这是"几百个包、多团队"规模的组织方式——所以它用 **Rush**（monorepo 编排）而不是 `create vite`（后者是给单 app 的）。

---

## 2. 工具链（`frontend/config/*`，各包 `workspace:*` 复用）

| config 包 | 内容 |
|---|---|
| `rsbuild-config` | **Rsbuild 1.1.0** + Semi Design rspack 插件 + 自定义 `PkgRootWebpackPlugin` |
| `ts-config` | TypeScript 5.8.2 基础配置 |
| `tailwind-config` | Tailwind 3.3.3 + PostCSS nesting |
| `eslint-config` | `@rushstack/eslint-config` + `typescript-eslint 8.x` |
| `stylelint-config` / `vitest-config` / `postcss-config` | 样式检查 / 单测 / PostCSS |

应用级 Rsbuild 配置（`apps/coze-studio/rsbuild.config.ts`）关键点：
```ts
defineConfig({
  server: { proxy: [
    { context: ['/api'], target: 'http://localhost:8888/' },  // 代理到后端 Hertz
    { context: ['/v1'],  target: 'http://localhost:8888/' },
  ]},
  tools: {
    postcss: (_, { addPlugins }) => addPlugins([require('tailwindcss')('./tailwind.config.ts')]),
    rspack(config, { addRules }) {
      addRules([{ test: /\.(css|less|jsx|tsx|ts|js)/, exclude:[/node_modules/],
                  use: '@coze-arch/import-watch-loader' }]);  // HMR 优化
    },
  },
});
// 插件链：React + Less + Sass + SVGR + Semi Design + PkgRoot
```

> 工具链全是字节系：**Rsbuild(Rspack)** 替代 webpack/vite，配套 Semi 的 rspack 插件——和官方后端"全 CloudWeGo 生态"一脉相承（前端是"全 ByteDance 前端生态"）。

---

## 3. 应用骨架（`apps/coze-studio/src`）

```ts
// index.tsx —— 启动顺序
const main = () => {
  initFlags();                                          // 特性开关
  initI18nInstance({ lng: localStorage.getItem('i18next') ?? 'zh-CN' }); // i18n
  dynamicImportMdBoxStyle();
  createRoot(document.getElementById('root')!).render(<App />);
};

// app.tsx —— 路由挂载（react-router v6）
export function App() {
  return <Suspense fallback={<Spin spinning />}><RouterProvider router={router} /></Suspense>;
}

// layout.tsx —— 全局布局
export const Layout = () => { useAppInit(); return <GlobalLayout />; };
```
- 目录：`index.tsx / app.tsx / layout.tsx / routes/ / pages/`，标准 SPA + 配置式路由。
- **状态管理 = zustand**（+ immer 不可变更新）。例：`@coze-studio/bot-studio-store`（level-1 基础 store）导出 `useAuthStore / useSpaceGrayStore` 等；workflow 编辑器内部也大量用 zustand 管局部状态。

---

## 4. ★ IDL → 前端 API 客户端（前后端契约统一，最关键）

**核心洞察**：前端和后端**共用同一套 `idl/*.thrift`**。后端用 thriftgo/hz 生成 Go；前端用自研 **idl2ts** 生成 TS 类型 + 请求函数。改一处 IDL，前后端类型同时更新——这是"契约先行"的全栈闭环。

代码生成工具链（`frontend/infra/idl/`）：
```
idl-parser(解析 thrift)
  → idl2ts-generator(@babel/traverse 遍历 AST 生成 TS)
  → idl2ts-runtime(运行时请求函数库) + idl2ts-helper + idl2ts-plugin(rspack 集成)
```
产物与封装：
```jsonc
// @coze-arch/idl  —— 75+ 个 auto-generated 服务入口
"exports": { "./knowledge": "./src/auto-generated/knowledge/index.ts", ... }

// @coze-arch/api-schema  —— "update": "idl2ts gen ./"（自定义 idl → TS）
// @coze-arch/bot-api     —— bot-http 之上的高层封装，导出生成的 IDL 客户端
```
HTTP 与流式：
```ts
// @coze-arch/bot-http/src/axios.ts —— 统一 axios 实例 + 拦截器
axiosInstance.interceptors.response.use(resp => {
  const { code, msg } = resp.data;          // 业务错误码统一处理
  emitAPIErrorEvent(new APIErrorEvent(...)); // 全局错误事件
}, error => { /* 401/403 重定向 */ });

// @coze-arch/fetch-stream —— SSE/流式（LLM 输出）
// 基于 eventsource-parser + web-streams-polyfill，逐 chunk onMessage，带超时控制
export async function fetchStream<M, D>(req, { onMessage, streamParser, betweenChunkTimeout }) { ... }
```

> 对照后端：后端 `api/model`（thriftgo 生成）+ 前端 `@coze-arch/idl`（idl2ts 生成）= **同源 IDL 的两端产物**。这是 coze 工程化最值得学的一点。

---

## 5. UI 体系（三层）

```
@douyinfe/semi-ui (2.72.x, Semi Design 基础库, 字节开源)
   ↑ 封装
@coze-arch/bot-semi (level-1)  —— 再导出 70+ 组件 + 注入 i18n / bot-icons / ahooks
   ↑ 之上
@coze-arch/coze-design (内部设计系统)  —— 品牌定制 / 主题
```
应用代码主要 import `@coze-arch/coze-design` 与 `bot-semi`，而非直接用 semi-ui——便于统一换肤/治理。

---

## 6. ★ Workflow 可视化编辑器（`packages/workflow/*`，crown jewel）

分包（每个是独立 package）：
```
playground         编辑器主体(host) + 节点注册 + 表单
render             FlowGram 渲染层接入(背景/连线/悬停/快捷键)
nodes              节点基础类型定义
setters            schema 驱动表单的"控件(setter)"库
variable           变量系统(输入/输出/引用/类型推导)
history            撤销重做(命令模式)
fabric-canvas      基于 Fabric.js 的图像编辑能力
feature-encapsulate 子工作流封装
adapter / base / sdk 适配/基础/提取工具
```

### 6.1 画布引擎 = FlowGram 自由布局
```tsx
// playground/src/workflow-playground.tsx
<DndProvider backend={HTML5Backend}>
 <QueryClientProvider client={workflowQueryClient}>
  <WorkflowRenderProvider
    containerModules={[WorkflowNodesContainerModule, WorkflowPageContainerModule, WorkflowHistoryContainerModule]}
    preset={preset}>   {/* ... */}

// render/src/workflow-render-provider.tsx —— 接 FlowGram 核心
<PlaygroundReactProvider
  containerModules={modules}
  plugins={preset}     // createFreeAutoLayoutPlugin / createFreeStackPlugin / createNodeCorePlugin
/>
```
- 画布的节点拖拽/连线/缩放/布局由 `@flowgram-adapter/free-layout-editor` 提供；coze 通过 `WorkflowRenderContribution` 注册自定义渲染层。
- 用 **inversify 依赖注入**（`@injectable`/`@inject`），`VariableEngine` 等核心服务来自 FlowGram。

### 6.2 节点注册（WorkflowNodeRegistry + FormMetaV2）
```ts
// playground/src/nodes-v2/llm/llm-node-registry.ts
export const LLM_NODE_REGISTRY: WorkflowNodeRegistry<NodeTestMeta> = {
  type: StandardNodeType.LLM,
  meta: { size:{width:360,height:130.7}, inputParametersPath:'/$$input_decorator$$/inputParameters',
          getLLMModelIdsByNodeJSON: nodeJSON => /* 提取模型id */ },
  formMeta: LLM_FORM_META,
};

// node-registries/code/form-meta.tsx —— schema 驱动表单
export const CODE_FORM_META: FormMetaV2<FormData> = {
  render: () => <FormRender />,
  validateTrigger: ValidateTrigger.onChange,
  validate: { nodeMeta: nodeMetaValidate, [CODE_PATH]: codeEmptyValidator, ... },
  effect:   { outputs: provideNodeOutputVariablesEffect, ... },   // 副作用(输出变量更新)
  formatOnInit:   transformOnInit,    // 后端 DTO → 表单数据
  formatOnSubmit: transformOnSubmit,  // 表单数据 → 后端 DTO
};
```
- 每种节点一个 registry（`node-registries/index.ts` 导出 50+ 个）：`type` + `meta`(尺寸/取值路径/动态端口) + `formMeta`(渲染/校验/副作用/DTO 双向转换)。

### 6.3 setters（控件）
```ts
// setters/src/types.ts
export type Setter<V=unknown, C=unknown> = React.FC<SetterProps<V, C>>;
```
setter = 一个受控表单控件（array/enum/input/expression-editor/system-prompt…）；`formMeta` 把节点配置 schema 映射成 setter 组件树，支持嵌套校验与副作用。

### 6.4 变量系统
```ts
// variable/src/core/workflow-variable-facade-service.ts
@injectable() export class WorkflowVariableFacadeService {
  @inject(VariableEngine) variableEngine;   // 来自 FlowGram
  // 节点输出变量重命名 → 自动更新所有引用表达式
  fieldRenameService.onRename(({before,after}) => traverseUpdateRefExpressionByRename(...));
}
```
管理节点输入/输出/引用变量（`WorkflowNode{Input,Output,Ref}VariablesData`）+ 跨节点引用 + 重命名联动 + 类型推导。

### 6.5 撤销重做（命令模式）
```ts
// history/src/operation-metas/add-node.ts
export const addNodeOperationMeta: OperationMeta<...> = {
  type: FreeOperationType.addNode,
  inverse: op => ({ ...op, type: FreeOperationType.deleteNode }), // 反操作
  apply:  (op, ctx) => ctx.get(WorkflowDocument).createWorkflowNode(...),
  shouldMerge,  // 连续操作是否合并
};
```
每种操作定义 `apply` + `inverse` + `shouldMerge`，注册到 FlowGram 的 `OperationRegistry`——标准命令模式撤销重做。

### 6.6 与后端协议（Canvas Schema）
- 编辑器内部 = FlowGram Document；保存时 `workflow-save-service.ts` 把 Document → `WorkflowJSON`，再经各节点 `formatOnSubmit` 转后端 DTO，序列化为 **Canvas Schema**（对应后端 `vo.Canvas`：nodes[{id,type,data{nodeMeta,inputs,outputs},meta{position}}] + lines/edges）。
- 即：前端"画"出来的 JSON，正是后端 `CanvasToWorkflowSchema` 的输入（见后端分析 §7.1）——**前后端在 Canvas Schema 这层对齐**。

---

## 7. 可观测与工程能力（`packages/arch/*`）

| 能力 | 包 | 职责 |
|---|---|---|
| APM | `slardar-adapter` / `slardar-interface` | 性能监控 + 错误上报（Slardar，字节 APM；适配器解耦） |
| 埋点 | `tea` / `tea-adapter` | 事件埋点（Tea，字节分析） |
| 特性开关 | `bot-flags` | 功能开关（EventEmitter3，动态下发） |
| i18n | `i18n` | i18next + ICU 多语言 |
| 日志 | `logger` | 结构化日志 + React Error Boundary |
| 代码编辑 | `bot-monaco-editor` | Monaco 0.45 封装（Code 节点/表达式编辑） |

---

## 8. 前后端如何拼成一个系统

```
同一套 idl/*.thrift
   ├─(thriftgo/hz)→ 后端 Go: api/model + router + handler
   └─(idl2ts)→ 前端 TS: @coze-arch/idl + bot-api 客户端
前端 FlowGram 画布 → 保存为 Canvas Schema(JSON) ──HTTP(bot-http/axios)──▶ 后端 workflow 域
   后端 CanvasToWorkflowSchema → Eino Graph → 执行
   执行流式结果 ──SSE(fetch-stream ◀ infra/sse)──▶ 前端 test-run 实时展示
```
**两端在两个地方对齐**：① IDL（请求/响应类型）；② Canvas Schema（工作流图）。这就是"对话/编排 → 执行 → 流式回显"的全栈协作骨架。

---

## 9. 对我们项目（vibe-studio）的启发

1. **画布选型分叉**：官方用 **FlowGram 自由布局**（开箱即用、就是为 Coze 这种自由布局做的）；我们规划用 **React Flow**（生态/教程大、底层可控但要自己写布局）。**取舍**：要"最对齐官方 + 自由布局少造轮子" → FlowGram；要"生态大、面试官熟、底层全懂" → React Flow。我们当前定 React Flow 仍成立，但**值得认真评估 FlowGram**（它正是 coze 同款）。
2. **状态管理 zustand**：与我们规划一致，✅。
3. **IDL 统一契约**是最值得偷师的一点：一套 thrift 同时喂前后端。我们右size 暂没上 IDL，但**这是将来对齐官方、消除前后端类型漂移的方向**（后端 §11 也提了）。
4. **schema 驱动表单(setters) + 节点 registry + 命令模式撤销重做 + 变量引用联动**：正是我们技术点07 前端编排器要做的"协议层/变量系统/撤销重做"——这套 FlowGram 上的实现是高质量参考。
5. **我们单 app 不该上 Rush**：官方 259 包才用 Rush；我们用 `pnpm create vite` 单 app（将来分出多 app/共享包再上 pnpm workspace + turbo）。

---

## 附：信息来源与核实状态

- **实测来源**：本地稀疏 clone `coze-dev/coze-studio`@main（2026-06），`frontend/{config,apps,infra,packages/workflow,packages/arch,packages/components}`。
- **代码片段**：摘自源码，为可读性裁剪；路径准确，**行号/版本号为快照近似**。
- **待核实/存疑**：`rush.json` 项目数以本地 `grep -c '"packageName"'` 得 **~259**（早期估计 400+，并行探查曾报 1338，以 grep 为准）；pnpm 版本 8.15.8 与个别探查报 8.8.15 略有出入；`coze-design` 与 `bot-semi` 的精确分工为推断。
- **配套**：后端见 `../backend/coze-studio-代码级架构分析.md`；整体取舍见 `项目规划/coze/官方架构剖析与我们的工程决策.md`。
