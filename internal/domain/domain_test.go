package domain

import (
	"errors"
	"slices"
	"testing"
	"time"
)

func TestApplyLifecycle(t *testing.T) {
	tr := Transfer{Status: TransferRequested}

	if _, err := Apply(&tr, Listing{}, Step{Action: "collect", ConfirmedKg: 5}); !errors.Is(err, ErrConflict) {
		t.Fatalf("collect before accept: got %v, want conflict", err)
	}

	ls, err := Apply(&tr, Listing{}, Step{Action: "accept"})
	if err != nil || ls != ListingReserved || tr.Status != TransferAccepted {
		t.Fatalf("accept: status=%s listing=%s err=%v", tr.Status, ls, err)
	}

	if _, err := Apply(&tr, Listing{}, Step{Action: "collect", ConfirmedKg: 0}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("collect without weight: got %v, want invalid", err)
	}
	if _, err := Apply(&tr, Listing{}, Step{Action: "collect", ConfirmedKg: 4.5}); err != nil {
		t.Fatalf("collect: %v", err)
	}
	if tr.ConfirmedKg != 4.5 {
		t.Fatalf("collect did not record the weight: %+v", tr)
	}

	ls, err = Apply(&tr, Listing{}, Step{Action: "outcome", Outcome: "reused", OutcomeNote: "  shelves  "})
	if err != nil || ls != ListingClosed || tr.Status != TransferCompleted || tr.OutcomeNote != "shelves" {
		t.Fatalf("outcome: %+v listing=%s err=%v", tr, ls, err)
	}
}

func TestFailedCollectionReopensListing(t *testing.T) {
	tr := Transfer{Status: TransferAccepted}
	ls, err := Apply(&tr, Listing{}, Step{Action: "fail"})
	if err != nil || ls != ListingOpen || tr.Status != TransferFailed {
		t.Fatalf("fail: status=%s listing=%s err=%v", tr.Status, ls, err)
	}
}

func TestSpecialHandlingIsRecyclersOnly(t *testing.T) {
	l := Listing{Status: ListingOpen, Route: RouteSpecial}
	if err := CanRequest(l, RoleIndividual); !errors.Is(err, ErrInvalid) {
		t.Fatalf("individual on special route: got %v, want invalid", err)
	}
	if err := CanRequest(l, RoleRecycler); err != nil {
		t.Fatalf("recycler on special route: %v", err)
	}
	l.Status = ListingReserved
	if err := CanRequest(l, RoleRecycler); !errors.Is(err, ErrConflict) {
		t.Fatalf("reserved listing: got %v, want conflict", err)
	}
}

func TestRankPrefersMaterialQuantityAndArea(t *testing.T) {
	now := time.Now()
	listings := []Listing{
		{ID: "far", Material: "wood", EstKg: 30, Route: RouteReuse, Area: "Matero", Status: ListingOpen, CreatedAt: now},
		{ID: "near", Material: "wood", EstKg: 30, Route: RouteReuse, Area: "Kabwata", Status: ListingOpen, CreatedAt: now},
		{ID: "small", Material: "wood", EstKg: 5, Route: RouteReuse, Area: "Kabwata", Status: ListingOpen, CreatedAt: now},
		{ID: "metal", Material: "metals", EstKg: 30, Route: RouteReuse, Area: "Kabwata", Status: ListingOpen, CreatedAt: now},
		{ID: "special", Material: "wood", EstKg: 30, Route: RouteSpecial, Area: "Kabwata", Status: ListingOpen, CreatedAt: now},
		{ID: "taken", Material: "wood", EstKg: 30, Route: RouteReuse, Area: "Kabwata", Status: ListingReserved, CreatedAt: now},
	}
	got := Rank(listings, Need{Material: "wood", Kg: 20, Area: "Kabwata", Role: RoleIndividual})
	var ids []string
	for _, m := range got {
		ids = append(ids, m.Listing.ID)
	}
	want := []string{"near", "far", "small"}
	if len(ids) != len(want) {
		t.Fatalf("ranked %v, want %v", ids, want)
	}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("ranked %v, want %v", ids, want)
		}
	}
	if got[0].Score != 100 {
		t.Fatalf("perfect match scored %d, want 100", got[0].Score)
	}
}

func TestTotalsCountOnlyCompletedHandOvers(t *testing.T) {
	listings := map[string]Listing{
		"w": {ID: "w", OwnerID: "ann", Material: "wood"},
		"m": {ID: "m", OwnerID: "bo", Material: "electronics", Pieces: 12},
	}
	transfers := []Transfer{
		{ListingID: "w", ReceiverID: "bo", Status: TransferCompleted, ConfirmedKg: 10, Outcome: "reused"},
		{ListingID: "m", ReceiverID: "ann", Status: TransferCompleted, ConfirmedPieces: 4, Outcome: "reused"},
		{ListingID: "w", ReceiverID: "cy", Status: TransferCollected, ConfirmedKg: 50},
		{ListingID: "w", ReceiverID: "cy", Status: TransferFailed},
	}
	all := ComputeTotals(transfers, listings, "")
	if all.HandedOn != (Amount{Kg: 10, Pieces: 4, HandOvers: 2}) || all.InProgress != 1 {
		t.Fatalf("community totals: %+v", all)
	}
	ann := ComputeTotals(transfers, listings, "ann")
	if ann.Given != (Amount{Kg: 10, HandOvers: 1}) || ann.Claimed != (Amount{Pieces: 4, HandOvers: 1}) {
		t.Fatalf("ann gave %+v and claimed %+v", ann.Given, ann.Claimed)
	}
}

func TestCollectInTheListingsUnit(t *testing.T) {
	l := Listing{Pieces: 10}
	tr := Transfer{Status: TransferAccepted}
	if _, err := Apply(&tr, l, Step{Action: "collect"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("collect without a count: got %v, want invalid", err)
	}
	if _, err := Apply(&tr, l, Step{Action: "collect", Pieces: 11}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("collect more pieces than listed: got %v, want invalid", err)
	}
	if _, err := Apply(&tr, l, Step{Action: "collect", Pieces: 4}); err != nil {
		t.Fatal(err)
	}
	if tr.ConfirmedPieces != 4 || tr.ConfirmedKg != 0 {
		t.Fatalf("counted collect: %+v", tr)
	}
}

func TestValidateListingUnits(t *testing.T) {
	base := Listing{ProviderName: "A", Item: "Monitors", Material: "electronics", Route: RouteReuse, Area: "Kabwata"}
	counted := base
	counted.Pieces, counted.EstKg = 12, 50
	if err := ValidateListing(&counted); err != nil || counted.EstKg != 0 {
		t.Fatalf("counted listing: %+v %v", counted, err)
	}
	if err := ValidateListing(&base); !errors.Is(err, ErrInvalid) {
		t.Fatalf("listing with no amount: got %v, want invalid", err)
	}
}

func TestRankByPieces(t *testing.T) {
	now := time.Now()
	ls := []Listing{
		{ID: "few", Material: "electronics", Pieces: 3, Route: RouteReuse, Area: "Kabwata", Status: ListingOpen, CreatedAt: now},
		{ID: "many", Material: "electronics", Pieces: 12, Route: RouteReuse, Area: "Kabwata", Status: ListingOpen, CreatedAt: now.Add(-time.Hour)},
		{ID: "bulk", Material: "electronics", EstKg: 80, Route: RouteReuse, Area: "Kabwata", Status: ListingOpen, CreatedAt: now.Add(-2 * time.Hour)},
	}
	got := Rank(ls, Need{Material: "electronics", Pieces: 10, Area: "Kabwata"})
	if len(got) != 3 || got[0].Listing.ID != "many" {
		t.Fatalf("ranking by pieces: %+v", got)
	}
	for _, m := range got {
		if m.Listing.ID == "bulk" && !slices.Contains(m.Reasons, "sold by weight") {
			t.Fatalf("weight-only listing should say so: %+v", m.Reasons)
		}
	}
}

func TestValidateSignup(t *testing.T) {
	ok := Signup{Name: " Chanda ", Email: " Chanda@Example.com ", Password: "longenough", Role: RoleIndividual, Area: "Kabwata"}
	if err := ValidateSignup(&ok); err != nil || ok.Email != "chanda@example.com" || ok.Name != "Chanda" {
		t.Fatalf("valid signup: %+v %v", ok, err)
	}
	for name, s := range map[string]Signup{
		"short password": {Name: "A", Email: "a@b.co", Password: "short", Role: RoleIndividual, Area: "Kabwata"},
		"bad email":      {Name: "A", Email: "not-an-email", Password: "longenough", Role: RoleIndividual, Area: "Kabwata"},
		"unknown role":   {Name: "A", Email: "a@b.co", Password: "longenough", Role: "admin", Area: "Kabwata"},
		"unknown area":   {Name: "A", Email: "a@b.co", Password: "longenough", Role: RoleIndividual, Area: "Atlantis"},
	} {
		if err := ValidateSignup(&s); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: got %v, want invalid", name, err)
		}
	}
}

func TestValidateNeed(t *testing.T) {
	if err := ValidateNeed(&SavedNeed{Material: "wood", Kg: 20}); err != nil {
		t.Fatal(err)
	}
	if err := ValidateNeed(&SavedNeed{Kg: 20}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("need without material: got %v, want invalid", err)
	}
	if err := ValidateNeed(&SavedNeed{Material: "wood", Kg: 5, Pieces: 2}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("need in kg and pieces: got %v, want invalid", err)
	}
}

func TestRecordsListCompletedHandOvers(t *testing.T) {
	listings := map[string]Listing{
		"w": {ID: "w", Material: "wood", Item: "Offcuts", Area: "Kabwata"},
		"e": {ID: "e", Material: "electronics", Item: "Monitors", Area: "Chilenje", Pieces: 12},
	}
	transfers := []Transfer{
		{ID: "1", ListingID: "w", Status: TransferCompleted, ConfirmedKg: 10, Outcome: "reused"},
		{ID: "2", ListingID: "e", Status: TransferCompleted, ConfirmedPieces: 10, Outcome: "repaired"},
		{ID: "3", ListingID: "w", Status: TransferCollected, ConfirmedKg: 50},
	}
	recs := Records(transfers, listings)
	if len(recs) != 2 {
		t.Fatalf("got %d records, want only the 2 completed", len(recs))
	}
	if recs[1].Pieces != 10 || recs[1].Outcome != "Repaired or repurposed" {
		t.Fatalf("monitor record: %+v", recs[1])
	}
}
