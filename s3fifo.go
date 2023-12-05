package fifo

import (
	"fmt"
	"sync"

	"github.com/scalalang2/golang-fifo/v1/queue"
)

const initCapacityInBytes = 1 << 20
const maxCapacityInBytes = 1 << 30

var ErrKeyNotFound = fmt.Errorf("key not found")

type S3FIFO[K comparable, V any] struct {
	lock sync.RWMutex

	// size is the maximum number of entries in the cache.
	size int

	// followings are the fundamental data structures of S3FIFO algorithm.
	freq    map[uint64]byte
	shortHM map[uint64]uint64 // shortHM represents short-term hashmap index
	longHM  map[uint64]uint64 // longHM represents long-term hashmap index
	short   *queue.BytesQueue
	long    *queue.BytesQueue
	ghost   map[uint64]bool // TODO: ghost should be replaced to the bucket-based hash table.
}

func NewS3FIFO[K comparable, V any](maxSize int) *S3FIFO[K, V] {
	return &S3FIFO[K, V]{
		size: maxSize,

		freq:    make(map[uint64]byte),
		shortHM: make(map[uint64]uint64),
		longHM:  make(map[uint64]uint64),
		short:   queue.NewBytesQueue(initCapacityInBytes, maxCapacityInBytes, false),
		long:    queue.NewBytesQueue(initCapacityInBytes, maxCapacityInBytes, false),
		ghost:   make(map[uint64]bool),
	}
}

func (s *S3FIFO[K, V]) Set(key K, value V) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	for s.Len() >= s.size {
		if err := s.evict(); err != nil {
			return err
		}
	}

	hashKey, err := fnvHash(key)
	if err != nil {
		return err
	}

	blob, err := wrapEntry(hashKey, value)
	if err != nil {
		return err
	}

	if _, exist := s.ghost[hashKey]; exist {
		idx, err := s.long.Push(blob)
		if err != nil {
			return err
		}

		s.longHM[hashKey] = uint64(idx)
		delete(s.ghost, hashKey)
	} else {
		idx, err := s.short.Push(blob)
		if err != nil {
			return err
		}

		s.shortHM[hashKey] = uint64(idx)
	}

	return nil
}

func (s *S3FIFO[K, V]) Get(key K) (value V, err error) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	hashKey, err := fnvHash(key)
	if err != nil {
		return value, err
	}

	// TODO: It needs to handle hash collision
	if idx, exist := s.shortHM[hashKey]; exist {
		blob, err := s.short.Get(int(idx))
		if err != nil {
			return value, err
		}

		_, value, err = unwrapEntry[V](blob)
		if err != nil {
			return value, err
		}

		s.freq[hashKey] = min(s.freq[hashKey]+1, 3)
		return value, nil
	}

	if idx, exist := s.longHM[hashKey]; exist {
		blob, err := s.long.Get(int(idx))
		if err != nil {
			return value, err
		}

		_, value, err = unwrapEntry[V](blob)
		if err != nil {
			return value, err
		}

		s.freq[hashKey] = min(s.freq[hashKey]+1, 3)
		return value, nil
	}

	return value, ErrKeyNotFound
}

func (s *S3FIFO[K, V]) Len() int {
	return s.short.Len() + s.long.Len()
}

func (s *S3FIFO[K, V]) evict() error {
	// if length of short-term queue is more than 10% of total cache size,
	// then evict the oldest entry in short-term queue.
	if s.short.Len() >= s.Len()/10 {
		return s.evictShort()
	} else {
		return s.evictLong()
	}
}

func (s *S3FIFO[K, V]) evictShort() error {
	evicted := false
	for !evicted && s.short.Len() > 0 {
		data, err := s.short.Peek()
		if err != nil {
			return err
		}

		hashKey, _, err := unwrapEntry[V](data)
		if err != nil {
			return err
		}

		if s.freq[hashKey] > 1 {
			// if length of long-term queue is more than 90% of total cache size,
			// then evict the oldest entry in long-term queue.
			if s.long.Len()+1 >= s.Len()*9/10 {
				if err := s.evictLong(); err != nil {
					return err
				}
			}

			idx, err := s.long.Push(data)
			if err != nil {
				return err
			}
			s.longHM[hashKey] = uint64(idx)
		} else {
			s.ghost[hashKey] = true
			delete(s.freq, hashKey)
			evicted = true
		}

		delete(s.shortHM, hashKey)
		_, err = s.short.Pop()
		return err
	}

	return nil
}

func (s *S3FIFO[K, V]) evictLong() error {
	evicted := false
	for !evicted && s.long.Len() > 0 {
		data, err := s.long.Pop()
		if err != nil {
			return err
		}

		hashKey, _, err := unwrapEntry[V](data)
		if err != nil {
			return err
		}

		delete(s.longHM, hashKey)

		if s.freq[hashKey] > 0 {
			// if freq is greater than or equal to 1, then promote the entry to the head.
			s.freq[hashKey]--
			idx, err := s.long.Push(data)
			if err != nil {
				return err
			}
			s.longHM[hashKey] = uint64(idx)
		} else {
			evicted = true
		}
	}

	return nil
}
