# frontend/ — React 规则（先读根 CLAUDE.md，再读本文件）

栈：React 18 · TypeScript · Vite · Tailwind v4 · Zustand · TanStack Query · openapi-fetch · react-router v6 · Vitest + Testing Library。pnpm + Turborepo monorepo（apps/ + packages/）。

## 状态管理分工（别混）
- **客户端态**（token、UI 开关等）：Zustand
- **服务端态**（请求 / 缓存 / 失效 / 重取）：TanStack Query
- **HTTP 客户端**：openapi-fetch，类型来自 openapi.yaml 生成的 `@vibe/api-client`
  → 别手写 fetch/axios，别手改生成的客户端；接口变更走 `make gen`

## 刻意自研，别引第三方库
UI 组件、画布编辑器（拖拽/连线/渲染/OT 协同）是本项目深核，**必须自研**以吃透原理。
别引 Semi / antd / MUI / FlowGram / react-flow 等替代。

## 样式分工
- Tailwind v4（Vite 插件，CSS-first 配置）：布局 / 工具类
- SCSS Module：自研组件的复杂样式

## 渲染三态
列表/集合渲染必须覆盖 loading / empty / error，empty 不能长得像 error。

## 交互反馈（硬约束）
所有可点击元素（按钮/链接/tab/可点卡片）必须有清晰反馈：`cursor-pointer`（disabled 用 `not-allowed`）+ 肉眼可见的 hover/active/focus 变化（深色面板优先改背景 `hover:bg-surface-2` 而非仅描边）+ `transition`。**禁止悬浮无变化的死按钮。** 注意 Tailwind v4 按钮默认 `cursor:default`，已在 `index.css` 全局给 pointer。

## lint 已自动管的，别操心
ESLint flat config(v9) + react-hooks + typescript-eslint + Prettier。格式/hooks 规则交给工具。
