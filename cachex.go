package cachex

// wencan
// 2017-08-31

import (
	"errors"
	"reflect"
	"sync"
	"time"
)

var (
	// ErrNotFound 没找到
	ErrNotFound = errors.New("not found")

	// ErrNotSupported 操作不支持
	ErrNotSupported = errors.New("not supported operation")
)

// Cachex 缓存处理类
type Cachex struct {
	storage Storage

	querier Querier

	sentinels sync.Map

	// useStale UseStaleWhenError
	useStale bool

	deletableStorage   DeletableStorage
	withTTLableStorage SetWithTTLableStorage
}

// NewCachex 新建缓存处理对象
func NewCachex(storage Storage, querier Querier) (c *Cachex) {
	c = &Cachex{
		storage: storage,
		querier: querier,
	}
	c.deletableStorage, _ = storage.(DeletableStorage)
	c.withTTLableStorage, _ = storage.(SetWithTTLableStorage)
	return c
}

// Get 获取
func (c *Cachex) Get(key, value interface{}) error {
	if v := reflect.ValueOf(value); v.Kind() != reflect.Ptr || v.IsNil() {
		panic("value not is non-nil pointer")
	}

	err := c.storage.Get(key, value)
	if err == nil {
		return nil
	} else if _, ok := err.(NotFound); ok {
		// 下面查询
	} else if _, ok := err.(Expired); ok {
		// 数据已过期，下面查询
	} else if err != nil {
		return err
	}

	if c.querier == nil {
		return ErrNotFound
	}

	// 在一份示例中
	// 不同时发起重复的查询请求——解决缓存失效风暴
	newSentinel := NewSentinel()
	actual, loaded := c.sentinels.LoadOrStore(key, newSentinel)
	sentinel := actual.(*Sentinel)
	if loaded {
		newSentinel.Destroy()
	}

	// 双重检查
	var staled interface{}
	err = c.storage.Get(key, value)
	if err == nil {
		if !loaded {
			// 将结果通知等待的过程
			sentinel.Done(reflect.ValueOf(value).Elem().Interface(), nil)
			c.sentinels.Delete(key)
		}
		return nil
	} else if err == nil {
		return nil
	} else if _, ok := err.(NotFound); ok {
		// 下面查询
	} else if _, ok := err.(Expired); ok {
		// 保存过期数据，如果下面查询失败，且useStale，返回过期数据
		staled = reflect.ValueOf(value).Elem().Interface()
	} else if err != nil {
		if !loaded {
			// 将错误通知等待的过程
			sentinel.Done(nil, err)
			c.sentinels.Delete(key)
		}
		return err
	}

	if !loaded {
		err := c.querier.Query(key, value)
		if err != nil && c.useStale && staled != nil {
			// 当查询发生错误时，使用过期的缓存数据。该特性需要Storage支持
			reflect.ValueOf(value).Elem().Set(reflect.ValueOf(staled))

			sentinel.Done(staled, err)
			c.sentinels.Delete(key)
			return err
		}

		if _, ok := err.(NotFound); ok {
			err = ErrNotFound
		}
		if err != nil {
			sentinel.Done(nil, err)
			c.sentinels.Delete(key)
			return err
		}

		elem := reflect.ValueOf(value).Elem().Interface()
		err = c.storage.Set(key, elem)

		sentinel.Done(elem, nil)
		c.sentinels.Delete(key)

		return err
	}

	return sentinel.Wait(value)
}

// Set 更新
func (c *Cachex) Set(key, value interface{}) (err error) {
	return c.storage.Set(key, value)
}

// SetWithTTL 设置缓存数据，并定制TTL
func (c *Cachex) SetWithTTL(key, value interface{}, TTL time.Duration) error {
	if c.withTTLableStorage != nil {
		c.withTTLableStorage.SetWithTTL(key, value, TTL)
	}
	return ErrNotSupported
}

// Del 删除
func (c *Cachex) Del(key interface{}) (err error) {
	if c.deletableStorage != nil {
		c.deletableStorage.Del(key)
	}
	return ErrNotSupported
}

// UseStaleWhenError 设置当查询发生错误时，使用过期的缓存数据。该特性需要Storage支持（Get返回并继续暂存过期的缓存数据）。默认关闭。
func (c *Cachex) UseStaleWhenError(use bool) {
	c.useStale = use
}
