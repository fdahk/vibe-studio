// Package migrations 内嵌版本化 SQL 迁移文件（golang-migrate 用）。
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
