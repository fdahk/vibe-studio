# Vibe Studio — 项目协作约定

> 全局行为约束见 `~/.claude/CLAUDE.md`（已生效，本文件不重复）。
> 技术选型/架构的「为什么」见 `README.md` 与 `docs/技术选型/`、`docs/KB/` —— 它们是权威。
> 本文件只列 agent 容易违反的硬约束 + 流程 + 完工门禁 + 命令。

## 这是什么
自研 AI 全栈开发平台（对标扣子，目标是自实现深核）。
Go(net/http, DDD 模块化单体) + React 18 monorepo(pnpm + Turborepo)。
选型主线：优先经典 / 可迁移 / 标准化 / 零 vendor 锁定。

## 开发流程（用已装的 superpowers skill，别裸跑默认行为）
- 新功能 / 改行为：先 `brainstorming` 对齐需求 → `writing-plans` 出计划 → `test-driven-development` 落地
- 出 bug / 测试失败：走 `systematic-debugging`，别瞎试；默认动作是改实现不是改测试
- 收尾：`requesting-code-review` 自检 → `verification-before-completion`（跑命令验证后才能说"做完"）

## 完工门禁（声称"完成/修好/通过"前必须跑，看到绿才算数）
- 后端改动：`make build` && `make test-be` && `make lint-be`
- 前端改动：`make test-fe` && `make lint-fe`
- 改了 `openapi.yaml`：必须 `make gen` 重新生成两端类型，再各自验证

## 跨层硬约束（违反 ≈ 架构债 / bug）
- ★ **契约先行**：`backend/api/openapi/openapi.yaml` 是前后端唯一契约源。改接口 → 改 yaml → `make gen`。
  **绝不手改生成物**：`backend/api/openapi/openapi.gen.go`、前端 `@vibe/api-client` 生成内容。
- **DDD 依赖方向**：`api → application → domain ← infra`。domain 不依赖任何框架/DB。
- 不引入 README 里标「⏳待定」的依赖（MQ、validator、Monaco、Sentry、ahooks…），除非确有场景并先确认。
- UI 组件 / 画布是刻意自研的深核，别引第三方组件库替代。

## 命令速查（详见 Makefile）
```
make up / down        本地中间件(mysql/redis/minio)
make dev / fe-dev     跑后端 / 前端
make test / lint      前后端全量（test-be/test-fe、lint-be/lint-fe 单跑）
make gen              openapi.yaml → 生成两端类型/客户端
```
