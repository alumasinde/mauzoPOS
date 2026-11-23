package cache

import (
	"errors"
	"testing"
	"time"
)

func TestTTLExpiresAndInvalidates(t *testing.T) {
	c := New[int, string](time.Minute, 10)
	now := time.Now()
	c.now = func() time.Time { return now }

	c.Set(1, "a")
	if v, ok := c.Get(1); !ok || v != "a" {
		t.Fatal("expected hit")
	}
	now = now.Add(2 * time.Minute)
	if _, ok := c.Get(1); ok {
		t.Fatal("expected expiry")
	}
	c.Set(2, "b")
	c.Delete(2)
	if _, ok := c.Get(2); ok {
		t.Fatal("expected delete")
	}
}

func TestTTLBounded(t *testing.T) {
	c := New[int, int](time.Hour, 5)
	for i := range 20 {
		c.Set(i, i)
	}
	if len(c.entries) > 5 {
		t.Fatalf("entries = %d", len(c.entries))
	}
}

func TestGetOrLoadDoesNotCacheErrors(t *testing.T) {
	c := New[string, int](time.Hour, 5)
	calls := 0
	boom := errors.New("boom")
	load := func() (int, error) { calls++; return 0, boom }
	_, _ = c.GetOrLoad("k", load)
	_, _ = c.GetOrLoad("k", load)
	if calls != 2 {
		t.Fatalf("errors were cached: calls=%d", calls)
	}
	v, err := c.GetOrLoad("k", func() (int, error) { return 7, nil })
	if err != nil || v != 7 {
		t.Fatal("load failed")
	}
	if v, _ := c.GetOrLoad("k", load); v != 7 {
		t.Fatal("value not cached")
	}
}
