package auth

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateRefreshTokenIsUnique(t *testing.T) {
	a, err := GenerateRefreshToken()
	assert.NoError(t, err)
	assert.NotEmpty(t, a)

	b, err := GenerateRefreshToken()
	assert.NoError(t, err)
	assert.NotEqual(t, a, b, "每次生成应不同（足够随机）")
}

func TestHashTokenIsDeterministicAndNotPlaintext(t *testing.T) {
	h := HashToken("some-refresh-token")
	assert.Equal(t, h, HashToken("some-refresh-token"), "同输入同哈希")
	assert.NotEqual(t, "some-refresh-token", h, "存的应是哈希而非明文")
	assert.NotEqual(t, HashToken("a"), HashToken("b"), "不同输入不同哈希")
}
