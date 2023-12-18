package s3fifo

import "container/list"

type bucketTable[K comparable] struct {
	size  int
	ll    *list.List
	items map[K]*list.Element
}

func newBucketTable[K comparable](size int) *bucketTable[K] {
	return &bucketTable[K]{
		size:  size,
		ll:    list.New(),
		items: make(map[K]*list.Element),
	}
}

func (b *bucketTable[K]) add(key K) {
	if _, ok := b.items[key]; ok {
		return
	}

	for b.ll.Len() >= b.size {
		e := b.ll.Back()
		delete(b.items, e.Value.(K))
		b.ll.Remove(e)
	}

	e := b.ll.PushFront(key)
	b.items[key] = e
}

func (b *bucketTable[K]) remove(key K) {
	if e, ok := b.items[key]; ok {
		b.ll.Remove(e)
		delete(b.items, key)
	}
}

func (b *bucketTable[K]) contains(key K) bool {
	_, ok := b.items[key]
	return ok
}
