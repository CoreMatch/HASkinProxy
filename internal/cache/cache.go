package cache

import (
	"encoding/json"
	"haskinproxy/config"

	"github.com/coocood/freecache"
)

type Cache struct {
	Store *freecache.Cache
}

func NewCache() *Cache {
	size := config.AppConfig.Cache.MaxSizeMB * 1024 * 1024
	if size <= 0 {
		size = 256 * 1024 * 1024
	}
	return &Cache{
		Store: freecache.NewCache(size),
	}
}

func (c *Cache) Get(key string, target interface{}) (bool, error) {
	val, err := c.Store.Get([]byte(key))
	if err != nil {
		if err == freecache.ErrNotFound {
			return false, nil
		}
		return false, err
	}
	err = json.Unmarshal(val, target)
	return true, err
}

func (c *Cache) Set(key string, value interface{}, ttl int) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return c.Store.Set([]byte(key), data, ttl)
}

func (c *Cache) GetRaw(key string) ([]byte, bool, error) {
	val, err := c.Store.Get([]byte(key))
	if err != nil {
		if err == freecache.ErrNotFound {
			return nil, false, nil
		}
		return nil, false, err
	}
	return val, true, nil
}

func (c *Cache) SetRaw(key string, val []byte, ttl int) error {
	return c.Store.Set([]byte(key), val, ttl)
}
