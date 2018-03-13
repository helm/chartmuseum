package cache

type (
	// Store is a generic interface for cache stores
	Store interface {
		Get(key string) (interface{}, error)
		Set(key string, contents interface{}) error
		Delete(key string) error
	}
)
