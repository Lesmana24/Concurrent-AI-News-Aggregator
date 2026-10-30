package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRedisCache_DisabledWhenHostEmpty(t *testing.T) {
	c := NewRedisCache("", "6379", "")
	assert.NotNil(t, c)
	assert.False(t, c.IsAvailable())

	ctx := context.Background()
	_, err := c.Get(ctx, "test_key")
	assert.Error(t, err)

	err = c.Set(ctx, "test_key", "value", 10*time.Second)
	assert.NoError(t, err)

	err = c.Delete(ctx, "test_key")
	assert.NoError(t, err)

	err = c.Close()
	assert.NoError(t, err)
}

func TestNewRedisCache_UnreachableServerGracefulFallback(t *testing.T) {
	// Attempt to connect to an unreachable port
	c := NewRedisCache("127.0.0.1", "59998", "")
	assert.NotNil(t, c)
	assert.False(t, c.IsAvailable())
}

