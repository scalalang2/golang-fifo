package s3fifo

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPushAndPopRingBuf(t *testing.T) {
	r := newRingBuf[int](10)
	for i := 0; i < 10; i++ {
		r.push(i)
	}

	for i := 0; i < 10; i++ {
		v := r.pop()
		require.Equal(t, i, v)
	}
}

func TestPushAndPopRingBufV2(t *testing.T) {
	r := newRingBuf[int](10)
	for i := 0; i < 10; i++ {
		r.push(i)
	}

	for i := 0; i < 5; i++ {
		v := r.pop()
		require.Equal(t, i, v)
	}

	for i := 10; i < 15; i++ {
		r.push(i)
	}

	for i := 5; i < 15; i++ {
		v := r.pop()
		require.Equal(t, i, v)
	}
}

func TestIsFullRingBuf(t *testing.T) {
	r := newRingBuf[int](10)
	for i := 0; i < 10; i++ {
		r.push(i)
	}

	require.True(t, r.full)
	require.False(t, r.push(10))
}
