package types

type OnEvictCallback[K comparable, V any] func(key K, value V)

// Cache is the interface for a cache.
type Cache[K comparable, V any] interface {
	// Set sets the value for the given key on cache.
	Set(key K, value V)

	// Get gets the value for the given key from cache.
	Get(key K) (value V, ok bool)

	// Remove removes the provided key from the cache.
	Remove(key K) (ok bool)

	// Contains check if a key exists in cache without updating the recent-ness
	Contains(key K) (ok bool)

	// Peek returns key's value without updating the recent-ness.
	Peek(key K) (value V, ok bool)

	// SetOnEvict sets the callback function that will be called when an entry is evicted from the cache.
	SetOnEvict(callback OnEvictCallback[K, V])

	// Len returns the number of entries in the cache.
	Len() int

	// Purge clears all cache entries
	Purge()

	// Close closes the cache and releases any resources associated with it.
	Close()
}
