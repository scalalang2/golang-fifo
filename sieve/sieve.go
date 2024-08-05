package sieve

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/scalalang2/golang-fifo/types"
)

// numberOfBuckets is the number of buckets to store the cache entries
//
// Notice: if this number exceeds 256, the type of nextCleanupBucket
// in the Sieve struct should be changed to int16
const numberOfBuckets = 100

// entry holds the key and value of a cache entry.
type entry[K comparable, V any] struct {
	key       K
	value     V
	visited   bool
	element   *list.Element
	expiredAt time.Time
	bucketID  int8 // bucketID is an index which the entry is stored in the bucket
}

// bucket is a container holding entries to be expired
type bucket[K comparable, V any] struct {
	entries     map[K]*entry[K, V]
	newestEntry time.Time
}

type Sieve[K comparable, V any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	size   int
	items  map[K]*entry[K, V]
	ll     *list.List
	hand   *list.Element

	buckets []bucket[K, V]

	// ttl is the time to live of the cache entry
	ttl time.Duration

	// nextCleanupBucket is an index of the next bucket to be cleaned up
	nextCleanupBucket int8

	// callback is the function that will be called when an entry is evicted from the cache
	callback types.OnEvictCallback[K, V]
}

var _ types.Cache[int, int] = (*Sieve[int, int])(nil)

func New[K comparable, V any](size int, ttl time.Duration) *Sieve[K, V] {
	ctx, cancel := context.WithCancel(context.Background())

	if ttl <= 0 {
		ttl = 0
	}

	if size <= 0 {
		panic("sieve: size must be greater than 0")
	}

	cache := &Sieve[K, V]{
		ctx:               ctx,
		cancel:            cancel,
		size:              size,
		items:             make(map[K]*entry[K, V]),
		ll:                list.New(),
		buckets:           make([]bucket[K, V], numberOfBuckets),
		ttl:               ttl,
		nextCleanupBucket: 0,
	}

	for i := 0; i < numberOfBuckets; i++ {
		cache.buckets[i].entries = make(map[K]*entry[K, V])
	}

	if ttl != 0 {
		go cache.cleanup(cache.ctx)
	}

	return cache
}

func (s *Sieve[K, V]) cleanup(ctx context.Context) {
	ticker := time.NewTicker(s.ttl / numberOfBuckets)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.deleteExpired()
		}
	}
}

func (s *Sieve[K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e, ok := s.items[key]; ok {
		s.removeFromBucket(e) // remove from the bucket as the entry is updated
		e.value = value
		e.visited = true
		e.expiredAt = time.Now().Add(s.ttl)
		s.addToBucket(e)
		return
	}

	if s.ll.Len() >= s.size {
		s.evict()
	}

	e := &entry[K, V]{
		key:       key,
		value:     value,
		element:   s.ll.PushFront(key),
		expiredAt: time.Now().Add(s.ttl),
	}
	s.items[key] = e
	s.addToBucket(e)
}

func (s *Sieve[K, V]) Get(key K) (value V, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.items[key]; ok {
		e.visited = true
		return e.value, true
	}

	return
}

func (s *Sieve[K, V]) Remove(key K) (ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e, ok := s.items[key]; ok {
		// if the element to be removed is the hand,
		// then move the hand to the previous one.
		if e.element == s.hand {
			s.hand = s.hand.Prev()
		}

		s.removeEntry(e, types.EvictReasonRemoved)
		return true
	}

	return false
}

func (s *Sieve[K, V]) Contains(key K) (ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, ok = s.items[key]
	return
}

func (s *Sieve[K, V]) Peek(key K) (value V, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if e, ok := s.items[key]; ok {
		return e.value, true
	}

	return
}

func (s *Sieve[K, V]) SetOnEvicted(callback types.OnEvictCallback[K, V]) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.callback = callback
}

func (s *Sieve[K, V]) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.ll.Len()
}

func (s *Sieve[K, V]) Purge() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, e := range s.items {
		s.removeEntry(e, types.EvictReasonRemoved)
	}

	for i := range s.buckets {
		for k := range s.buckets[i].entries {
			delete(s.buckets[i].entries, k)
		}
	}

	// hand pointer must also be reset
	s.hand = nil
	s.nextCleanupBucket = 0
	s.ll.Init()
}

func (s *Sieve[K, V]) Close() {
	s.Purge()
	s.mu.Lock()
	s.cancel()
	s.mu.Unlock()
}

func (s *Sieve[K, V]) removeEntry(e *entry[K, V], reason types.EvictReason) {
	if s.callback != nil {
		s.callback(e.key, e.value, reason)
	}

	s.ll.Remove(e.element)
	s.removeFromBucket(e)
	delete(s.items, e.key)
}

func (s *Sieve[K, V]) evict() {
	o := s.hand
	// if o is nil, then assign it to the tail element in the list
	if o == nil {
		o = s.ll.Back()
	}

	el, ok := s.items[o.Value.(K)]
	if !ok {
		panic("sieve: evicting non-existent element")
	}

	for el.visited {
		el.visited = false
		o = o.Prev()
		if o == nil {
			o = s.ll.Back()
		}

		el, ok = s.items[o.Value.(K)]
		if !ok {
			panic("sieve: evicting non-existent element")
		}
	}

	s.hand = o.Prev()
	s.removeEntry(el, types.EvictReasonEvicted)
}

func (s *Sieve[K, V]) addToBucket(e *entry[K, V]) {
	if s.ttl == 0 {
		return
	}
	bucketId := (numberOfBuckets + int(s.nextCleanupBucket) - 1) % numberOfBuckets
	e.bucketID = int8(bucketId)
	s.buckets[bucketId].entries[e.key] = e
	if s.buckets[bucketId].newestEntry.Before(e.expiredAt) {
		s.buckets[bucketId].newestEntry = e.expiredAt
	}
}

func (s *Sieve[K, V]) removeFromBucket(e *entry[K, V]) {
	if s.ttl == 0 {
		return
	}
	delete(s.buckets[e.bucketID].entries, e.key)
}

func (s *Sieve[K, V]) deleteExpired() {
	s.mu.Lock()

	bucketId := s.nextCleanupBucket
	s.nextCleanupBucket = (s.nextCleanupBucket + 1) % numberOfBuckets
	bucket := &s.buckets[bucketId]
	timeToExpire := time.Until(bucket.newestEntry)
	if timeToExpire > 0 {
		s.mu.Unlock()
		time.Sleep(timeToExpire)
		s.mu.Lock()
	}

	for _, e := range bucket.entries {
		s.removeEntry(e, types.EvictReasonExpired)
	}

	s.mu.Unlock()
}
