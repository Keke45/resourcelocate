package api

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/KeKe45/resourcelocate/internal/domain"
	"github.com/KeKe45/resourcelocate/internal/store"
)

// The demo activity loads once, gives both demo accounts something waiting
// on them, and is not added again on the next start.
func TestDemoActivity(t *testing.T) {
	ctx := context.Background()
	st, err := store.Open(filepath.Join(t.TempDir(), "demo.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	for range 2 {
		s := New(st, false)
		if err := s.EnableDemoLogins(ctx); err != nil {
			t.Fatal(err)
		}
	}
	ls, _ := st.Listings(ctx)
	if len(ls) != len(demoListings) {
		t.Fatalf("%d listings after two starts, want %d", len(ls), len(demoListings))
	}
	ts, _ := st.Transfers(ctx)
	if len(ts) != len(demoTransfers) {
		t.Fatalf("%d hand-overs, want %d", len(ts), len(demoTransfers))
	}

	s := New(st, false)
	if err := s.EnableDemoLogins(ctx); err != nil {
		t.Fatal(err)
	}
	m := byID(ls)
	for _, key := range []string{"giver", "seeker"} {
		u := s.demo[key]
		mine, _ := st.TransfersFor(ctx, u.ID)
		todo := 0
		for _, t := range mine {
			giving := m[t.ListingID].OwnerID == u.ID
			if giving && (t.Status == domain.TransferRequested || t.Status == domain.TransferAccepted) ||
				!giving && t.Status == domain.TransferCollected {
				todo++
			}
		}
		needs, _ := s.needViews(ctx, u)
		matched := 0
		for _, n := range needs {
			matched += len(n.Matches)
		}
		if todo == 0 || len(needs) == 0 || matched == 0 {
			t.Errorf("%s: %d waiting on them, %d needs, %d matches; want some of each", key, todo, len(needs), matched)
		}
	}
}
