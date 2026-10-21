package sieve

import "testing"

func TestPeekDoesNotProtectFromEviction(t *testing.T) {
	cache := New[int, string](2, noEvictionTTL)
	defer cache.Close()

	cache.Set(1, "first")
	cache.Set(2, "second")
	for i := 0; i < 3; i++ {
		if value, ok := cache.Peek(1); !ok || value != "first" {
			t.Fatalf("Peek(1) = (%q, %v), want (first, true)", value, ok)
		}
	}
	if value, ok := cache.Peek(99); ok || value != "" {
		t.Fatalf("Peek(missing) = (%q, %v), want empty value and false", value, ok)
	}

	cache.Set(3, "third")
	if cache.Contains(1) {
		t.Fatal("Peek protected the oldest entry from eviction")
	}
	for key, want := range map[int]string{2: "second", 3: "third"} {
		if value, ok := cache.Peek(key); !ok || value != want {
			t.Fatalf("Peek(%d) = (%q, %v), want (%q, true)", key, value, ok, want)
		}
	}
	if cache.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", cache.Len())
	}
}

func TestEvictionAfterAllEntriesVisited(t *testing.T) {
	cache := New[int, int](2, noEvictionTTL)
	defer cache.Close()

	cache.Set(1, 10)
	cache.Set(2, 20)
	for _, key := range []int{1, 2} {
		if value, ok := cache.Get(key); !ok || value != key*10 {
			t.Fatalf("Get(%d) = (%d, %v)", key, value, ok)
		}
	}

	// The eviction hand must wrap around after clearing both visited bits.
	cache.Set(3, 30)
	if cache.Contains(1) || !cache.Contains(2) || !cache.Contains(3) {
		t.Fatal("expected entries 2 and 3 after the eviction hand wrapped")
	}
	// A subsequent eviction must continue from the saved hand position.
	cache.Set(4, 40)
	if cache.Contains(2) || !cache.Contains(3) || !cache.Contains(4) {
		t.Fatal("expected entries 3 and 4 after the next eviction")
	}
	if cache.Len() != 2 {
		t.Fatalf("Len() = %d, want 2", cache.Len())
	}
}
