package request

import "time"

type GetCache struct {
	key string
}

func NewGetCache(key string) *GetCache {
	return &GetCache{key: key}
}

func (c *GetCache) Query() string {
	return `query($key: String!) {
		getCache(key: $key) 
	}`
}

func (c *GetCache) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{
		"key": c.key,
	}, nil
}

type SetCache struct {
	key    string
	value  string
	expiry time.Duration
}

func NewSetCache(key, value string, expiry time.Duration) *SetCache {
	return &SetCache{key: key, value: value, expiry: expiry}
}

func (c *SetCache) Query() string {
	return `mutation($key: String!, $value: String!, $expiry: Int!) {
		setCache(key: $key, value: $value, expiry: $expiry)
	}`
}

func (c *SetCache) Vars() (map[string]interface{}, error) {
	return map[string]interface{}{
		"key":    c.key,
		"value":  c.value,
		"expiry": c.expiry.Seconds(),
	}, nil
}
