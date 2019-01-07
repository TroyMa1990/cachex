/*
 * redis存储支持
 *
 * wencan
 * 2018-12-30
 */

package rdscache

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gomodule/redigo/redis"
	"github.com/vmihailenco/msgpack"
)

var (
	// Marshal 数据序列化函数
	Marshal = msgpack.Marshal

	// Unmarshal 数据反序列化函数
	Unmarshal = msgpack.Unmarshal
)

// NotFound 没找到错误
type NotFound struct{}

// NotFound 实现cachex.NotFound错误接口
func (NotFound) NotFound() {}
func (NotFound) Error() string {
	return "not found"
}

var notFound = NotFound{}

// RdsCache redis存储实现
type RdsCache struct {
	rdsPool *redis.Pool

	keyPrefix string

	ttlMilliseconds int
}

// PoolConfig redis连接池配置
type PoolConfig struct {
	MaxIdle int

	MaxActive int

	IdleTimeout time.Duration

	Wait bool

	MaxConnLifetime time.Duration
}

// RdsConfig rdscache配置
type RdsConfig struct {
	opt *redis.DialOption

	poolCfg *PoolConfig

	keyPrefix string

	ttl time.Duration
}

// RdsDial redis连接函数
func RdsDial(dial func(network, addr string) (net.Conn, error)) RdsConfig {
	opt := redis.DialNetDial(dial)
	return RdsConfig{
		opt: &opt,
	}
}

// RdsDB redis db配置
func RdsDB(db int) RdsConfig {
	opt := redis.DialDatabase(db)
	return RdsConfig{
		opt: &opt,
	}
}

// RdsPassword redis密码
func RdsPassword(password string) RdsConfig {
	opt := redis.DialPassword(password)
	return RdsConfig{
		opt: &opt,
	}
}

// RdsPoolConfig redis连接池配置对象
func RdsPoolConfig(poolCfg PoolConfig) RdsConfig {
	return RdsConfig{
		poolCfg: &poolCfg,
	}
}

// RdsTTL redis key生存时间
func RdsTTL(ttl time.Duration) RdsConfig {
	return RdsConfig{
		ttl: ttl,
	}
}

// RdsKeyPrefix redis key前缀
func RdsKeyPrefix(keyPrefix string) RdsConfig {
	return RdsConfig{
		keyPrefix: keyPrefix,
	}
}

// NewRdsCache 创建redis缓存对象
func NewRdsCache(network, address string, rdsCfgs ...RdsConfig) *RdsCache {
	var opts []redis.DialOption
	var poolCfg *PoolConfig
	var keyPrefix string
	var ttl time.Duration

	for _, c := range rdsCfgs {
		if c.opt != nil {
			opts = append(opts, *c.opt)
		}
		if c.poolCfg != nil {
			poolCfg = c.poolCfg
		}
		if c.keyPrefix != "" {
			keyPrefix = c.keyPrefix
		}
		if c.ttl != 0 {
			ttl = c.ttl
		}
	}

	rdsPool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial(network, address, opts...)
		},
	}
	if poolCfg != nil {
		rdsPool.MaxIdle = poolCfg.MaxIdle
		rdsPool.MaxActive = poolCfg.MaxActive
		rdsPool.IdleTimeout = poolCfg.IdleTimeout
		rdsPool.Wait = poolCfg.Wait
		rdsPool.MaxConnLifetime = poolCfg.MaxConnLifetime
	}

	return &RdsCache{
		rdsPool:         rdsPool,
		keyPrefix:       keyPrefix,
		ttlMilliseconds: int(ttl / time.Millisecond),
	}
}

// stringKey 将interface{} key转为字符串并加上前缀，不支持类型返回错误
func (c *RdsCache) stringKey(key interface{}) (string, error) {
	var skey string
	switch t := key.(type) {
	case fmt.Stringer:
		skey = t.String()
	case string, []byte, int, int32, int64, uint, uint32, uint64, float32, float64, bool:
		skey = fmt.Sprint(key)
	default:
		return "", errors.New("key type is unacceptable")
	}

	if c.keyPrefix != "" {
		skey = strings.Join([]string{c.keyPrefix, skey}, ":")
	}
	return skey, nil
}

// Set 设置缓存数据
func (c *RdsCache) Set(key, value interface{}) error {
	skey, err := c.stringKey(key)
	if err != nil {
		return err
	}

	data, err := Marshal(value)
	if err != nil {
		return err
	}

	conn := c.rdsPool.Get()
	defer conn.Close()

	_, err = conn.Do("SET", skey, data)
	if err != nil {
		return err
	}
	if c.ttlMilliseconds != 0 {
		_, err = conn.Do("PEXPIRE", skey, c.ttlMilliseconds)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get 获取缓存数据
func (c *RdsCache) Get(key, value interface{}) error {
	skey, err := c.stringKey(key)
	if err != nil {
		return err
	}

	conn := c.rdsPool.Get()
	data, err := redis.Bytes(conn.Do("GET", skey))
	conn.Close()
	if err == redis.ErrNil {
		return notFound
	} else if err != nil {
		return err
	}

	err = Unmarshal(data, value)
	if err != nil {
		return err
	}

	return nil
}

// Del 删除缓存数据
func (c *RdsCache) Del(key interface{}) error {
	skey, err := c.stringKey(key)
	if err != nil {
		return err
	}

	conn := c.rdsPool.Get()
	defer conn.Close()

	_, err = conn.Do("DEL", skey)
	if err != nil {
		return err
	}

	return nil
}
