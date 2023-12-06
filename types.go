package fifo

// Cache is the interface for a cache.
type Cache[K comparable, V any] interface {
	// Set sets the value for the given key on cache.
	Set(key K, value V) error

	// Get gets the value for the given key from cache.
	Get(key K) (value V, err error)

	// Contains check if a key exists in cache without updating the recent-ness
	Contains(key K) (ok bool, err error)

	// Peek returns key's value without updating the recent-ness.
	Peek(key K) (value V, ok bool, err error)

	// Len returns the number of entries in the cache.
	Len() int

	// Clean clears all cache entries
	Clean() error
}
