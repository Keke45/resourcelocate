package store

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/KeKe45/resourcelocate/internal/domain"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func newUser(t *testing.T, s *Store, name string) domain.User {
	t.Helper()
	u := domain.User{Name: name, Email: name + "@example.com", Role: domain.RoleIndividual, Area: "Kabwata"}
	if err := s.CreateUser(context.Background(), &u, "disabled"); err != nil {
		t.Fatal(err)
	}
	return u
}

func TestHandOverRulesAndPermissions(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	giver, a, b := newUser(t, s, "giver"), newUser(t, s, "a"), newUser(t, s, "b")

	l := domain.Listing{OwnerID: giver.ID, ProviderName: giver.Name, Item: "Offcuts", Material: "wood", EstKg: 10, Route: domain.RouteReuse, Area: "Kabwata"}
	if err := s.CreateListing(ctx, &l); err != nil {
		t.Fatal(err)
	}
	own := domain.Transfer{ListingID: l.ID, ReceiverID: giver.ID, ReceiverRole: domain.RoleIndividual}
	if err := s.RequestListing(ctx, &own); !errors.Is(err, domain.ErrInvalid) {
		t.Fatalf("requesting own listing: got %v, want invalid", err)
	}
	ta := domain.Transfer{ListingID: l.ID, ReceiverID: a.ID, ReceiverName: "A", ReceiverRole: domain.RoleIndividual}
	tb := domain.Transfer{ListingID: l.ID, ReceiverID: b.ID, ReceiverName: "B", ReceiverRole: domain.RoleIndividual}
	if err := s.RequestListing(ctx, &ta); err != nil {
		t.Fatal(err)
	}
	again := ta
	if err := s.RequestListing(ctx, &again); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("second request by the same user: got %v, want conflict", err)
	}
	if err := s.RequestListing(ctx, &tb); err != nil {
		t.Fatal(err)
	}

	if _, err := s.Advance(ctx, a.ID, ta.ID, domain.Step{Action: "accept"}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("receiver accepting own request: got %v, want forbidden", err)
	}
	if _, err := s.Advance(ctx, giver.ID, ta.ID, domain.Step{Action: "accept"}); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Listing(ctx, l.ID)
	if got.Status != domain.ListingReserved {
		t.Fatalf("listing status %s, want reserved", got.Status)
	}
	bt, _ := s.TransfersFor(ctx, b.ID)
	if len(bt) != 1 || bt[0].Status != domain.TransferDeclined {
		t.Fatalf("other request: %+v, want one declined", bt)
	}

	if _, err := s.Advance(ctx, giver.ID, ta.ID, domain.Step{Action: "collect", ConfirmedKg: 9}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Advance(ctx, giver.ID, ta.ID, domain.Step{Action: "outcome", Outcome: "reused"}); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("giver recording outcome: got %v, want forbidden", err)
	}
	if _, err := s.Advance(ctx, a.ID, ta.ID, domain.Step{Action: "outcome", Outcome: "reused"}); err != nil {
		t.Fatal(err)
	}
	got, _ = s.Listing(ctx, l.ID)
	if got.Status != domain.ListingClosed {
		t.Fatalf("listing status %s, want closed", got.Status)
	}
	if mine, _ := s.TransfersFor(ctx, giver.ID); len(mine) != 2 {
		t.Fatalf("giver sees %d transfers, want 2", len(mine))
	}
}

func TestWithdrawListing(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	giver, other := newUser(t, s, "giver"), newUser(t, s, "other")
	l := domain.Listing{OwnerID: giver.ID, ProviderName: giver.Name, Item: "Desks", Material: "furniture", Pieces: 4, EstKg: 60, Route: domain.RouteRepair, Area: "Chilenje"}
	if err := s.CreateListing(ctx, &l); err != nil {
		t.Fatal(err)
	}
	req := domain.Transfer{ListingID: l.ID, ReceiverID: other.ID, ReceiverName: "Other", ReceiverRole: domain.RoleIndividual}
	if err := s.RequestListing(ctx, &req); err != nil {
		t.Fatal(err)
	}
	if err := s.WithdrawListing(ctx, other.ID, l.ID); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("withdraw by someone else: got %v, want forbidden", err)
	}
	if err := s.WithdrawListing(ctx, giver.ID, l.ID); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Listing(ctx, l.ID)
	ts, _ := s.TransfersFor(ctx, other.ID)
	if got.Status != domain.ListingWithdrawn || ts[0].Status != domain.TransferDeclined {
		t.Fatalf("after withdraw: listing %s, request %s", got.Status, ts[0].Status)
	}
}

func TestNeedsPersistUntilClosed(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	u, other := newUser(t, s, "seeker"), newUser(t, s, "other")
	n := domain.SavedNeed{UserID: u.ID, Material: "electronics", Pieces: 10}
	if err := s.CreateNeed(ctx, &n); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.NeedsBy(ctx, u.ID); len(got) != 1 || got[0].Pieces != 10 {
		t.Fatalf("needs after save: %+v", got)
	}
	if err := s.CloseNeed(ctx, other.ID, n.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("closing someone else's need: got %v, want not found", err)
	}
	if err := s.CloseNeed(ctx, u.ID, n.ID); err != nil {
		t.Fatal(err)
	}
	if got, _ := s.NeedsBy(ctx, u.ID); len(got) != 0 {
		t.Fatalf("closed need still listed: %+v", got)
	}
}

func TestSessions(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	u := newUser(t, s, "user")
	dup := domain.User{Name: "Dup", Email: u.Email, Role: domain.RoleIndividual, Area: "Kabwata"}
	if err := s.CreateUser(ctx, &dup, "disabled"); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("duplicate email: got %v, want conflict", err)
	}
	token, _, err := s.CreateSession(ctx, u.ID)
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.UserForSession(ctx, token)
	if err != nil || got.ID != u.ID {
		t.Fatalf("session lookup: %+v %v", got, err)
	}
	if err := s.DeleteSession(ctx, token); err != nil {
		t.Fatal(err)
	}
	if _, err := s.UserForSession(ctx, token); !errors.Is(err, ErrNotFound) {
		t.Fatalf("after logout: got %v, want not found", err)
	}
}

func TestListingPhotos(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	giver, other := newUser(t, s, "giver"), newUser(t, s, "other")
	l := domain.Listing{OwnerID: giver.ID, ProviderName: giver.Name, Item: "Monitors", Material: "electronics", Pieces: 2, Route: domain.RouteReuse, Area: "Kabwata"}
	if err := s.CreateListing(ctx, &l); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddPhoto(ctx, other.ID, l.ID, "image/jpeg", []byte("x")); !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("photo by someone else: got %v, want forbidden", err)
	}
	for i := 0; i < domain.MaxPhotos; i++ {
		n, err := s.AddPhoto(ctx, giver.ID, l.ID, "image/jpeg", []byte{byte('a' + i)})
		if err != nil || n != i {
			t.Fatalf("photo %d: n=%d err=%v", i, n, err)
		}
	}
	if _, err := s.AddPhoto(ctx, giver.ID, l.ID, "image/jpeg", []byte("z")); !errors.Is(err, domain.ErrConflict) {
		t.Fatalf("photo over the limit: got %v, want conflict", err)
	}
	got, _ := s.Listing(ctx, l.ID)
	ct, data, err := s.Photo(ctx, l.ID, 2)
	if got.Photos != domain.MaxPhotos || ct != "image/jpeg" || string(data) != "c" || err != nil {
		t.Fatalf("photo round trip: photos=%d ct=%s data=%q err=%v", got.Photos, ct, data, err)
	}
}

func TestChangePasswordEndsOtherSessions(t *testing.T) {
	ctx := context.Background()
	s := openTemp(t)
	u := newUser(t, s, "user")
	here, _, _ := s.CreateSession(ctx, u.ID)
	phone, _, _ := s.CreateSession(ctx, u.ID)
	if err := s.SetPassword(ctx, u.ID, "new-hash", here); err != nil {
		t.Fatal(err)
	}
	if h, _ := s.PasswordHash(ctx, u.ID); h != "new-hash" {
		t.Fatalf("hash after change: %q", h)
	}
	if _, err := s.UserForSession(ctx, here); err != nil {
		t.Fatalf("current session ended: %v", err)
	}
	if _, err := s.UserForSession(ctx, phone); !errors.Is(err, ErrNotFound) {
		t.Fatalf("other session: got %v, want not found", err)
	}
}
