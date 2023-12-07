package fifo

import (
	"fmt"
	"sync"

	"github.com/scalalang2/golang-fifo/v1/internal"
)

var ErrKeyNotFound = fmt.Errorf("key not found")

type S3FIFO[K comparable, V any] struct {
	lock sync.RWMutex

	// size is the maximum number of entries in the cache.
	size int

	// followings are the fundamental data structures of S3FIFO algorithm.
	items map[K]*internal.Entry[K, V]
	small *ringBuf[K]
	main  *ringBuf[K]
	ghost *bucketTable[K]
}

func NewS3FIFO[K comparable, V any](size int) *S3FIFO[K, V] {
	return &S3FIFO[K, V]{
		size:  size,
		items: make(map[K]*internal.Entry[K, V]),
		small: newRingBuf[K](size),
		main:  newRingBuf[K](size),
		ghost: newBucketHash[K](size),
	}
}

func (s *S3FIFO[K, V]) Set(key K, value V) {
	s.lock.Lock()
	if s.small.length()+s.main.length() >= s.size {
		s.evict()
	}

	ent := &internal.Entry[K, V]{
		Key:  key,
		Val:  value,
		Freq: 0,
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
	s.items[key] = ent
	s.lock.Unlock()
}

func (s *S3FIFO[K, V]) Get(key K) (value V, ok bool) {
	s.lock.RLock()
	if _, ok := s.items[key]; !ok {
		return value, false
	}
	entry := s.items[key]
	entry.Freq = min(entry.Freq+1, 3)
	s.lock.RUnlock()
	return entry.Val, true
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

	entry, ok := s.items[key]
	if !ok {
		return value, false
	}

	s.lock.RUnlock()
	return entry.Val, true
}

func (s *S3FIFO[K, V]) Len() int {
	return s.small.length() + s.main.length()
}

func (s *S3FIFO[K, V]) Clean() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.items = make(map[K]*internal.Entry[K, V])
	s.small = newRingBuf[K](s.size)
	s.main = newRingBuf[K](s.size)
	s.ghost = newBucketHash[K](s.size)
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
		if s.items[key].Freq > 1 {
			s.main.push(key)
			if s.main.isFull() {
				s.evictFromMain()
			}
		} else {
			evicted = true
			s.ghost.remove(key)
			delete(s.items, key)
		}
	}
}

func (s *S3FIFO[K, V]) evictFromMain() {
	evicted := false
	for !evicted && !s.main.isEmpty() {
		key := s.main.pop()
		if s.items[key].Freq > 0 {
			s.main.push(key)
			s.items[key].Freq--
		} else {
			s.ghost.remove(key)
			evicted = true
		}
	}
}
