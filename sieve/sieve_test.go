package sieve

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAndSetOnSieve(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	cache := New[int, int](10)

	for _, v := range items {
		cache.Set(v, v*10)
	}

	for _, v := range items {
		val, ok := cache.Get(v)
		require.True(t, ok)
		require.Equal(t, v*10, val)
	}
}

func TestContainsOnSieve(t *testing.T) {
	cache := New[string, string](10)
	require.False(t, cache.Contains("hello"))

	cache.Set("hello", "world")
	require.True(t, cache.Contains("hello"))
}

func TestLenOnSieve(t *testing.T) {
	cache := New[int, int](10)
	require.Equal(t, 0, cache.Len())

	cache.Set(1, 1)
	require.Equal(t, 1, cache.Len())

	// duplicated keys only update the recent-ness of the key and value
	cache.Set(1, 1)
	require.Equal(t, 1, cache.Len())

	cache.Set(2, 2)
	require.Equal(t, 2, cache.Len())
}
