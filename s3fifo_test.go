package fifo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSetAndGetOnCache(t *testing.T) {
	cache := NewS3FIFO[string, string](10)
	err := cache.Set("hello", "world")
	require.NoError(t, err)

	value, err := cache.Get("hello")
	require.NoError(t, err)
	require.Equal(t, "world", value)
}

func TestEvictOneHitWonders(t *testing.T) {
	cache := NewS3FIFO[int, int](10)
	oneHitWonders := []int{1, 2}
	popularObjects := []int{3, 4, 5, 6, 7, 8, 9, 10}

	// add objects to the cache
	for _, v := range oneHitWonders {
		require.NoError(t, cache.Set(v, v))
	}
	for _, v := range popularObjects {
		require.NoError(t, cache.Set(v, v))
	}

	// hit one-hit wonders only once
	for _, v := range oneHitWonders {
		_, err := cache.Get(v)
		require.NoError(t, err)
	}

	// hit the popular objects
	for i := 0; i < 3; i++ {
		for _, v := range popularObjects {
			_, err := cache.Get(v)
			require.NoError(t, err)
		}
	}

	// add more objects to the cache
	// this should evict the one hit wonders
	for i := 11; i < 20; i++ {
		require.NoError(t, cache.Set(i, i))
	}

	for _, v := range oneHitWonders {
		_, err := cache.Get(v)
		require.Error(t, err, ErrKeyNotFound)
	}

	// popular objects should still be in the cache
	for _, v := range popularObjects {
		_, err := cache.Get(v)
		require.NoError(t, err)
	}
}

func TestPeekOnCache(t *testing.T) {
	cache := NewS3FIFO[int, int](5)
	entries := []int{1, 2, 3, 4, 5}

	for _, v := range entries {
		require.NoError(t, cache.Set(v, v*10))
	}

	// peek each entry 10 times
	// this should not change the recent-ness of the entry.
	for i := 0; i < 10; i++ {
		for _, v := range entries {
			value, exist, err := cache.Peek(v)
			require.NoError(t, err)
			require.True(t, exist)
			require.Equal(t, v*10, value)
		}
	}

	// add more entries to the cache
	// this should evict the first 5 entries
	for i := 6; i <= 10; i++ {
		require.NoError(t, cache.Set(i, i*10))
	}

	// peek the first 5 entries
	// they should not exist in the cache
	for _, v := range entries {
		_, exist, err := cache.Peek(v)
		require.NoError(t, err)
		require.False(t, exist)
	}
}

func TestContainsOnCache(t *testing.T) {
	cache := NewS3FIFO[int, int](5)
	entries := []int{1, 2, 3, 4, 5}

	for _, v := range entries {
		require.NoError(t, cache.Set(v, v*10))
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
	cache := NewS3FIFO[string, string](10)

	require.NoError(t, cache.Set("hello", "world"))
	require.Equal(t, 1, cache.Len())

	require.NoError(t, cache.Set("hello2", "world"))
	require.NoError(t, cache.Set("hello", "changed"))
	require.Equal(t, 3, cache.Len())

	value, err := cache.Get("hello")
	require.NoError(t, err)
	require.Equal(t, "changed", value)
}
