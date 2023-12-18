package s3fifo

// ringBuf is a non thread-safe ring buffer implementation.
type ringBuf[K comparable] struct {
	buf                  []K
	head, tail, len, cap int
	full                 bool
}

func newRingBuf[K comparable](size int) *ringBuf[K] {
	return &ringBuf[K]{
		buf:  make([]K, size),
		cap:  size,
		full: false,
	}
}

func (r *ringBuf[K]) push(v K) bool {
	if r.full {
		return false
	}

	r.buf[r.tail] = v
	r.tail = (r.tail + 1) % r.cap
	r.len++

	if r.tail == r.head {
		r.full = true
	}
	return true
}

func (r *ringBuf[K]) peek() K {
	if r.len <= 0 {
		panic("ringBuf: peek() called on empty ring buffer")
	}
	return r.buf[r.head]
}

func (r *ringBuf[K]) pop() K {
	if r.len <= 0 {
		panic("ringBuf: pop() called on empty ring buffer")
	}

	var defaultVal K
	v := r.buf[r.head]
	r.buf[r.head] = defaultVal
	r.head = (r.head + 1) % r.cap
	r.len--
	r.full = false
	return v
}

func (r *ringBuf[K]) length() int {
	return r.len
}

func (r *ringBuf[K]) capacity() int {
	return r.cap
}

func (r *ringBuf[K]) isFull() bool {
	return r.full
}

func (r *ringBuf[K]) isEmpty() bool {
	return r.len <= 0
}
