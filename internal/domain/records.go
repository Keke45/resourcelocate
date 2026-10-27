package domain

import "sort"

// Record is one finished hand-over as published in the public records
// export. It leaves out names, contacts and free-text notes, so anyone can
// check the totals without seeing who took part.
type Record struct {
	ID           string
	CompletedOn  string // YYYY-MM-DD, when the receiver recorded the outcome
	Material     string
	Item         string
	Area         string
	ReceiverRole string
	Pieces       int
	Kg           float64
	Outcome      string
}

// Records lists completed hand-overs, oldest first, so the export adds up
// to the totals on the site.
func Records(transfers []Transfer, listings map[string]Listing) []Record {
	out := []Record{}
	for _, t := range transfers {
		if t.Status != TransferCompleted {
			continue
		}
		l := listings[t.ListingID]
		mat, ok := MaterialByID(l.Material)
		if !ok {
			mat, _ = MaterialByID("other")
		}
		r := Record{
			ID: t.ID, CompletedOn: t.UpdatedAt.UTC().Format("2006-01-02"),
			Material: mat.Label, Item: l.Item, Area: l.Area, ReceiverRole: t.ReceiverRole,
			Outcome: t.Outcome,
		}
		if l.Pieces > 0 {
			r.Pieces = t.ConfirmedPieces
		} else {
			r.Kg = t.ConfirmedKg
		}
		for _, o := range Outcomes {
			if o.ID == t.Outcome {
				r.Outcome = o.Label
			}
		}
		out = append(out, r)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CompletedOn < out[j].CompletedOn })
	return out
}
