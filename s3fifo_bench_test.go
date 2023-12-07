package fifo

import (
	"testing"

	lru "github.com/hashicorp/golang-lru/v2"
)

func BenchmarkS3FIFOCacheSet(b *testing.B) {
	const items = 1 << 16

	b.ReportAllocs()
	b.SetBytes(items)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache := NewS3FIFO[int, int](items)

			for i := 0; i < items; i++ {
				cache.Set(i, i)
			}
		}
	})
}

func BenchmarkS3FIFOCacheSetAndGet(b *testing.B) {
	const items = 1 << 16

	b.ReportAllocs()
	b.SetBytes(items)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache := NewS3FIFO[int, int](items)

			for i := 0; i < items; i++ {
				cache.Set(i, i)
			}

			for i := 0; i < items; i++ {
				cache.Get(i)
			}
		}
	})
}

func BenchmarkLRUCacheSet(b *testing.B) {
	const items = 1 << 16

	b.ReportAllocs()
	b.SetBytes(items)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache, err := lru.New[int, int](items)
			if err != nil {
				b.Fatal(err)
			}

			for i := 0; i < items; i++ {
				cache.Add(i, i)
			}
		}
	})
}

func BenchmarkLRUCacheSetAndGet(b *testing.B) {
	const items = 1 << 16

	b.ReportAllocs()
	b.SetBytes(items)

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache, err := lru.New[int, int](items)
			if err != nil {
				b.Fatal(err)
			}

			for i := 0; i < items; i++ {
				cache.Add(i, i)
			}

			for i := 0; i < items; i++ {
				cache.Get(i)
			}
		}
	})
}
