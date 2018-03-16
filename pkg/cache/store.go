package cache

type (
	// Store is a generic interface for cache stores
	Store interface {
		Get(key string) ([]byte, error)
		Set(key string, contents []byte) error
		Delete(key string) error
	}
)
