package sieve

import (
	"container/list"
	"sync"

	"github.com/scalalang2/golang-fifo"
)

// entry holds the key and value of a cache entry.
type entry[K comparable, V any] struct {
	key     K
	value   V
	visited bool
}

type Sieve[K comparable, V any] struct {
	lock  sync.Mutex
	size  int
	items map[K]*list.Element
	ll    *list.List
	hand  *list.Element
}

func New[K comparable, V any](size int) fifo.Cache[K, V] {
	return &Sieve[K, V]{
		size:  size,
		items: make(map[K]*list.Element),
		ll:    list.New(),
	}
}

func (s *Sieve[K, V]) Set(key K, value V) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if e, ok := s.items[key]; ok {
		e.Value.(*entry[K, V]).value = value
		e.Value.(*entry[K, V]).visited = true
		return
	}

	if s.ll.Len() >= s.size {
		s.evict()
	}
	e := &entry[K, V]{key: key, value: value}
	s.items[key] = s.ll.PushFront(e)
}

func (s *Sieve[K, V]) Get(key K) (value V, ok bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if e, ok := s.items[key]; ok {
		e.Value.(*entry[K, V]).visited = true
		return e.Value.(*entry[K, V]).value, true
	}

	return
}

func (s *Sieve[K, V]) Remove(key K) (ok bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if e, ok := s.items[key]; ok {
		// if the element to be removed is the hand,
		// then move the hand to the previous one.
		if e == s.hand {
			s.hand = s.hand.Prev()
		}
		s.ll.Remove(e)
		delete(s.items, key)

		return true
	}

	return false
}

func (s *Sieve[K, V]) Contains(key K) (ok bool) {
	s.lock.Lock()
	defer s.lock.Unlock()
	_, ok = s.items[key]
	return
}

func (s *Sieve[K, V]) Peek(key K) (value V, ok bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if e, ok := s.items[key]; ok {
		return e.Value.(*entry[K, V]).value, true
	}

	return
}

func (s *Sieve[K, V]) Len() int {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.ll.Len()
}

func (s *Sieve[K, V]) Purge() {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.items = make(map[K]*list.Element)
	s.ll = list.New()
}

func (s *Sieve[K, V]) evict() {
	o := s.hand
	// if o is nil, then assign it to the tail element in the list
	if o == nil {
		o = s.ll.Back()
	}

	for o.Value.(*entry[K, V]).visited {
		o.Value.(*entry[K, V]).visited = false
		o = o.Prev()
		if o == nil {
			o = s.ll.Back()
		}
	}

	s.hand = o.Prev()
	delete(s.items, o.Value.(*entry[K, V]).key)
	s.ll.Remove(o)
}
