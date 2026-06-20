package sms

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"math/big"
	"time"

	goredis "github.com/redis/go-redis/v9"

	"vibe-studio/backend/pkg/errorx"
)

// maxVerifyAttempts 单个验证码最多可尝试校验的次数，超限即作废（防爆破）。
const maxVerifyAttempts = 5

// RedisCodeStore 用 Redis 存短信验证码：TTL 过期、发送冷却、校验次数限制（安全 + 高可用）。
type RedisCodeStore struct {
	rdb      *goredis.Client
	ttl      time.Duration
	cooldown time.Duration
}

func NewRedisCodeStore(rdb *goredis.Client, ttl, cooldown time.Duration) *RedisCodeStore {
	return &RedisCodeStore{rdb: rdb, ttl: ttl, cooldown: cooldown}
}

func keyCode(phone string) string     { return "sms:code:" + phone }
func keyCooldown(phone string) string { return "sms:cd:" + phone }
func keyAttempts(phone string) string { return "sms:try:" + phone }

// Issue 生成并存储验证码；冷却期内拒发（防刷）。返回明文供下发。
func (s *RedisCodeStore) Issue(ctx context.Context, phone string) (string, error) {
	if s.rdb == nil {
		return "", errorx.ErrInternal
	}
	// 冷却闸：SET NX，命中说明仍在冷却期。
	ok, err := s.rdb.SetNX(ctx, keyCooldown(phone), "1", s.cooldown).Result()
	if err != nil {
		return "", errorx.ErrInternal.Wrap(err)
	}
	if !ok {
		return "", errorx.ErrSMSTooFrequent
	}
	code, err := genCode()
	if err != nil {
		return "", errorx.ErrInternal.Wrap(err)
	}
	pipe := s.rdb.TxPipeline()
	pipe.Set(ctx, keyCode(phone), code, s.ttl)
	pipe.Del(ctx, keyAttempts(phone))
	if _, err := pipe.Exec(ctx); err != nil {
		return "", errorx.ErrInternal.Wrap(err)
	}
	return code, nil
}

// Verify 校验验证码：错误计数超限即作废（防爆破），常量时间比较（防时序侧信道）。
func (s *RedisCodeStore) Verify(ctx context.Context, phone, code string) error {
	if s.rdb == nil {
		return errorx.ErrInternal
	}
	want, err := s.rdb.Get(ctx, keyCode(phone)).Result()
	if errors.Is(err, goredis.Nil) {
		return errorx.ErrSMSCodeInvalid
	}
	if err != nil {
		return errorx.ErrInternal.Wrap(err)
	}
	attempts, err := s.rdb.Incr(ctx, keyAttempts(phone)).Result()
	if err != nil {
		return errorx.ErrInternal.Wrap(err)
	}
	if attempts == 1 {
		s.rdb.Expire(ctx, keyAttempts(phone), s.ttl)
	}
	if attempts > maxVerifyAttempts {
		s.rdb.Del(ctx, keyCode(phone))
		return errorx.ErrSMSCodeInvalid
	}
	if subtle.ConstantTimeCompare([]byte(want), []byte(code)) != 1 {
		return errorx.ErrSMSCodeInvalid
	}
	s.rdb.Del(ctx, keyCode(phone), keyAttempts(phone))
	return nil
}

// genCode 用 crypto/rand 生成 6 位数字验证码（不可预测）。
func genCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}
