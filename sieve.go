package fifo

import (
	"container/list"
	"sync"
)

type sieveItem[K comparable, V any] struct {
	key     K
	value   V
	visited bool
}

type Sieve[K comparable, V any] struct {
	lock  sync.RWMutex
	size  int
	items map[K]*list.Element
	ll    *list.List
	hand  *list.Element
}

func NewSieve[K comparable, V any](size int) Cache[K, V] {
	s := &Sieve[K, V]{
		size:  size,
		items: make(map[K]*list.Element),
		ll:    list.New(),
	}
	return s
}

func (s *Sieve[K, V]) Set(key K, value V) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if s.ll.Len() >= s.size {
		s.evict()
	}

	if e, ok := s.items[key]; ok {
		e.Value.(*sieveItem[K, V]).visited = true
		e.Value.(*sieveItem[K, V]).value = value
		return
	}

	// push to the head
	ent := &sieveItem[K, V]{
		key:     key,
		value:   value,
		visited: false,
	}
	elem := s.ll.PushFront(ent)
	s.items[key] = elem
	if s.hand == nil {
		s.hand = elem
	}
}

func (s *Sieve[K, V]) Get(key K) (value V, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if item, ok := s.items[key]; ok {
		s.items[key].Value.(*sieveItem[K, V]).visited = true
		return item.Value.(*sieveItem[K, V]).value, true
	}

	return
}

func (s *Sieve[K, V]) Contains(key K) (ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if _, ok := s.items[key]; ok {
		return true
	}

	return false
}

func (s *Sieve[K, V]) Peek(key K) (value V, ok bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	item, ok := s.items[key]
	if !ok {
		return value, false
	}
	return item.Value.(*sieveItem[K, V]).value, true
}

func (s *Sieve[K, V]) Len() int {
	return s.ll.Len()
}

func (s *Sieve[K, V]) Clean() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.ll = list.New()
	s.items = make(map[K]*list.Element)
}

func (s *Sieve[K, V]) evict() {
	evicted := false
	for s.ll.Len() > 0 && !evicted {
		if s.hand == nil {
			s.hand = s.ll.Back()
		}
		if s.hand.Value.(*sieveItem[K, V]).visited {
			s.hand.Value.(*sieveItem[K, V]).visited = false
			s.hand = s.hand.Next()
		} else {
			evicted = true
		}
	}

	// evict
	if evicted {
		e := s.hand.Next()
		delete(s.items, s.hand.Value.(*sieveItem[K, V]).key)
		s.ll.Remove(s.hand)
		s.hand = e
		if s.hand == nil {
			s.hand = s.ll.Back()
		}
	}
}
