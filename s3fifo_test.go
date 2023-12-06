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
