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
