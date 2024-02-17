package cache

type Cache interface {
	Store(key string, i interface{})
	Get(key string) interface{}
}

type FileCache interface {
	Store(key string, i interface{}) error
	Get(key string) (interface{}, error)
}
