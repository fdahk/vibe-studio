package auth

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("secret123")
	require.NoError(t, err)
	assert.NotEqual(t, "secret123", hash, "哈希不应等于明文")
	assert.True(t, VerifyPassword(hash, "secret123"), "正确密码应校验通过")
	assert.False(t, VerifyPassword(hash, "wrong"), "错误密码应校验失败")
}

func TestJWTIssueAndParse(t *testing.T) {
	j := NewJWT("test-secret", time.Hour)
	token, err := j.Issue("user-123")
	require.NoError(t, err)
	require.NotEmpty(t, token)

	uid, err := j.Parse(token)
	require.NoError(t, err)
	assert.Equal(t, "user-123", uid)
}

func TestJWTParseInvalid(t *testing.T) {
	j := NewJWT("test-secret", time.Hour)
	_, err := j.Parse("not-a-jwt")
	assert.Error(t, err)
}

func TestJWTWrongSecretRejected(t *testing.T) {
	token, err := NewJWT("secret-a", time.Hour).Issue("u1")
	require.NoError(t, err)
	_, err = NewJWT("secret-b", time.Hour).Parse(token)
	assert.Error(t, err, "用不同密钥签发的 token 应校验失败")
}
