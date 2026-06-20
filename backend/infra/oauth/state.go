package oauth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// StateStore 存 OAuth state：CSRF 防护 + 单次使用（用 Redis 的 GETDEL 原子消费）。
type StateStore struct {
	rdb *goredis.Client
	ttl time.Duration
}

func NewStateStore(rdb *goredis.Client, ttl time.Duration) *StateStore {
	return &StateStore{rdb: rdb, ttl: ttl}
}

// Issue 生成并存储一个随机 state。
func (s *StateStore) Issue(ctx context.Context) (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	state := hex.EncodeToString(b)
	if err := s.rdb.Set(ctx, key(state), "1", s.ttl).Err(); err != nil {
		return "", err
	}
	return state, nil
}

// Consume 校验并消费 state（取出即删，单次有效）。
func (s *StateStore) Consume(ctx context.Context, state string) bool {
	if state == "" {
		return false
	}
	v, err := s.rdb.GetDel(ctx, key(state)).Result()
	return err == nil && v == "1"
}

func key(state string) string { return "oauth:state:" + state }
