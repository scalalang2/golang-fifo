package s3fifo

import (
	"sync"
	"testing"
	"time"

	"fortio.org/assert"
	"github.com/scalalang2/golang-fifo/types"
)

const noEvictionTTL = 0

func TestSetAndGet(t *testing.T) {
	cache := New[string, string](10, noEvictionTTL)
	cache.Set("hello", "world")

	value, ok := cache.Get("hello")
	assert.True(t, ok)
	assert.Equal(t, "world", value)
}

func TestRemove(t *testing.T) {
	cache := New[int, int](10, noEvictionTTL)
	cache.Set(1, 10)

	val, ok := cache.Get(1)
	assert.True(t, ok)
	assert.Equal(t, 10, val)

	// After removing the key, it should not be found
	removed := cache.Remove(1)
	assert.True(t, removed)

	_, ok = cache.Get(1)
	assert.False(t, ok)

	// This should not panic
	removed = cache.Remove(-1)
	assert.False(t, removed)
}

func TestEvictOneHitWonders(t *testing.T) {
	cache := New[int, int](10, noEvictionTTL)
	oneHitWonders := []int{1, 2}
	popularObjects := []int{3, 4, 5, 6, 7, 8, 9, 10}

	// add objects to the cache
	for _, v := range oneHitWonders {
		cache.Set(v, v)
	}
	for _, v := range popularObjects {
		cache.Set(v, v)
	}

	// hit one-hit wonders only once
	for _, v := range oneHitWonders {
		_, ok := cache.Get(v)
		assert.True(t, ok)
	}

	// hit the popular objects
	for i := 0; i < 3; i++ {
		for _, v := range popularObjects {
			_, ok := cache.Get(v)
			assert.True(t, ok)
		}
	}

	// add more objects to the cache
	// this should evict the one hit wonders
	for i := 11; i < 20; i++ {
		cache.Set(i, i)
	}

	for _, v := range oneHitWonders {
		_, ok := cache.Get(v)
		assert.False(t, ok)
	}

	// popular objects should still be in the cache
	for _, v := range popularObjects {
		_, ok := cache.Get(v)
		assert.True(t, ok)
	}
}

func TestPeek(t *testing.T) {
	cache := New[int, int](5, noEvictionTTL)
	entries := []int{1, 2, 3, 4, 5}

	for _, v := range entries {
		cache.Set(v, v*10)
	}

	// peek each entry 10 times
	// this should not change the recent-ness of the entry.
	for i := 0; i < 10; i++ {
		for _, v := range entries {
			value, exist := cache.Peek(v)
			assert.True(t, exist)
			assert.Equal(t, v*10, value)
		}
	}

	// add more entries to the cache
	// this should evict the first 5 entries
	for i := 6; i <= 10; i++ {
		cache.Set(i, i*10)
	}

	// peek the first 5 entries
	// they should not exist in the cache
	for _, v := range entries {
		_, exist := cache.Peek(v)
		assert.False(t, exist)
	}
}

func TestContains(t *testing.T) {
	cache := New[int, int](5, noEvictionTTL)
	entries := []int{1, 2, 3, 4, 5}

	for _, v := range entries {
		cache.Set(v, v*10)
	}

	// check if each entry exists in the cache
	for _, v := range entries {
		assert.True(t, cache.Contains(v))
	}

	for i := 6; i <= 10; i++ {
		assert.False(t, cache.Contains(i))
	}
}

func TestLength(t *testing.T) {
	cache := New[string, string](10, noEvictionTTL)

	cache.Set("hello", "world")
	assert.Equal(t, 1, cache.Len())

	cache.Set("hello2", "world")
	cache.Set("hello", "changed")
	assert.Equal(t, 2, cache.Len())

	value, ok := cache.Get("hello")
	assert.True(t, ok)
	assert.Equal(t, "changed", value)
}

func TestClean(t *testing.T) {
	cache := New[int, int](10, noEvictionTTL)
	entries := []int{1, 2, 3, 4, 5}

	for _, v := range entries {
		cache.Set(v, v*10)
	}
	assert.Equal(t, 5, cache.Len())
	cache.Purge()

	// check if each entry exists in the cache
	for _, v := range entries {
		_, exist := cache.Peek(v)
		assert.False(t, exist)
	}
	assert.Equal(t, 0, cache.Len())
}

func TestTimeToLive(t *testing.T) {
	ttl := time.Second
	cache := New[int, int](10, ttl)
	numberOfEntries := 10

	for num := 1; num <= numberOfEntries; num++ {
		cache.Set(num, num)
		val, ok := cache.Get(num)
		assert.True(t, ok)
		assert.Equal(t, num, val)
	}

	time.Sleep(ttl * 2)

	// check all entries are evicted
	for num := 1; num <= numberOfEntries; num++ {
		_, ok := cache.Get(num)
		assert.False(t, ok)
	}
}

func TestEvictionCallback(t *testing.T) {
	cache := New[int, int](10, noEvictionTTL)
	evicted := make(map[int]int)

	cache.SetOnEvicted(func(key int, value int, _ types.EvictReason) {
		evicted[key] = value
	})

	// add objects to the cache
	for i := 1; i <= 10; i++ {
		cache.Set(i, i)
	}

	// add another object to the cache
	cache.Set(11, 11)

	// check the first object is evicted
	_, ok := cache.Get(1)
	assert.False(t, ok)
	assert.Equal(t, 1, evicted[1])

	cache.Close()
}

func TestEvictionCallbackWithTTL(t *testing.T) {
	var mu sync.Mutex
	cache := New[int, int](10, time.Second)
	evicted := make(map[int]int)
	cache.SetOnEvicted(func(key int, value int, _ types.EvictReason) {
		mu.Lock()
		evicted[key] = value
		mu.Unlock()
	})

	// add objects to the cache
	for i := 1; i <= 10; i++ {
		cache.Set(i, i)
	}

	timeout := time.After(5 * time.Second)
	ticker := time.NewTicker(100 * time.Millisecond)
	for {
		select {
		case <-timeout:
			t.Fatal("timeout")
		case <-ticker.C:
			mu.Lock()
			if len(evicted) == 10 {
				for i := 1; i <= 10; i++ {
					assert.Equal(t, i, evicted[i])
				}
				return
			}
			mu.Unlock()
		}
	}
}
