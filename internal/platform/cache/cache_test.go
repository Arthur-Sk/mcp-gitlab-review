package cache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New(5*time.Minute, 100)

	c.Set("key1", "value1")

	val, ok := c.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "value1", val)
}

func TestCache_GetMissing(t *testing.T) {
	c := New(5*time.Minute, 100)

	val, ok := c.Get("nonexistent")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestCache_Expiration(t *testing.T) {
	c := New(10*time.Millisecond, 100)

	c.Set("key1", "value1")

	time.Sleep(20 * time.Millisecond)

	val, ok := c.Get("key1")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestCache_MaxSize(t *testing.T) {
	c := New(5*time.Minute, 2)

	c.Set("key1", "value1")
	c.Set("key2", "value2")
	c.Set("key3", "value3")

	_, ok := c.Get("key3")
	assert.True(t, ok)

	count := 0
	for _, key := range []string{"key1", "key2", "key3"} {
		if _, ok := c.Get(key); ok {
			count++
		}
	}

	assert.LessOrEqual(t, count, 2)
}

func TestCache_Overwrite(t *testing.T) {
	c := New(5*time.Minute, 100)

	c.Set("key1", "value1")
	c.Set("key1", "value2")

	val, ok := c.Get("key1")
	require.True(t, ok)
	assert.Equal(t, "value2", val)
}
