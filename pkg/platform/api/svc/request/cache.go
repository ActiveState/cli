package request

type CacheRequest struct {
	key string
}

func NewCacheRequest(key string) *CacheRequest {
	return &CacheRequest{
		key: key,
	}
}

func (c *CacheRequest) Query() string {
	return `query($key: String!)  {
		getCache(key: $key) 
	}`
}

func (c *CacheRequest) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{
		"key": c.key,
	}, nil
}

type StoreCacheRequest struct {
	key   string
	value string
}

func NewStoreCacheRequest(key, value string) *StoreCacheRequest {
	return &StoreCacheRequest{
		key:   key,
		value: value,
	}
}

func (c *StoreCacheRequest) Query() string {
	return `query($key: String!, $value: String!)  {
		storeCache(key: $key, value: $value) 
	}`
}

func (c *StoreCacheRequest) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{
		"key":   c.key,
		"value": c.value,
	}, nil
}
