package auth

import (
	"fmt"
	"testing"
	"time"
)

func TestLockoutLocksAtThreshold(t *testing.T) {
	l := newLockout(3, time.Minute)
	if l.fail("k") || l.fail("k") {
		t.Fatal("locked too early")
	}
	if !l.fail("k") {
		t.Fatal("third failure should lock")
	}
	if !time.Now().Before(l.lockedUntil("k")) {
		t.Fatal("key should be locked")
	}
	l.succeed("k")
	if !l.lockedUntil("k").IsZero() {
		t.Fatal("succeed should clear the key")
	}
}

func TestLockoutStaysBounded(t *testing.T) {
	l := newLockout(5, time.Minute)
	for i := range lockoutMaxEntries + 50 {
		l.fail(fmt.Sprintf("spray-%d", i))
	}
	if len(l.entries) > lockoutMaxEntries {
		t.Fatalf("entries = %d, want <= %d", len(l.entries), lockoutMaxEntries)
	}
}

func TestLockoutPruneKeepsActiveLocks(t *testing.T) {
	l := newLockout(1, time.Hour) // long window: slow machines must not expire it
	l.fail("victim")              // locks immediately
	for i := range lockoutMaxEntries + 10 {
		l.fail(fmt.Sprintf("spray-%d", i))
	}
	if !time.Now().Before(l.lockedUntil("victim")) {
		t.Fatal("prune must not release an active lock")
	}
}
