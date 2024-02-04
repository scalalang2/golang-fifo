package internal

// Entry is an element of the cache.
type Entry[K comparable, V any] struct {
	// next and prev pointers in the doubly linked list of entries.
	next, prev *Entry[K, V]

	// the List which this entry belongs to.
	List *List[K, V]

	// The Key of the element.
	Key K

	// Value is a value stored in the element.
	Value V

	// Frequency is the number of times the element has been accessed.
	Frequency int

	// Expire is the time when the element will be removed from the cache.
	Expire int64
}
