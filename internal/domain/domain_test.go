package domain

import (
	"errors"
	"math"
	"slices"
	"testing"
	"time"
)

func TestApplyLifecycle(t *testing.T) {
	tr := Transfer{Status: TransferRequested}

	if _, err := Apply(&tr, Listing{}, Step{Action: "collect", ConfirmedKg: 5, Counterfactual: "landfill"}); !errors.Is(err, ErrConflict) {
		t.Fatalf("collect before accept: got %v, want conflict", err)
	}

	ls, err := Apply(&tr, Listing{}, Step{Action: "accept"})
	if err != nil || ls != ListingReserved || tr.Status != TransferAccepted {
		t.Fatalf("accept: status=%s listing=%s err=%v", tr.Status, ls, err)
	}

	if _, err := Apply(&tr, Listing{}, Step{Action: "collect", ConfirmedKg: 0, Counterfactual: "landfill"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("collect without weight: got %v, want invalid", err)
	}
	if _, err := Apply(&tr, Listing{}, Step{Action: "collect", ConfirmedKg: 4, Counterfactual: "maybe"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("collect with unknown counterfactual: got %v, want invalid", err)
	}
	if _, err := Apply(&tr, Listing{}, Step{Action: "collect", ConfirmedKg: 4.5, Counterfactual: "landfill"}); err != nil {
		t.Fatalf("collect: %v", err)
	}
	if tr.ConfirmedKg != 4.5 || tr.Counterfactual != "landfill" {
		t.Fatalf("collect did not record weight and counterfactual: %+v", tr)
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

func TestImpactCountsOnlyDivertedCompletedWeight(t *testing.T) {
	listings := map[string]Listing{
		"w": {ID: "w", Material: "wood"},
		"p": {ID: "p", Material: "paper"},
	}
	transfers := []Transfer{
		{ListingID: "w", Status: TransferCompleted, ConfirmedKg: 10, Counterfactual: "dumped_burned", Outcome: "reused"},
		{ListingID: "p", Status: TransferCompleted, ConfirmedKg: 8, Counterfactual: "storage", Outcome: "reused"},
		{ListingID: "w", Status: TransferCollected, ConfirmedKg: 50, Counterfactual: "landfill"},
		{ListingID: "w", Status: TransferFailed},
	}
	imp := ComputeImpact(transfers, listings)
	if imp.Completed != 2 || imp.Failed != 1 || imp.InProgress != 1 {
		t.Fatalf("counts: %+v", imp)
	}
	if imp.HandedKg != 18 || imp.DivertedKg != 10 {
		t.Fatalf("handed=%v diverted=%v, want 18 and 10", imp.HandedKg, imp.DivertedKg)
	}
	wood, _ := MaterialByID("wood")
	if math.Abs(imp.KgCO2e-10*wood.FactorKgCO2e) > 1e-9 {
		t.Fatalf("kgCO2e=%v", imp.KgCO2e)
	}
	if math.Abs(imp.CompletionRate-2.0/3.0) > 1e-9 {
		t.Fatalf("completion rate=%v", imp.CompletionRate)
	}
}

func TestCollectCountedItems(t *testing.T) {
	l := Listing{Pieces: 10, EstKg: 45}
	tr := Transfer{Status: TransferAccepted}
	if _, err := Apply(&tr, l, Step{Action: "collect", Counterfactual: "storage"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("collect without a count: got %v, want invalid", err)
	}
	if _, err := Apply(&tr, l, Step{Action: "collect", Pieces: 11, Counterfactual: "storage"}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("collect more pieces than listed: got %v, want invalid", err)
	}
	// No weight: estimated from 4.5 kg per piece.
	if _, err := Apply(&tr, l, Step{Action: "collect", Pieces: 4, Counterfactual: "landfill"}); err != nil {
		t.Fatal(err)
	}
	if tr.ConfirmedPieces != 4 || tr.ConfirmedKg != 18 || !tr.KgEstimated {
		t.Fatalf("estimated collect: %+v", tr)
	}
	// A weight from the scale wins over the estimate.
	tr = Transfer{Status: TransferAccepted}
	if _, err := Apply(&tr, l, Step{Action: "collect", Pieces: 4, ConfirmedKg: 16.2, Counterfactual: "landfill"}); err != nil {
		t.Fatal(err)
	}
	if tr.ConfirmedKg != 16.2 || tr.KgEstimated {
		t.Fatalf("weighed collect: %+v", tr)
	}
}

func TestRankByPieces(t *testing.T) {
	now := time.Now()
	ls := []Listing{
		{ID: "few", Material: "electronics", Pieces: 3, EstKg: 13, Route: RouteReuse, Area: "Kabwata", Status: ListingOpen, CreatedAt: now},
		{ID: "many", Material: "electronics", Pieces: 12, EstKg: 54, Route: RouteReuse, Area: "Kabwata", Status: ListingOpen, CreatedAt: now.Add(-time.Hour)},
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
