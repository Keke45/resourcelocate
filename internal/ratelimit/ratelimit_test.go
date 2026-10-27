package ratelimit

import (
	"testing"
	"time"
)

func TestBurstThenRefill(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(3, 10*time.Second)
	l.now = func() time.Time { return now }
	for i := 0; i < 3; i++ {
		if ok, _ := l.Allow("a"); !ok {
			t.Fatalf("request %d refused inside the burst", i+1)
		}
	}
	ok, wait := l.Allow("a")
	if ok || wait != 10*time.Second {
		t.Fatalf("4th request: ok=%v wait=%v, want refused for 10s", ok, wait)
	}
	if ok, _ := l.Allow("b"); !ok {
		t.Fatal("another key shares the bucket")
	}
	now = now.Add(10 * time.Second)
	if ok, _ := l.Allow("a"); !ok {
		t.Fatal("no token after refill")
	}
	if ok, _ := l.Allow("a"); ok {
		t.Fatal("refill gave more than one token")
	}
}

func TestSweepDropsIdleKeys(t *testing.T) {
	now := time.Unix(0, 0)
	l := New(2, time.Second)
	l.now = func() time.Time { return now }
	l.Allow("a")
	now = now.Add(2 * time.Minute)
	l.Allow("b")
	if _, ok := l.buckets["a"]; ok || len(l.buckets) != 1 {
		t.Fatalf("buckets after sweep: %v", l.buckets)
	}
}
