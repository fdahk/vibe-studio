//go:build integration

package oauth

import (
	"context"
	"os"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateStoreIntegration(t *testing.T) {
	addr := os.Getenv("TEST_REDIS_ADDR")
	if addr == "" {
		addr = "127.0.0.1:6379"
	}
	rdb := goredis.NewClient(&goredis.Options{Addr: addr})
	ctx := context.Background()
	store := NewStateStore(rdb, time.Minute)

	st, err := store.Issue(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, st)

	assert.True(t, store.Consume(ctx, st), "首次消费成功")
	assert.False(t, store.Consume(ctx, st), "单次有效，再消费失败")
	assert.False(t, store.Consume(ctx, "nonexistent"))
}
