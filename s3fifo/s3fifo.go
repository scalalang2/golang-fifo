package s3fifo

import (
	"container/list"
	"context"
	"sync"
	"time"

	"github.com/scalalang2/golang-fifo/types"
)

const numberOfBuckets = 100

// entry holds the key and value of a cache entry.
type entry[K comparable, V any] struct {
	key       K
	value     V
	freq      byte
	element   *list.Element
	expiredAt time.Time
	bucketID  int8 // bucketID is an index which the entry is stored in the bucket
}

// bucket is a container holding entries to be expired
// ref. hashicorp/golang-lru
type bucket[K comparable, V any] struct {
	entries     map[K]*entry[K, V]
	newestEntry time.Time
}

type S3FIFO[K comparable, V any] struct {
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex

	// size is the maximum number of entries in the cache.
	size int

	// followings are the fundamental data structures of S3FIFO algorithm.
	items map[K]*entry[K, V]
	small *list.List
	main  *list.List
	ghost *ghost[K]

	buckets []bucket[K, V]

	// ttl is the time to live of the cache entry
	ttl time.Duration

	// nextCleanupBucket is an index of the next bucket to be cleaned up
	nextCleanupBucket int8

	// callback is the function that will be called when an entry is evicted from the cache
	callback types.OnEvictCallback[K, V]
}

var _ types.Cache[int, int] = (*S3FIFO[int, int])(nil)

func New[K comparable, V any](size int, ttl time.Duration) *S3FIFO[K, V] {
	ctx, cancel := context.WithCancel(context.Background())

	if ttl <= 0 {
		ttl = 0
	}

	cache := &S3FIFO[K, V]{
		ctx:               ctx,
		cancel:            cancel,
		size:              size,
		items:             make(map[K]*entry[K, V]),
		small:             list.New(),
		main:              list.New(),
		ghost:             newGhost[K](size),
		buckets:           make([]bucket[K, V], numberOfBuckets),
		ttl:               ttl,
		nextCleanupBucket: 0,
	}

	for i := 0; i < numberOfBuckets; i++ {
		cache.buckets[i].entries = make(map[K]*entry[K, V])
	}

	if ttl != 0 {
		go func(ctx context.Context) {
			ticker := time.NewTicker(ttl / numberOfBuckets)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					cache.deleteExpired()
				}
			}
		}(cache.ctx)
	}

	return cache
}

func (s *S3FIFO[K, V]) Set(key K, value V) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if el, ok := s.items[key]; ok {
		s.removeFromBucket(el) // remove from the bucket as the entry is updated
		el.value = value
		el.freq = min(el.freq+1, 3)
		el.expiredAt = time.Now().Add(s.ttl)
		s.addToBucket(el)
		return
	}

	for s.small.Len()+s.main.Len() >= s.size {
		s.evict()
	}

	// create a new entry to append it to the cache.
	ent := &entry[K, V]{
		key:       key,
		value:     value,
		freq:      0,
		expiredAt: time.Now().Add(s.ttl),
	}

	if s.ghost.contains(key) {
		s.ghost.remove(key)
		ent.element = s.main.PushFront(key)
	} else {
		ent.element = s.small.PushFront(key)
	}

	s.items[key] = ent
	s.addToBucket(ent)
}

func (s *S3FIFO[K, V]) Get(key K) (value V, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[key]; !ok {
		return value, false
	}

	s.items[key].freq = min(s.items[key].freq+1, 3)
	s.ghost.remove(key)
	return s.items[key].value, true
}

func (s *S3FIFO[K, V]) Remove(key K) (ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e, ok := s.items[key]; ok {
		s.removeEntry(e, types.EvictReasonRemoved)
		return true
	}

	return false
}

func (s *S3FIFO[K, V]) Contains(key K) (ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.items[key]; ok {
		return true
	}
	return false
}

func (s *S3FIFO[K, V]) Peek(key K) (value V, ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	el, ok := s.items[key]
	if !ok {
		return value, false
	}
	return el.value, ok
}

func (s *S3FIFO[K, V]) SetOnEvicted(callback types.OnEvictCallback[K, V]) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.callback = callback
}

func (s *S3FIFO[K, V]) Len() int {
	return s.small.Len() + s.main.Len()
}

func (s *S3FIFO[K, V]) Purge() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for k := range s.items {
		delete(s.items, k)
	}

	s.small.Init()
	s.main.Init()
	s.ghost.clear()
}

func (s *S3FIFO[K, V]) Close() {
	s.Purge()
	s.mu.Lock()
	s.cancel()
	s.mu.Unlock()
}

func (s *S3FIFO[K, V]) removeEntry(e *entry[K, V], reason types.EvictReason) {
	if s.callback != nil {
		s.callback(e.key, e.value, reason)
	}

	if s.ghost.contains(e.key) {
		s.ghost.remove(e.key)
	}

	s.main.Remove(e.element)
	s.small.Remove(e.element)
	delete(s.items, e.key)
}

func (s *S3FIFO[K, V]) addToBucket(e *entry[K, V]) {
	if s.ttl == 0 {
		return
	}
	bucketId := (numberOfBuckets + s.nextCleanupBucket - 1) % numberOfBuckets
	e.bucketID = bucketId
	s.buckets[bucketId].entries[e.key] = e
	if s.buckets[bucketId].newestEntry.Before(e.expiredAt) {
		s.buckets[bucketId].newestEntry = e.expiredAt
	}
}

func (s *S3FIFO[K, V]) removeFromBucket(e *entry[K, V]) {
	if s.ttl == 0 {
		return
	}
	delete(s.buckets[e.bucketID].entries, e.key)
}

func (s *S3FIFO[K, V]) deleteExpired() {
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

func (s *S3FIFO[K, V]) evict() {
	// if size of the small queue is greater than 10% of the total cache size.
	// then, evict from the small queue
	if s.small.Len() > s.size/10 {
		s.evictFromSmall()
		return
	}
	s.evictFromMain()
}

func (s *S3FIFO[K, V]) evictFromSmall() {
	mainCacheSize := s.size / 10 * 9

	evicted := false
	for !evicted && s.small.Len() > 0 {
		key := s.small.Back().Value.(K)
		el, ok := s.items[key]
		if !ok {
			panic("s3fifo: entry not found in the cache")
		}

		if el.freq > 1 {
			// move the entry from the small queue to the main queue
			s.small.Remove(el.element)
			s.items[key].element = s.main.PushFront(el.key)

			if s.main.Len() > mainCacheSize {
				s.evictFromMain()
			}
		} else {
			s.removeEntry(el, types.EvictReasonEvicted)
			s.ghost.add(key)
			evicted = true
			delete(s.items, key)
		}
	}
}

func (s *S3FIFO[K, V]) evictFromMain() {
	evicted := false
	for !evicted && s.main.Len() > 0 {
		key := s.main.Back().Value.(K)
		el, ok := s.items[key]
		if !ok {
			panic("s3fifo: entry not found in the cache")
		}

		if el.freq > 0 {
			s.main.Remove(el.element)
			s.items[key].freq -= 1
			s.items[key].element = s.main.PushFront(el.key)
		} else {
			s.removeEntry(el, types.EvictReasonEvicted)
			evicted = true
			delete(s.items, key)
		}
	}
}
