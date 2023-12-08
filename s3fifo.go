package fifo

import (
	"fmt"
	"sync"
)

type S3FIFO[K comparable, V any] struct {
	lock sync.RWMutex

	// size is the maximum number of entries in the cache.
	size int

	// followings are the fundamental data structures of S3FIFO algorithm.
	items map[K]V
	freq  map[K]byte
	small *ringBuf[K]
	main  *ringBuf[K]
	ghost *bucketTable[K]
}

func NewS3FIFO[K comparable, V any](size int) Cache[K, V] {
	return &S3FIFO[K, V]{
		size:  size,
		items: make(map[K]V),
		small: newRingBuf[K](size),
		main:  newRingBuf[K](size),
		ghost: newBucketTable[K](size),
		freq:  make(map[K]byte),
	}
}

func (s *S3FIFO[K, V]) Set(key K, value V) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.small.length()+s.main.length() >= s.size {
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
	s.items[key] = value
}

func (s *S3FIFO[K, V]) Get(key K) (value V, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	// on cache miss
	if _, ok := s.items[key]; !ok {
		if s.small.length()+s.main.length() >= s.size {
			s.evict()
		}
		if s.ghost.contains(key) {
			s.ghost.remove(key)
			if ok := s.main.push(key); !ok {
				panic("get(): main ring buffer is full, this is unexpected bug")
			}
		} else {
			if ok := s.small.push(key); !ok {
				panic(fmt.Errorf("get(): small ring buffer is full, this is unexpected bug, len:%d, cap: %d", s.small.length(), s.small.capacity()))
			}
		}
		return value, false
	}

	s.freq[key] = min(s.freq[key]+1, 3)
	return s.items[key], true
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

	value, ok = s.items[key]
	return
}

func (s *S3FIFO[K, V]) Len() int {
	return s.small.length() + s.main.length()
}

func (s *S3FIFO[K, V]) Clean() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.items = make(map[K]V)
	s.small = newRingBuf[K](s.size)
	s.main = newRingBuf[K](s.size)
	s.ghost = newBucketTable[K](s.size)
}

func (s *S3FIFO[K, V]) evict() {
	if s.small.length() >= s.size/10 {
		s.evictFromSmall()
	} else {
		s.evictFromMain()
	}
}

func (s *S3FIFO[K, V]) evictFromSmall() {
	evicted := false
	for !evicted && !s.small.isEmpty() {
		key := s.small.pop()
		if s.freq[key] > 1 {
			if s.main.isFull() {
				s.evictFromMain()
			}
			s.main.push(key)
		} else {
			evicted = true
			s.ghost.add(key)
			delete(s.freq, key)
			delete(s.items, key)
		}
	}
}

func (s *S3FIFO[K, V]) evictFromMain() {
	evicted := false
	for !evicted && !s.main.isEmpty() {
		key := s.main.pop()
		if s.freq[key] > 0 {
			s.main.push(key)
			s.freq[key]--
		} else {
			evicted = true
			s.ghost.remove(key)
			delete(s.freq, key)
			delete(s.items, key)
		}
	}
}
