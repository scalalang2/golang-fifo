package s3fifo

import (
	"container/list"
	"sync"

	"github.com/scalalang2/golang-fifo"
)

type entry[K comparable, V any] struct {
	key   K
	value V
	freq  byte
}

type S3FIFO[K comparable, V any] struct {
	lock sync.RWMutex

	// size is the maximum number of entries in the cache.
	size int

	// followings are the fundamental data structures of S3FIFO algorithm.
	items map[K]*list.Element
	small *list.List
	main  *list.List
	ghost *bucketTable[K]
}

func New[K comparable, V any](size int) fifo.Cache[K, V] {
	return &S3FIFO[K, V]{
		size:  size,
		items: make(map[K]*list.Element),
		small: list.New(),
		main:  list.New(),
		ghost: newBucketTable[K](size),
	}
}

func (s *S3FIFO[K, V]) Set(key K, value V) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.items[key]; ok {
		el := s.items[key].Value.(*entry[K, V])
		el.value = value
		el.freq = min(el.freq+1, 3)
		return
	}

	for s.small.Len()+s.main.Len() >= s.size {
		s.evict()
	}

	// create a new entry to append it to the cache.
	ent := &entry[K, V]{
		key:   key,
		value: value,
		freq:  0,
	}

	if s.ghost.contains(key) {
		s.ghost.remove(key)
		s.items[key] = s.main.PushFront(ent)
	} else {
		s.items[key] = s.small.PushFront(ent)
	}
}

func (s *S3FIFO[K, V]) Get(key K) (value V, ok bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.items[key]; !ok {
		return value, false
	}

	ent := s.items[key].Value.(*entry[K, V])
	ent.freq = min(ent.freq+1, 3)
	s.ghost.remove(key)
	return ent.value, true
}

func (s *S3FIFO[K, V]) Remove(key K) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if e, ok := s.items[key]; ok {
		if s.ghost.contains(key) {
			s.ghost.remove(key)
		}

		s.main.Remove(e)
		s.small.Remove(e)
		delete(s.items, key)
	}
}

func (s *S3FIFO[K, V]) Contains(key K) (ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if _, ok := s.items[key]; ok {
		return true
	}
	return false
}

func (s *S3FIFO[K, V]) Peek(key K) (value V, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	ent, ok := s.items[key]
	if !ok {
		return value, false
	}
	return ent.Value.(*entry[K, V]).value, ok
}

func (s *S3FIFO[K, V]) Len() int {
	return s.small.Len() + s.main.Len()
}

func (s *S3FIFO[K, V]) Purge() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.items = make(map[K]*list.Element)
	s.small = list.New()
	s.main = list.New()
	s.ghost = newBucketTable[K](s.size)
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
		el := s.small.Back()
		key := el.Value.(*entry[K, V]).key
		if el.Value.(*entry[K, V]).freq > 1 {
			// move the entry from the small queue to the main queue
			s.items[key] = s.main.PushFront(el.Value)
			s.small.Remove(el)

			if s.main.Len() > mainCacheSize {
				s.evictFromMain()
			}
		} else {
			s.small.Remove(el)
			s.ghost.add(key)
			evicted = true
			delete(s.items, key)
		}
	}
}

func (s *S3FIFO[K, V]) evictFromMain() {
	evicted := false
	for !evicted && s.main.Len() > 0 {
		el := s.main.Back()
		key := el.Value.(*entry[K, V]).key
		if el.Value.(*entry[K, V]).freq > 0 {
			ent := el.Value.(*entry[K, V])
			ent.freq -= 1
			s.items[key] = s.main.PushFront(ent)
			s.main.Remove(el)
		} else {
			s.main.Remove(el)
			evicted = true
			delete(s.items, key)
		}
	}
}
