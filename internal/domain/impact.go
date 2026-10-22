package domain

import "sort"

type MaterialImpact struct {
	Material   string  `json:"material"`
	Label      string  `json:"label"`
	HandedKg   float64 `json:"handedKg"`
	DivertedKg float64 `json:"divertedKg"`
	KgCO2e     float64 `json:"kgCO2e"`
	Pieces     int     `json:"pieces"`
}

type Impact struct {
	// Completed transfers have a confirmed weight and a recorded outcome.
	Completed int `json:"completed"`
	Failed    int `json:"failed"`
	// InProgress counts requested, accepted and collected transfers.
	InProgress int `json:"inProgress"`
	// CompletionRate is completed / (completed + failed); 0 when neither exists.
	CompletionRate float64 `json:"completionRate"`
	HandedKg       float64 `json:"handedKg"`
	DivertedKg     float64 `json:"divertedKg"`
	KgCO2e         float64 `json:"kgCO2e"`
	// Pieces counts items handed on from counted listings (monitors, desks...).
	Pieces int `json:"pieces"`
	// EstimatedKg is the part of HandedKg worked out from weight per piece
	// rather than weighed.
	EstimatedKg float64          `json:"estimatedKg"`
	ByMaterial  []MaterialImpact `json:"byMaterial"`
	ByOutcome   map[string]int   `json:"byOutcome"`
}

// ComputeImpact counts only completed transfers. Weight counts as
// diverted only when the provider said the material would otherwise have
// been dumped, burned or landfilled.
func ComputeImpact(transfers []Transfer, listings map[string]Listing) Impact {
	imp := Impact{ByOutcome: map[string]int{}}
	byMat := map[string]*MaterialImpact{}
	for _, t := range transfers {
		switch t.Status {
		case TransferFailed:
			imp.Failed++
			continue
		case TransferRequested, TransferAccepted, TransferCollected:
			imp.InProgress++
			continue
		case TransferCompleted:
		default:
			continue
		}
		imp.Completed++
		imp.ByOutcome[t.Outcome]++

		l := listings[t.ListingID]
		mat, ok := MaterialByID(l.Material)
		if !ok {
			mat, _ = MaterialByID("other")
		}
		mi := byMat[mat.ID]
		if mi == nil {
			mi = &MaterialImpact{Material: mat.ID, Label: mat.Label}
			byMat[mat.ID] = mi
		}
		mi.HandedKg += t.ConfirmedKg
		imp.HandedKg += t.ConfirmedKg
		mi.Pieces += t.ConfirmedPieces
		imp.Pieces += t.ConfirmedPieces
		if t.KgEstimated {
			imp.EstimatedKg += t.ConfirmedKg
		}
		if c, ok := CounterfactualByID(t.Counterfactual); ok && c.Diverted {
			mi.DivertedKg += t.ConfirmedKg
			mi.KgCO2e += t.ConfirmedKg * mat.FactorKgCO2e
			imp.DivertedKg += t.ConfirmedKg
			imp.KgCO2e += t.ConfirmedKg * mat.FactorKgCO2e
		}
	}
	if done := imp.Completed + imp.Failed; done > 0 {
		imp.CompletionRate = float64(imp.Completed) / float64(done)
	}
	imp.ByMaterial = []MaterialImpact{}
	for _, mi := range byMat {
		imp.ByMaterial = append(imp.ByMaterial, *mi)
	}
	sort.Slice(imp.ByMaterial, func(i, j int) bool {
		return imp.ByMaterial[i].HandedKg > imp.ByMaterial[j].HandedKg
	})
	return imp
}
