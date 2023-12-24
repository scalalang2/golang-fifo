package s3fifo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetAndGetOnCache(t *testing.T) {
	cache := New[string, string](10)
	cache.Set("hello", "world")

	value, ok := cache.Get("hello")
	require.True(t, ok)
	require.Equal(t, "world", value)
}

func TestEvictOneHitWonders(t *testing.T) {
	cache := New[int, int](10)
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
		require.True(t, ok)
	}

	// hit the popular objects
	for i := 0; i < 3; i++ {
		for _, v := range popularObjects {
			_, ok := cache.Get(v)
			require.True(t, ok)
		}
	}

	// add more objects to the cache
	// this should evict the one hit wonders
	for i := 11; i < 20; i++ {
		cache.Set(i, i)
	}

	for _, v := range oneHitWonders {
		_, ok := cache.Get(v)
		require.False(t, ok)
	}

	// popular objects should still be in the cache
	for _, v := range popularObjects {
		_, ok := cache.Get(v)
		require.True(t, ok)
	}
}

func TestPeekOnCache(t *testing.T) {
	cache := New[int, int](5)
	entries := []int{1, 2, 3, 4, 5}

	for _, v := range entries {
		cache.Set(v, v*10)
	}

	// peek each entry 10 times
	// this should not change the recent-ness of the entry.
	for i := 0; i < 10; i++ {
		for _, v := range entries {
			value, exist := cache.Peek(v)
			require.True(t, exist)
			require.Equal(t, v*10, value)
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
		require.False(t, exist)
	}
}

func TestContainsOnCache(t *testing.T) {
	cache := New[int, int](5)
	entries := []int{1, 2, 3, 4, 5}

	for _, v := range entries {
		cache.Set(v, v*10)
	}

	// check if each entry exists in the cache
	for _, v := range entries {
		require.True(t, cache.Contains(v))
	}

	for i := 6; i <= 10; i++ {
		require.False(t, cache.Contains(i))
	}
}

func TestLengthOnCache(t *testing.T) {
	cache := New[string, string](10)

	cache.Set("hello", "world")
	require.Equal(t, 1, cache.Len())

	cache.Set("hello2", "world")
	cache.Set("hello", "changed")
	require.Equal(t, 2, cache.Len())

	value, ok := cache.Get("hello")
	require.True(t, ok)
	require.Equal(t, "changed", value)
}

func TestCleanOnCache(t *testing.T) {
	cache := New[int, int](10)
	entries := []int{1, 2, 3, 4, 5}

	for _, v := range entries {
		cache.Set(v, v*10)
	}
	require.Equal(t, 5, cache.Len())
	cache.Purge()

	// check if each entry exists in the cache
	for _, v := range entries {
		_, exist := cache.Peek(v)
		require.False(t, exist)
	}
	require.Equal(t, 0, cache.Len())
}
