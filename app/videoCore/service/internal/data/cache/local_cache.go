package cache

import (
	"encoding/json"
	"github.com/coocood/freecache"
)

// LocalCache 基于 FreeCache 的本地缓存封装
type LocalCache struct {
	cache *freecache.Cache
}

func NewLocalCache(size int) *LocalCache {
	if size <= 0 {
		size = 100 * 1024 * 1024 // 默认 100MB
	}
	return &LocalCache{
		cache: freecache.NewCache(size),
	}
}

func (c *LocalCache) Set(key []byte, value interface{}, ttl int) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.cache.Set(key, data, ttl)
}

func (c *LocalCache) Get(key []byte, dest interface{}) (bool, error) {
	data, err := c.cache.Get(key)
	if err != nil {
		if err == freecache.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	err = json.Unmarshal(data, dest)
	if err != nil {
		return false, err
	}
	return true, nil
}

func (c *LocalCache) Del(key []byte) bool {
	return c.cache.Del(key)
}
