package cache

import (
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

type MemoStore struct {
	Store *sync.Map
}

type itemWithTtl struct {
	expires int64
	value   interface{}
}

func newItem(value interface{}, expires int) itemWithTtl {
	expires64 := int64(expires)
	if expires > 0 {
		expires64 = time.Now().Unix() + expires64
	}
	return itemWithTtl{
		expires: expires64,
		value:   value,
	}
}

func getValue(item interface{}, ok bool) (interface{}, bool) {
	if !ok {
		return nil, ok
	}

	var itemObj itemWithTtl
	if itemObj, ok = item.(itemWithTtl); !ok {
		return item, true
	}

	if itemObj.expires > 0 && itemObj.expires < time.Now().Unix() {
		return nil, false
	}

	return itemObj.value, ok
}

func NewMemoStore() *MemoStore {
	return &MemoStore{Store: &sync.Map{}}
}

func (store *MemoStore) Set(key string, value interface{}, ttl int) error {
	store.Store.Store(key, newItem(value, ttl))
	return nil
}

func (store *MemoStore) Get(key string) (interface{}, bool) {
	return getValue(store.Store.Load(key))
}

func (store *MemoStore) Gets(keys []string, prefix string) (map[string]interface{}, []string) {
	var res = make(map[string]interface{})
	var notFound = make([]string, 0, len(keys))

	for _, key := range keys {
		if value, ok := getValue(store.Store.Load(prefix + key)); ok {
			res[key] = value
		} else {
			notFound = append(notFound, key)
		}
	}
	return res, notFound
}

func (store *MemoStore) Sets(values map[string]interface{}, prefix string) error {
	for key, value := range values {
		store.Store.Store(prefix+key, value)
	}
	return nil
}

func (store *MemoStore) Delete(keys []string, prefix string) error {
	for _, key := range keys {
		store.Store.Delete(prefix + key)
	}
	return nil
}

func (store *MemoStore) GarbageCollect() {
	store.Store.Range(func(key, value interface{}) bool {
		if item, ok := value.(itemWithTtl); ok {
			if item.expires > 0 && item.expires < time.Now().Unix() {
				logrus.Debugf("collect garbage[%s]\n", key.(string))
				store.Store.Delete(key)
			}
		}
		return true
	})
}
