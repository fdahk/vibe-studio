//go:build integration

package sms

import (
	"context"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"vibe-studio/backend/pkg/errorx"
)

func testRedis() *goredis.Client {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	return goredis.NewClient(&goredis.Options{Addr: addr})
}

func TestRedisCodeStoreIssueAndVerify(t *testing.T) {
	rdb := testRedis()
	ctx := context.Background()
	rdb.FlushDB(ctx)
	store := NewRedisCodeStore(rdb, 5*time.Minute, time.Minute)
	const phone = "13800138000"

	code, err := store.Issue(ctx, phone)
	require.NoError(t, err)
	assert.Len(t, code, 6)

	// 冷却期内再发 → 拒绝（防刷）。
	_, err = store.Issue(ctx, phone)
	assert.ErrorIs(t, err, errorx.ErrSMSTooFrequent)

	// 错码失败、对码通过、用过即失效。
	assert.ErrorIs(t, store.Verify(ctx, phone, "000000"), errorx.ErrSMSCodeInvalid)
	assert.NoError(t, store.Verify(ctx, phone, code))
	assert.ErrorIs(t, store.Verify(ctx, phone, code), errorx.ErrSMSCodeInvalid)
}

func TestRedisCodeStoreAttemptsLimit(t *testing.T) {
	rdb := testRedis()
	ctx := context.Background()
	rdb.FlushDB(ctx)
	store := NewRedisCodeStore(rdb, 5*time.Minute, time.Minute)
	const phone = "13900139000"

	code, err := store.Issue(ctx, phone)
	require.NoError(t, err)
	for i := 0; i < maxVerifyAttempts; i++ {
		assert.Error(t, store.Verify(ctx, phone, "111111"))
	}
	// 错够次数后作废：即便给对的码也失败（防爆破）。
	assert.ErrorIs(t, store.Verify(ctx, phone, code), errorx.ErrSMSCodeInvalid)
}
