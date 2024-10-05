package cache

type FileCache interface {
	Store(key string, i interface{}) error
	Get(key string, target interface{}) error
	All() (FileCacheAllResult, error)
	Purge() error
}

type FileCacheAllResult interface {
	Get(key string, receiver interface{}) error
	Length() int
	Keys() []string
}
