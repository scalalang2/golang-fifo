package types

// EvictReason is the reason for an entry to be evicted from the cache.
// It is used in the [OnEvictCallback] function.
type EvictReason int

const (
	// EvictReasonExpired is used when an item is removed because its TTL has expired.
	EvictReasonExpired = iota
	// EvictReasonEvicted is used when an item is removed because the cache size limit was exceeded.
	EvictReasonEvicted
	// EvictReasonRemoved is used when an item is explicitly deleted.
	EvictReasonRemoved
)

type OnEvictCallback[K comparable, V any] func(key K, value V, reason EvictReason)

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

	// SetOnEvicted sets the callback function that will be called when an entry is evicted from the cache.
	SetOnEvicted(callback OnEvictCallback[K, V])

	// Len returns the number of entries in the cache.
	Len() int

	// Purge clears all cache entries
	Purge()

	// Close closes the cache and releases any resources associated with it.
	Close()
}
