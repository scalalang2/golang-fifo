package internal

type Entry[K comparable, V any] struct {
	Key  K
	Val  V
	Freq byte
}
