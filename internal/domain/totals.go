package domain

import "sort"

// Amount is how much changed hands: weight for bulk material, pieces for
// counted items, and the number of finished hand-overs.
type Amount struct {
	Kg        float64 `json:"kg"`
	Pieces    int     `json:"pieces"`
	HandOvers int     `json:"handOvers"`
}

// add counts transfer t of listing l in the listing's own unit. Hand-overs
// recorded by older versions could carry an estimated weight for counted
// items too; that estimate is left out.
func (a *Amount) add(t Transfer, l Listing) {
	if l.Pieces > 0 {
		a.Pieces += t.ConfirmedPieces
	} else {
		a.Kg += t.ConfirmedKg
	}
	a.HandOvers++
}

type MaterialTotal struct {
	Material string `json:"material"`
	Label    string `json:"label"`
	Amount
}

// Totals is the impact we show: what was actually handed on, nothing more.
// Only completed hand-overs count, so a listing alone never adds to it.
type Totals struct {
	HandedOn Amount `json:"handedOn"`
	// Given and Claimed split HandedOn by side for one user's dashboard:
	// what they gave away and what they received. Both are zero in
	// community totals.
	Given      Amount          `json:"given"`
	Claimed    Amount          `json:"claimed"`
	InProgress int             `json:"inProgress"`
	ByMaterial []MaterialTotal `json:"byMaterial"`
}

// ComputeTotals adds up completed transfers. When userID is set, it also
// splits them into given (userID owns the listing) and claimed (userID is
// the receiver).
func ComputeTotals(transfers []Transfer, listings map[string]Listing, userID string) Totals {
	var tot Totals
	byMat := map[string]*MaterialTotal{}
	for _, t := range transfers {
		switch t.Status {
		case TransferRequested, TransferAccepted, TransferCollected:
			tot.InProgress++
			continue
		case TransferCompleted:
		default:
			continue
		}
		l := listings[t.ListingID]
		tot.HandedOn.add(t, l)
		if userID != "" && l.OwnerID == userID {
			tot.Given.add(t, l)
		}
		if userID != "" && t.ReceiverID == userID {
			tot.Claimed.add(t, l)
		}
		mat, ok := MaterialByID(l.Material)
		if !ok {
			mat, _ = MaterialByID("other")
		}
		mt := byMat[mat.ID]
		if mt == nil {
			mt = &MaterialTotal{Material: mat.ID, Label: mat.Label}
			byMat[mat.ID] = mt
		}
		mt.add(t, l)
	}
	tot.ByMaterial = []MaterialTotal{}
	for _, mt := range byMat {
		tot.ByMaterial = append(tot.ByMaterial, *mt)
	}
	sort.Slice(tot.ByMaterial, func(i, j int) bool {
		return tot.ByMaterial[i].HandOvers > tot.ByMaterial[j].HandOvers
	})
	return tot
}
