package lrucache

// wencan
// 2017-08-31

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/wencan/cachex"
)

func TestLRUCache(t *testing.T) {
	cache := NewLRUCache(10, 0)
	assert.Implements(t, (*cachex.Storage)(nil), cache)
}

func TestLRUCacheMaxEntries(t *testing.T) {
	// 最多缓存10个元素，不过期
	cache := NewLRUCache(10, 0)
	// 缓存11个元素，将会丢弃第一个
	for i := 0; i < 11; i++ {
		err := cache.Set(i, i*i)
		if !assert.NoError(t, err) {
			return
		}
	}

	var value int
	err := cache.Get(5, &value)
	assert.NoError(t, err)
	assert.Equal(t, 5*5, value)

	err = cache.Get(0, &value)
	assert.Implements(t, (*cachex.NotFound)(nil), err)
}

func TestLRUCacheExpire(t *testing.T) {
	cache := NewLRUCache(0, time.Millisecond*10)

	key := "test"
	value := "test"
	err := cache.Set(key, value)
	if !assert.NoError(t, err) {
		return
	}

	var cached string
	err = cache.Get(value, &cached)
	assert.NoError(t, err)
	assert.Equal(t, value, cached)

	time.Sleep(time.Millisecond * 20)

	// 支持StaleWhenError
	err = cache.Get(value, &cached)
	assert.Implements(t, (*cachex.Expired)(nil), err)
	assert.Equal(t, value, cached)
}

func TestLRUCacheCustomExpire(t *testing.T) {
	cache := NewLRUCache(0, time.Millisecond*10)

	key := "test"
	value := "test"
	err := cache.SetWithTTL(key, value, time.Millisecond*20)
	if !assert.NoError(t, err) {
		return
	}

	var cached string
	err = cache.Get(value, &cached)
	assert.NoError(t, err)
	assert.Equal(t, value, cached)

	time.Sleep(time.Millisecond * 15)

	// 尚未过期
	err = cache.Get(value, &cached)
	assert.NoError(t, err)
	assert.Equal(t, value, cached)

	time.Sleep(time.Millisecond * 10)

	// 已经过期
	// 支持StaleWhenError
	err = cache.Get(value, &cached)
	assert.Implements(t, (*cachex.Expired)(nil), err)
	assert.Equal(t, value, cached)
}

func TestLRUCacheLength(t *testing.T) {
	cache := NewLRUCache(10, 0)

	for i := 0; i < 10; i++ {
		err := cache.Set(i, i*i)
		if !assert.NoError(t, err) {
			return
		}
		assert.Equal(t, i+1, cache.Len())
	}

	err := cache.Set("test", "test")
	assert.NoError(t, err)
	assert.Equal(t, 10, cache.Len())

	cache.Clear()
	assert.Equal(t, 0, cache.Len())
}

func TestLRUCacheDel(t *testing.T) {
	cache := NewLRUCache(0, time.Second)

	key := "test"
	value := "test"
	err := cache.Set(key, value)
	if !assert.NoError(t, err) {
		return
	}

	var cached string
	err = cache.Get(value, &cached)
	assert.NoError(t, err)
	assert.Equal(t, value, cached)

	err = cache.Del(key)
	assert.NoError(t, err)

	err = cache.Get(value, &cached)
	assert.Equal(t, NotFound{}, err)
}
