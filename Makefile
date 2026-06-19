.PHONY: help up down logs tidy build dev fe-install fe-dev test test-be test-fe gen gen-be gen-fe lint lint-be lint-fe

help:
	@echo "make up         启动本地中间件(mysql/redis/minio)"
	@echo "make down       停止中间件"
	@echo "make logs       查看中间件日志"
	@echo "make tidy       go mod tidy(拉取后端依赖)"
	@echo "make build      go build 后端"
	@echo "make dev        运行后端(需先 make up)"
	@echo "make fe-install 安装前端依赖"
	@echo "make fe-dev     运行前端 dev server"
	@echo "make test       前后端全部测试"
	@echo "make test-be    后端测试(go test)"
	@echo "make test-fe    前端测试(vitest, turbo)"
	@echo "make gen        从 openapi.yaml 重新生成前后端类型/客户端"
	@echo "make lint       前后端 lint (golangci-lint + eslint)"

up:
	docker compose up -d

down:
	docker compose down

logs:
	docker compose logs -f

tidy:
	cd backend && go mod tidy

build:
	cd backend && go build ./...

dev:
	cd backend && go run ./cmd/server

fe-install:
	cd frontend && pnpm install

fe-dev:
	cd frontend && pnpm dev

test: test-be test-fe

test-be:
	cd backend && go test ./...

test-fe:
	cd frontend && pnpm test

# 改了 backend/api/openapi/openapi.yaml 后，重新生成前后端类型/客户端
gen: gen-be gen-fe

gen-be:
	cd backend && $$(go env GOPATH)/bin/oapi-codegen -generate types -package openapi -o api/openapi/openapi.gen.go api/openapi/openapi.yaml

gen-fe:
	cd frontend && pnpm --filter @vibe/api-client gen

lint: lint-be lint-fe

lint-be:
	cd backend && $$(go env GOPATH)/bin/golangci-lint run ./...

lint-fe:
	cd frontend && pnpm lint
