package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

const (
	ListingOpen     = "open"
	ListingReserved = "reserved"
	ListingClosed   = "closed"
	// ListingWithdrawn means the giver took it down before it was handed over.
	ListingWithdrawn = "withdrawn"
)

// MaxPhotos is how many reference photos a listing can carry.
const MaxPhotos = 4

type Listing struct {
	ID string `json:"id"`
	// OwnerID is the account that listed it; only the owner can accept
	// requests and confirm the hand-over.
	OwnerID      string `json:"ownerId"`
	ProviderName string `json:"providerName"`
	Contact      string `json:"contact"`
	Item         string `json:"item"`
	Material     string `json:"material"`
	Quantity     string `json:"quantity"`
	// A listing is measured one way: Pieces for things that are counted,
	// such as monitors or desks, or EstKg (estimated total weight) for bulk
	// material. Exactly one of the two is set.
	Pieces    int     `json:"pieces"`
	EstKg     float64 `json:"estKg"`
	Condition string  `json:"condition"`
	Route     string  `json:"route"`
	Area      string  `json:"area"`
	Notes     string  `json:"notes"`
	Status    string  `json:"status"`
	// Photos is how many reference photos the giver attached, served from
	// /api/listings/{id}/photos/{n} for n from 0 to Photos-1.
	Photos    int       `json:"photos"`
	CreatedAt time.Time `json:"createdAt"`
}

const (
	TransferRequested = "requested"
	TransferAccepted  = "accepted"
	TransferCollected = "collected"
	TransferCompleted = "completed"
	TransferDeclined  = "declined"
	TransferFailed    = "failed"
)

type Transfer struct {
	ID        string `json:"id"`
	ListingID string `json:"listingId"`
	// ReceiverID is the account that asked for the material; only the
	// receiver records the outcome.
	ReceiverID   string  `json:"receiverId"`
	ReceiverName string  `json:"receiverName"`
	ReceiverRole string  `json:"receiverRole"`
	IntendedUse  string  `json:"intendedUse"`
	Status       string  `json:"status"`
	ConfirmedKg  float64 `json:"confirmedKg"`
	// ConfirmedPieces is how many items changed hands, for counted listings;
	// ConfirmedKg is the weight handed over, for listings sold by weight.
	ConfirmedPieces int       `json:"confirmedPieces"`
	Outcome         string    `json:"outcome"`
	OutcomeNote     string    `json:"outcomeNote"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

// ErrInvalid marks a request the caller can fix; handlers answer 400.
var ErrInvalid = errors.New("invalid")

// ErrForbidden marks a step the signed-in user is not allowed to take;
// handlers answer 403.
var ErrForbidden = errors.New("forbidden")

func forbiddenf(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrForbidden, fmt.Sprintf(format, a...))
}

// ErrConflict marks a step that does not fit the current state; handlers answer 409.
var ErrConflict = errors.New("conflict")

func invalidf(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, a...))
}

func conflictf(format string, a ...any) error {
	return fmt.Errorf("%w: %s", ErrConflict, fmt.Sprintf(format, a...))
}

// ValidateListing trims the listing's text fields and checks it can be published.
func ValidateListing(l *Listing) error {
	l.ProviderName = strings.TrimSpace(l.ProviderName)
	l.Item = strings.TrimSpace(l.Item)
	l.Quantity = strings.TrimSpace(l.Quantity)
	l.Condition = strings.TrimSpace(l.Condition)
	l.Contact = strings.TrimSpace(l.Contact)
	l.Notes = strings.TrimSpace(l.Notes)
	switch {
	case l.ProviderName == "":
		return invalidf("add who is giving the material away")
	case l.Item == "":
		return invalidf("describe the item")
	case l.Pieces < 0 || l.Pieces > 100000:
		return invalidf("number of pieces must be between 1 and 100,000")
	case l.Pieces == 0 && (l.EstKg <= 0 || l.EstKg > 100000):
		return invalidf("estimated weight must be between 0 and 100,000 kg")
	}
	if l.Pieces > 0 {
		// Counted listings are measured in pieces only.
		l.EstKg = 0
	}
	if _, ok := MaterialByID(l.Material); !ok {
		return invalidf("unknown material %q", l.Material)
	}
	if _, ok := RouteByID(l.Route); !ok {
		return invalidf("unknown route %q", l.Route)
	}
	if !ValidArea(l.Area) {
		return invalidf("unknown area %q", l.Area)
	}
	return nil
}

// CanRequest reports whether someone in role may request this listing.
func CanRequest(l Listing, role string) error {
	if l.Status != ListingOpen {
		return conflictf("this listing is no longer open")
	}
	if !ValidRole(role) {
		return invalidf("unknown role %q", role)
	}
	r, _ := RouteByID(l.Route)
	if r.RecyclersOnly && role != RoleRecycler {
		return invalidf("only recyclers can collect material that needs specialised handling")
	}
	return nil
}

// Authorize checks that user may take action on transfer t of listing l.
// The giver answers requests and confirms the hand-over; the receiver
// records what they did with it.
func Authorize(userID string, t Transfer, l Listing, action string) error {
	switch action {
	case "accept", "decline", "collect", "fail":
		if userID != l.OwnerID {
			return forbiddenf("only the person who listed this material can do that")
		}
	case "outcome":
		if userID != t.ReceiverID {
			return forbiddenf("only the person who received the material can record what happened to it")
		}
	}
	return nil
}

// Step is one move in a transfer's lifecycle.
type Step struct {
	Action      string  `json:"action"`
	ConfirmedKg float64 `json:"confirmedKg"`
	Pieces      int     `json:"pieces"`
	Outcome     string  `json:"outcome"`
	OutcomeNote string  `json:"outcomeNote"`
}

// Apply moves a transfer for listing l one step and returns the listing
// status that should follow. An empty listing status means the listing is
// unchanged.
func Apply(t *Transfer, l Listing, s Step) (listingStatus string, err error) {
	switch s.Action {
	case "accept":
		if t.Status != TransferRequested {
			return "", conflictf("only a requested transfer can be accepted")
		}
		t.Status = TransferAccepted
		return ListingReserved, nil
	case "decline":
		if t.Status != TransferRequested {
			return "", conflictf("only a requested transfer can be declined")
		}
		t.Status = TransferDeclined
		return "", nil
	case "collect":
		if t.Status != TransferAccepted {
			return "", conflictf("only an accepted transfer can be collected")
		}
		// The giver confirms how much actually changed hands, in the
		// listing's own unit; it can be less than was listed.
		if l.Pieces > 0 {
			if s.Pieces < 1 || s.Pieces > l.Pieces {
				return "", invalidf("enter how many pieces were handed over, from 1 to %d", l.Pieces)
			}
			t.ConfirmedPieces = s.Pieces
		} else {
			if s.ConfirmedKg <= 0 || s.ConfirmedKg > 100000 {
				return "", invalidf("enter the weight handed over, in kg")
			}
			t.ConfirmedKg = s.ConfirmedKg
		}
		t.Status = TransferCollected
		return "", nil
	case "fail":
		if t.Status != TransferAccepted {
			return "", conflictf("only an accepted transfer can be marked as failed")
		}
		t.Status = TransferFailed
		return ListingOpen, nil
	case "outcome":
		if t.Status != TransferCollected {
			return "", conflictf("record the hand-over before the outcome")
		}
		if !ValidOutcome(s.Outcome) {
			return "", invalidf("choose what happened to the material")
		}
		t.Status = TransferCompleted
		t.Outcome = s.Outcome
		t.OutcomeNote = strings.TrimSpace(s.OutcomeNote)
		return ListingClosed, nil
	}
	return "", invalidf("unknown action %q", s.Action)
}
