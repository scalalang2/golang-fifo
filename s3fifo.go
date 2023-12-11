package fifo

import (
	"fmt"
	"sync"
)

type s3fifoEntry[V any] struct {
	value V
	freq  byte
}

type S3FIFO[K comparable, V any] struct {
	lock sync.RWMutex

	// size is the maximum number of entries in the cache.
	size int

	// followings are the fundamental data structures of S3FIFO algorithm.
	items map[K]*s3fifoEntry[V]
	small *ringBuf[K]
	main  *ringBuf[K]
	ghost *bucketTable[K]
}

func NewS3FIFO[K comparable, V any](size int) Cache[K, V] {
	return &S3FIFO[K, V]{
		size:  size,
		items: make(map[K]*s3fifoEntry[V]),
		small: newRingBuf[K](size),
		main:  newRingBuf[K](size),
		ghost: newBucketTable[K](size),
	}
}

func (s *S3FIFO[K, V]) Set(key K, value V) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.items[key]; ok {
		s.items[key].value = value
		s.items[key].freq = min(s.items[key].freq+1, 3)
		return
	}

	for s.small.length()+s.main.length() >= s.size {
		s.evict()
	}

	if s.ghost.contains(key) {
		s.ghost.remove(key)
		if ok := s.main.push(key); !ok {
			panic("main ring buffer is full, this is unexpected bug")
		}
	} else {
		if ok := s.small.push(key); !ok {
			panic(fmt.Errorf("small ring buffer is full, this is unexpected bug, len:%d, cap: %d", s.small.length(), s.small.capacity()))
		}
	}

	ent := &s3fifoEntry[V]{value: value, freq: 0}
	s.items[key] = ent
}

func (s *S3FIFO[K, V]) Get(key K) (value V, ok bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.items[key]; !ok {
		return value, false
	}

	s.items[key].freq = min(s.items[key].freq+1, 3)
	s.ghost.remove(key)
	return s.items[key].value, true
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
	return ent.value, ok
}

func (s *S3FIFO[K, V]) Len() int {
	return s.small.length() + s.main.length()
}

func (s *S3FIFO[K, V]) Clean() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.items = make(map[K]*s3fifoEntry[V])
	s.small = newRingBuf[K](s.size)
	s.main = newRingBuf[K](s.size)
	s.ghost = newBucketTable[K](s.size)
}

func (s *S3FIFO[K, V]) evict() {
	mainCacheSize := s.size / 10 * 9
	if s.main.length() > mainCacheSize || s.small.length() == 0 {
		s.evictFromMain()
		return
	}
	s.evictFromSmall()
}

func (s *S3FIFO[K, V]) evictFromSmall() {
	evicted := false
	for !evicted && !s.small.isEmpty() {
		key := s.small.pop()
		if s.items[key].freq > 1 {
			s.main.push(key)
			if s.main.isFull() {
				s.evictFromMain()
			}
		} else {
			s.ghost.add(key)
			evicted = true
			delete(s.items, key)
		}
	}
}

func (s *S3FIFO[K, V]) evictFromMain() {
	evicted := false
	for !evicted && !s.main.isEmpty() {
		key := s.main.pop()
		if s.items[key].freq > 0 {
			s.main.push(key)
			s.items[key].freq--
		} else {
			evicted = true
			delete(s.items, key)
		}
	}
}
