package golang_fifo

import (
	"strconv"
	"testing"

	"github.com/scalalang2/golang-fifo/sieve"
)

type value struct {
	bytes []byte
}

type compositeKey struct {
	key1 string
	key2 string
}

type benchTypes interface {
	int32 | int64 | string | compositeKey
}

func BenchmarkCache(b *testing.B) {
	b.Run("cache=sieve", func(b *testing.B) {
		b.Run("t=int32", bench[int32](genKeysInt32))
		b.Run("t=int64", bench[int64](genKeysInt64))
		b.Run("t=string", bench[string](genKeysString))
		b.Run("t=composite", bench[compositeKey](genKeysComposite))
	})
}

func bench[T benchTypes](gen func(workload int) []T) func(b *testing.B) {
	cacheSize := 100000

	return func(b *testing.B) {
		benchmarkSieveCache[T](b, cacheSize, gen)
	}
}

func genKeysInt32(workload int) []int32 {
	keys := make([]int32, workload)
	for i := range keys {
		keys[i] = int32(i)
	}
	return keys
}

func genKeysInt64(workload int) []int64 {
	keys := make([]int64, workload)
	for i := range keys {
		keys[i] = int64(i)
	}
	return keys
}

func genKeysString(workload int) []string {
	keys := make([]string, workload)
	for i := range keys {
		keys[i] = strconv.Itoa(i)
	}
	return keys
}

func genKeysComposite(workload int) []compositeKey {
	keys := make([]compositeKey, workload)
	for i := range keys {
		keys[i].key1 = strconv.Itoa(i)
		keys[i].key2 = strconv.Itoa(i)
	}
	return keys
}

func benchmarkSieveCache[T benchTypes](b *testing.B, cacheSize int, genKey func(size int) []T) {
	cache := sieve.New[T, value](cacheSize, 0)
	keys := genKey(b.N)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := keys[i]
		cache.Set(key, value{
			bytes: make([]byte, 10),
		})
		cache.Get(key)
	}
	cache.Purge()
}
