package cache

import (
	"github.com/gin-gonic/gin"
	"github.com/jylc/cloudserver/pkg/conf"
)

var Store Driver = NewMemoStore()

func Init() {
	if conf.Rc.Server != "" && gin.Mode() != gin.TestMode {
		Store = NewRedisStore(10, conf.Rc.Network, conf.Rc.Server, conf.Rc.Password, conf.Rc.Db)
	}

}

type Driver interface {
	Set(key string, value interface{}, ttl int) error

	Get(key string) (interface{}, bool)

	Gets(keys []string, prefix string) (map[string]interface{}, []string)

	Sets(values map[string]interface{}, prefix string) error

	Delete(keys []string, prefix string) error
}

func Set(key string, value interface{}, ttl int) error {
	return Store.Set(key, value, ttl)
}

func Get(key string) (interface{}, bool) {
	return Store.Get(key)
}

func Deletes(keys []string, prefix string) error {
	return Store.Delete(keys, prefix)
}

func GetSettings(keys []string, prefix string) (map[string]string, []string) {
	raw, miss := Store.Gets(keys, prefix)
	res := make(map[string]string, len(raw))
	for k, v := range raw {
		res[k] = v.(string)
	}
	return res, miss
}

func SetSettings(values map[string]string, prefix string) error {
	var toBeSet = make(map[string]interface{}, len(values))
	for key, value := range values {
		toBeSet[key] = interface{}(value)
	}
	return Store.Sets(toBeSet, prefix)
}
