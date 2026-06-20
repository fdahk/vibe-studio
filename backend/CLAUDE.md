# backend/ — Go 规则（先读根 CLAUDE.md，再读本文件）

栈：Go 1.26 · net/http(1.22 ServeMux) · GORM/MySQL 8 · go-redis · MinIO · JWT(golang-jwt v5) · slog · golang-migrate。

## 三层数据模型（别混用，详见 docs/KB/IDL与数据模型分层.md）
- **DTO**（传输）：`api/openapi/openapi.gen.go` —— 由 openapi.yaml 生成，**不手改**
- **Entity**（领域实体）：`domain/*/entity.go` —— 手写，纯领域，无 gorm/json tag
- **PO**（持久化）：`infra/persistence/*` —— 手写，带 gorm tag
- 三者间显式转换，别让一个结构体跨层复用

## 依赖方向（依赖倒置，违反=架构债）
`api → application → domain ← infra`
- `domain` 包**禁止 import** 任何 infra / 框架 / DB 包
- `infra` 实现 `domain` 定义的接口；组合根在 `api/router/router.go` 装配

## HTTP
- 路由用 net/http 1.22 ServeMux：`"POST /api/v1/users"`（方法+路径）
- 中间件经典签名 `func(http.Handler) http.Handler`；全局链：recover/请求ID/CORS/访问日志，Auth 走路由级
- handler 的请求/响应类型来自生成的 `openapi` 包

## 迁移（golang-migrate，backend/migrations/，go:embed 内嵌）
- 成对 `NNNNNN_name.up.sql` / `.down.sql`
- **未上线的 schema 直接改最终版迁移 + 重置 dev 库**，别叠增量 ALTER / 数据搬迁；一文件一条语句（免 multiStatements），down 只留 DROP

## lint 已自动管的，别在代码里操心
golangci-lint v2（standard 集 + misspell）+ gofmt + goimports（local-prefix `vibe-studio/backend`）。格式/imports/未用变量交给工具。
