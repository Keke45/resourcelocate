package domain

import (
	"strings"
	"time"
)

const (
	NeedOpen   = "open"
	NeedClosed = "closed"
)

// SavedNeed is a search a user keeps on their account: "I'm looking for
// 10 monitors in Kabwata". It stays open, and its matches update as new
// listings arrive, until the user closes it.
type SavedNeed struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Material  string    `json:"material"`
	Kg        float64   `json:"kg"`
	Pieces    int       `json:"pieces"`
	Area      string    `json:"area"`
	Note      string    `json:"note"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
}

// ValidateNeed trims and checks a need before it is saved. A need must
// at least name a material, so it can be matched and shown to givers.
func ValidateNeed(n *SavedNeed) error {
	n.Note = strings.TrimSpace(n.Note)
	switch {
	case n.Material == "":
		return invalidf("choose the material you need")
	case n.Kg < 0 || n.Kg > 100000:
		return invalidf("amount must be between 0 and 100,000 kg")
	case n.Pieces < 0 || n.Pieces > 100000:
		return invalidf("number of pieces must be between 0 and 100,000")
	case n.Kg > 0 && n.Pieces > 0:
		return invalidf("give the amount in kg or in pieces, not both")
	case len(n.Note) > 200:
		return invalidf("keep the note under 200 characters")
	}
	if _, ok := MaterialByID(n.Material); !ok {
		return invalidf("unknown material %q", n.Material)
	}
	if n.Area != "" && !ValidArea(n.Area) {
		return invalidf("unknown area %q", n.Area)
	}
	return nil
}

// AsNeed turns a saved need into the matcher's input for a user in role.
func (n SavedNeed) AsNeed(role string) Need {
	return Need{Material: n.Material, Kg: n.Kg, Pieces: n.Pieces, Area: n.Area, Role: role}
}

// Wants reports whether an open need is looking for this listing's
// material, so a giver can be told someone is waiting for it.
func (n SavedNeed) Wants(l Listing) bool {
	return n.Status == NeedOpen && n.Material == l.Material && n.UserID != l.OwnerID
}
