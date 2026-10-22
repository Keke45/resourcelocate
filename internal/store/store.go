// Package store keeps accounts, listings, needs and transfers in SQLite.
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"github.com/KeKe45/resourcelocate/internal/domain"
)

var ErrNotFound = errors.New("not found")

type Store struct {
	db *sql.DB
}

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id            TEXT PRIMARY KEY,
	name          TEXT NOT NULL,
	email         TEXT NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	role          TEXT NOT NULL,
	area          TEXT NOT NULL,
	created_at    TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS sessions (
	token_hash TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users(id),
	expires_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS listings (
	id            TEXT PRIMARY KEY,
	owner_id      TEXT NOT NULL REFERENCES users(id),
	provider_name TEXT NOT NULL,
	contact       TEXT NOT NULL DEFAULT '',
	item          TEXT NOT NULL,
	material      TEXT NOT NULL,
	quantity      TEXT NOT NULL DEFAULT '',
	pieces        INTEGER NOT NULL DEFAULT 0,
	est_kg        REAL NOT NULL,
	condition     TEXT NOT NULL DEFAULT '',
	route         TEXT NOT NULL,
	area          TEXT NOT NULL,
	notes         TEXT NOT NULL DEFAULT '',
	status        TEXT NOT NULL,
	created_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS listings_owner ON listings(owner_id);
CREATE TABLE IF NOT EXISTS needs (
	id         TEXT PRIMARY KEY,
	user_id    TEXT NOT NULL REFERENCES users(id),
	material   TEXT NOT NULL,
	kg         REAL NOT NULL DEFAULT 0,
	pieces     INTEGER NOT NULL DEFAULT 0,
	area       TEXT NOT NULL DEFAULT '',
	note       TEXT NOT NULL DEFAULT '',
	status     TEXT NOT NULL,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS needs_user ON needs(user_id);
CREATE TABLE IF NOT EXISTS transfers (
	id               TEXT PRIMARY KEY,
	listing_id       TEXT NOT NULL REFERENCES listings(id),
	receiver_id      TEXT NOT NULL REFERENCES users(id),
	receiver_name    TEXT NOT NULL,
	receiver_role    TEXT NOT NULL,
	intended_use     TEXT NOT NULL DEFAULT '',
	status           TEXT NOT NULL,
	confirmed_kg     REAL NOT NULL DEFAULT 0,
	confirmed_pieces INTEGER NOT NULL DEFAULT 0,
	kg_estimated     INTEGER NOT NULL DEFAULT 0,
	counterfactual   TEXT NOT NULL DEFAULT '',
	outcome          TEXT NOT NULL DEFAULT '',
	outcome_note     TEXT NOT NULL DEFAULT '',
	created_at       TEXT NOT NULL,
	updated_at       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS transfers_listing ON transfers(listing_id);
CREATE INDEX IF NOT EXISTS transfers_receiver ON transfers(receiver_id);
`

// Open opens (and if needed creates) the database at path.
func Open(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path+"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)")
	if err != nil {
		return nil, err
	}
	// SQLite allows one writer; a single connection keeps transactions simple.
	db.SetMaxOpenConns(1)
	if _, err := db.Exec(schema); err != nil {
		db.Close()
		return nil, fmt.Errorf("create schema: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func NewID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func ts(t time.Time) string { return t.Format(time.RFC3339Nano) }

func parseTS(s string) time.Time {
	t, _ := time.Parse(time.RFC3339Nano, s)
	return t
}

type scanner interface{ Scan(...any) error }

// prefixed qualifies each column in a comma-separated list with a table alias.
func prefixed(alias, cols string) string {
	parts := strings.Split(cols, ",")
	for i, c := range parts {
		parts[i] = alias + strings.TrimSpace(c)
	}
	return strings.Join(parts, ", ")
}

// ---- Listings ----

const listingCols = `id, owner_id, provider_name, contact, item, material, quantity, pieces, est_kg, condition, route, area, notes, status, created_at`

func scanListing(r scanner) (domain.Listing, error) {
	var l domain.Listing
	var created string
	err := r.Scan(&l.ID, &l.OwnerID, &l.ProviderName, &l.Contact, &l.Item, &l.Material, &l.Quantity, &l.Pieces, &l.EstKg,
		&l.Condition, &l.Route, &l.Area, &l.Notes, &l.Status, &created)
	l.CreatedAt = parseTS(created)
	return l, err
}

func queryListings(ctx context.Context, q interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}, where string, args ...any) ([]domain.Listing, error) {
	rows, err := q.QueryContext(ctx, `SELECT `+listingCols+` FROM listings `+where+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Listing{}
	for rows.Next() {
		l, err := scanListing(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// Listings returns every listing, newest first.
func (s *Store) Listings(ctx context.Context) ([]domain.Listing, error) {
	return queryListings(ctx, s.db, "")
}

// ListingsBy returns the listings a user has published, newest first.
func (s *Store) ListingsBy(ctx context.Context, userID string) ([]domain.Listing, error) {
	return queryListings(ctx, s.db, "WHERE owner_id = ?", userID)
}

func (s *Store) Listing(ctx context.Context, id string) (domain.Listing, error) {
	l, err := scanListing(s.db.QueryRowContext(ctx, `SELECT `+listingCols+` FROM listings WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return l, ErrNotFound
	}
	return l, err
}

func (s *Store) CreateListing(ctx context.Context, l *domain.Listing) error {
	if l.ID == "" {
		l.ID = NewID()
	}
	if l.CreatedAt.IsZero() {
		l.CreatedAt = time.Now().UTC()
	}
	if l.Status == "" {
		l.Status = domain.ListingOpen
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO listings (`+listingCols+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		l.ID, l.OwnerID, l.ProviderName, l.Contact, l.Item, l.Material, l.Quantity, l.Pieces, l.EstKg, l.Condition,
		l.Route, l.Area, l.Notes, l.Status, ts(l.CreatedAt))
	return err
}

// WithdrawListing closes an open listing on its owner's request and
// declines any requests still waiting on it. A listing with an accepted
// hand-over cannot be withdrawn; that hand-over must finish or fail first.
func (s *Store) WithdrawListing(ctx context.Context, userID, id string) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	l, err := scanListing(tx.QueryRowContext(ctx, `SELECT `+listingCols+` FROM listings WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if l.OwnerID != userID {
		return fmt.Errorf("%w: only the person who listed this material can withdraw it", domain.ErrForbidden)
	}
	if l.Status != domain.ListingOpen {
		return fmt.Errorf("%w: only an open listing can be withdrawn", domain.ErrConflict)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE listings SET status=? WHERE id=?`, domain.ListingWithdrawn, id); err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE transfers SET status=?, updated_at=? WHERE listing_id=? AND status=?`,
		domain.TransferDeclined, ts(time.Now().UTC()), id, domain.TransferRequested)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// ---- Needs ----

const needCols = `id, user_id, material, kg, pieces, area, note, status, created_at`

func scanNeed(r scanner) (domain.SavedNeed, error) {
	var n domain.SavedNeed
	var created string
	err := r.Scan(&n.ID, &n.UserID, &n.Material, &n.Kg, &n.Pieces, &n.Area, &n.Note, &n.Status, &created)
	n.CreatedAt = parseTS(created)
	return n, err
}

func (s *Store) queryNeeds(ctx context.Context, where string, args ...any) ([]domain.SavedNeed, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT `+needCols+` FROM needs `+where+` ORDER BY created_at DESC`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.SavedNeed{}
	for rows.Next() {
		n, err := scanNeed(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// NeedsBy returns a user's open needs, newest first.
func (s *Store) NeedsBy(ctx context.Context, userID string) ([]domain.SavedNeed, error) {
	return s.queryNeeds(ctx, "WHERE user_id = ? AND status = ?", userID, domain.NeedOpen)
}

// OpenNeeds returns everyone's open needs, so givers can see demand.
func (s *Store) OpenNeeds(ctx context.Context) ([]domain.SavedNeed, error) {
	return s.queryNeeds(ctx, "WHERE status = ?", domain.NeedOpen)
}

func (s *Store) CreateNeed(ctx context.Context, n *domain.SavedNeed) error {
	n.ID = NewID()
	n.Status = domain.NeedOpen
	n.CreatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `INSERT INTO needs (`+needCols+`) VALUES (?,?,?,?,?,?,?,?,?)`,
		n.ID, n.UserID, n.Material, n.Kg, n.Pieces, n.Area, n.Note, n.Status, ts(n.CreatedAt))
	return err
}

// CloseNeed marks one of the user's needs as no longer wanted.
func (s *Store) CloseNeed(ctx context.Context, userID, id string) error {
	res, err := s.db.ExecContext(ctx, `UPDATE needs SET status=? WHERE id=? AND user_id=?`, domain.NeedClosed, id, userID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- Transfers ----

const transferCols = `id, listing_id, receiver_id, receiver_name, receiver_role, intended_use, status, confirmed_kg, confirmed_pieces, kg_estimated, counterfactual, outcome, outcome_note, created_at, updated_at`

func scanTransfer(r scanner) (domain.Transfer, error) {
	var t domain.Transfer
	var created, updated string
	err := r.Scan(&t.ID, &t.ListingID, &t.ReceiverID, &t.ReceiverName, &t.ReceiverRole, &t.IntendedUse, &t.Status,
		&t.ConfirmedKg, &t.ConfirmedPieces, &t.KgEstimated, &t.Counterfactual, &t.Outcome, &t.OutcomeNote, &created, &updated)
	t.CreatedAt, t.UpdatedAt = parseTS(created), parseTS(updated)
	return t, err
}

func (s *Store) queryTransfers(ctx context.Context, query string, args ...any) ([]domain.Transfer, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.Transfer{}
	for rows.Next() {
		t, err := scanTransfer(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Transfers returns every transfer, most recently updated first.
func (s *Store) Transfers(ctx context.Context) ([]domain.Transfer, error) {
	return s.queryTransfers(ctx, `SELECT `+transferCols+` FROM transfers ORDER BY updated_at DESC`)
}

// TransfersFor returns the transfers userID is part of, as the giver of the
// listing or as the receiver, most recently updated first.
func (s *Store) TransfersFor(ctx context.Context, userID string) ([]domain.Transfer, error) {
	return s.queryTransfers(ctx, `SELECT `+prefixed("t.", transferCols)+` FROM transfers t
		JOIN listings l ON l.id = t.listing_id
		WHERE t.receiver_id = ? OR l.owner_id = ?
		ORDER BY t.updated_at DESC`, userID, userID)
}

// RequestListing creates a requested transfer after checking, inside the
// same transaction, that the listing is still open to this user.
func (s *Store) RequestListing(ctx context.Context, t *domain.Transfer) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	l, err := scanListing(tx.QueryRowContext(ctx, `SELECT `+listingCols+` FROM listings WHERE id = ?`, t.ListingID))
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if err := domain.CanRequest(l, t.ReceiverRole); err != nil {
		return err
	}
	if l.OwnerID == t.ReceiverID {
		return fmt.Errorf("%w: you listed this material yourself", domain.ErrInvalid)
	}
	var waiting int
	err = tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM transfers WHERE listing_id=? AND receiver_id=? AND status=?`,
		l.ID, t.ReceiverID, domain.TransferRequested).Scan(&waiting)
	if err != nil {
		return err
	}
	if waiting > 0 {
		return fmt.Errorf("%w: you have already requested this material", domain.ErrConflict)
	}
	now := time.Now().UTC()
	t.ID = NewID()
	t.Status = domain.TransferRequested
	t.CreatedAt, t.UpdatedAt = now, now
	_, err = tx.ExecContext(ctx, `INSERT INTO transfers (`+transferCols+`) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.ListingID, t.ReceiverID, t.ReceiverName, t.ReceiverRole, t.IntendedUse, t.Status, t.ConfirmedKg,
		t.ConfirmedPieces, t.KgEstimated, t.Counterfactual, t.Outcome, t.OutcomeNote, ts(t.CreatedAt), ts(t.UpdatedAt))
	if err != nil {
		return err
	}
	return tx.Commit()
}

// Advance applies one lifecycle step, taken by user actorID, to a transfer
// and updates its listing to match. Accepting a request declines the other
// open requests for the same listing.
func (s *Store) Advance(ctx context.Context, actorID, id string, step domain.Step) (domain.Transfer, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Transfer{}, err
	}
	defer tx.Rollback()

	t, err := scanTransfer(tx.QueryRowContext(ctx, `SELECT `+transferCols+` FROM transfers WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return t, ErrNotFound
	} else if err != nil {
		return t, err
	}
	l, err := scanListing(tx.QueryRowContext(ctx, `SELECT `+listingCols+` FROM listings WHERE id = ?`, t.ListingID))
	if err != nil {
		return t, err
	}
	if err := domain.Authorize(actorID, t, l, step.Action); err != nil {
		return t, err
	}
	listingStatus, err := domain.Apply(&t, l, step)
	if err != nil {
		return t, err
	}
	t.UpdatedAt = time.Now().UTC()
	_, err = tx.ExecContext(ctx, `UPDATE transfers SET status=?, confirmed_kg=?, confirmed_pieces=?, kg_estimated=?, counterfactual=?, outcome=?, outcome_note=?, updated_at=? WHERE id=?`,
		t.Status, t.ConfirmedKg, t.ConfirmedPieces, t.KgEstimated, t.Counterfactual, t.Outcome, t.OutcomeNote, ts(t.UpdatedAt), t.ID)
	if err != nil {
		return t, err
	}
	if listingStatus != "" {
		if _, err := tx.ExecContext(ctx, `UPDATE listings SET status=? WHERE id=?`, listingStatus, t.ListingID); err != nil {
			return t, err
		}
	}
	if step.Action == "accept" {
		_, err = tx.ExecContext(ctx, `UPDATE transfers SET status=?, updated_at=? WHERE listing_id=? AND id<>? AND status=?`,
			domain.TransferDeclined, ts(t.UpdatedAt), t.ListingID, t.ID, domain.TransferRequested)
		if err != nil {
			return t, err
		}
	}
	return t, tx.Commit()
}
