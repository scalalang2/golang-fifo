package fifo

// Cache is the interface for a cache.
type Cache[K comparable, V any] interface {
	// Set sets the value for the given key on cache.
	Set(key K, value V)

	// Get gets the value for the given key from cache.
	Get(key K) (value V, ok bool)

	// Contains check if a key exists in cache without updating the recent-ness
	Contains(key K) (ok bool)

	// Peek returns key's value without updating the recent-ness.
	Peek(key K) (value V, ok bool)

	// Len returns the number of entries in the cache.
	Len() int

	// Purge clears all cache entries
	Purge()
}
