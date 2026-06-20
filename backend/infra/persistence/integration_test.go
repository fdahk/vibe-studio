//go:build integration

package persistence

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	domain "vibe-studio/backend/domain/user"
	dbmigrate "vibe-studio/backend/infra/migrate"
	"vibe-studio/backend/pkg/errorx"
)

var testDB *gorm.DB

// TestMain 建一个独立测试库 vibe_studio_test 并跑迁移，集成测试都跑在它上面。
func TestMain(m *testing.M) {
	base := envOr("TEST_MYSQL_BASE", "root:root@tcp(127.0.0.1:3306)")

	root, err := sql.Open("mysql", base+"/?parseTime=true")
	if err != nil {
		panic(err)
	}
	if _, err := root.Exec("CREATE DATABASE IF NOT EXISTS vibe_studio_test CHARACTER SET utf8mb4"); err != nil {
		panic(err)
	}
	_ = root.Close()

	db, err := gorm.Open(mysql.Open(base+"/vibe_studio_test?charset=utf8mb4&parseTime=True&loc=Local"), &gorm.Config{})
	if err != nil {
		panic(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		panic(err)
	}
	if err := dbmigrate.Run(sqlDB); err != nil {
		panic(err)
	}
	testDB = db
	os.Exit(m.Run())
}

func envOr(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func TestRepoIntegration(t *testing.T) {
	testDB.Exec("DELETE FROM identities")
	testDB.Exec("DELETE FROM users")
	repo := NewRepo(testDB)
	ctx := context.Background()

	u := &domain.User{ID: "u1", Username: "alice", Email: "a@x.com", Status: "active"}
	id := &domain.Identity{ID: "i1", UserID: "u1", Provider: "local", ProviderUID: "alice", Secret: "hash"}
	require.NoError(t, repo.CreateAccount(ctx, u, id))

	got, err := repo.GetUserByID(ctx, "u1")
	require.NoError(t, err)
	assert.Equal(t, "alice", got.Username)

	gid, err := repo.GetIdentity(ctx, "local", "alice")
	require.NoError(t, err)
	assert.Equal(t, "u1", gid.UserID)

	_, err = repo.GetIdentity(ctx, "local", "ghost")
	assert.ErrorIs(t, err, errorx.ErrNotFound)
}

func TestSessionRepoIntegration(t *testing.T) {
	testDB.Exec("DELETE FROM sessions")
	repo := NewSessionRepo(testDB)
	ctx := context.Background()
	now := time.Now()

	s := &domain.Session{ID: "s1", UserID: "u1", TokenHash: "h1", ExpiresAt: now.Add(time.Hour), LastUsedAt: now}
	require.NoError(t, repo.Create(ctx, s))

	got, err := repo.FindByTokenHash(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, "u1", got.UserID)

	// 轮换：这正是 Save 把 created_at 写成越界值那个 bug 的复现路径。
	got.PrevTokenHash = got.TokenHash
	got.TokenHash = "h2"
	got.LastUsedAt = time.Now()
	require.NoError(t, repo.Update(ctx, got), "Update 不应因 created_at 越界失败")

	_, err = repo.FindByTokenHash(ctx, "h2")
	require.NoError(t, err)
	prev, err := repo.FindByPrevTokenHash(ctx, "h1")
	require.NoError(t, err)
	assert.Equal(t, "s1", prev.ID)

	// 吊销。
	revoked := time.Now()
	prev.RevokedAt = &revoked
	require.NoError(t, repo.Update(ctx, prev))
	cur, err := repo.FindByTokenHash(ctx, "h2")
	require.NoError(t, err)
	assert.True(t, cur.IsRevoked())

	_, err = repo.FindByTokenHash(ctx, "nope")
	assert.ErrorIs(t, err, errorx.ErrNotFound)
}
