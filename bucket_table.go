package fifo

type bucketTable[K comparable] struct {
	queue *ringBuf[K]
	hash  map[K]bool
}

func newBucketTable[K comparable](size int) *bucketTable[K] {
	return &bucketTable[K]{
		queue: newRingBuf[K](size),
		hash:  make(map[K]bool),
	}
}

func (b *bucketTable[K]) add(key K) {
	if b.queue.isFull() {
		old := b.queue.pop()
		delete(b.hash, old)
	}

	b.queue.push(key)
}

func (b *bucketTable[K]) remove(key K) {
	delete(b.hash, key)
}

func (b *bucketTable[K]) contains(key K) bool {
	return b.hash[key]
}
