// Package api serves the JSON API used by the React front end.
package api

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/KeKe45/resourcelocate/internal/auth"
	"github.com/KeKe45/resourcelocate/internal/domain"
	"github.com/KeKe45/resourcelocate/internal/store"
)

// maxListingPhotoBytes caps stored photos. The browser shrinks photos to
// about 1280 px before upload, which is far below this.
const maxListingPhotoBytes = 2 << 20

type Server struct {
	store  *store.Store
	limits *limits
	// dummyHash is checked when a login email is unknown, so a wrong email
	// takes as long as a wrong password and does not reveal who has an account.
	dummyHash string
	// demo holds the shared demo accounts by key ("giver", "seeker"); empty
	// unless EnableDemoLogins was called.
	demo map[string]domain.User
}

// New returns the API server. trustProxy says whether to take the client's
// IP from X-Forwarded-For, which is only safe behind a proxy that sets it.
func New(s *store.Store, trustProxy bool) *Server {
	h, err := auth.HashPassword("not a real password")
	if err != nil {
		panic(err)
	}
	return &Server{store: s, limits: newLimits(trustProxy), dummyHash: h}
}

// Routes registers every API endpoint on mux. Endpoints wrapped in
// s.signedIn need a session cookie; the rest are public. Every endpoint is
// rate limited per client IP, and the ones that check a password (wrapped
// in s.authLimited) much more tightly.
func (s *Server) Routes(mux *http.ServeMux) {
	api := http.NewServeMux()
	api.HandleFunc("GET /api/meta", s.meta)
	api.HandleFunc("GET /api/stats", s.stats)
	api.HandleFunc("POST /api/auth/signup", s.authLimited(s.signup))
	api.HandleFunc("POST /api/auth/login", s.authLimited(s.login))
	api.HandleFunc("POST /api/auth/logout", s.logout)
	api.HandleFunc("POST /api/auth/demo", s.authLimited(s.demoLogin))
	api.HandleFunc("GET /api/records.csv", s.records)
	api.HandleFunc("GET /api/me", s.signedIn(s.me))
	api.HandleFunc("PUT /api/me", s.signedIn(s.updateProfile))
	api.HandleFunc("POST /api/me/password", s.authLimited(s.signedIn(s.changePassword)))

	api.HandleFunc("GET /api/dashboard", s.signedIn(s.dashboard))
	api.HandleFunc("POST /api/listings", s.signedIn(s.createListing))
	api.HandleFunc("PUT /api/listings/{id}", s.signedIn(s.updateListing))
	api.HandleFunc("POST /api/listings/{id}/withdraw", s.signedIn(s.withdrawListing))
	api.HandleFunc("POST /api/listings/{id}/photos", s.signedIn(s.uploadPhoto))
	api.HandleFunc("DELETE /api/listings/{id}/photos/{n}", s.signedIn(s.deletePhoto))
	api.HandleFunc("GET /api/listings/{id}/photos/{n}", s.photo)
	api.HandleFunc("GET /api/public/listings/{id}", s.publicListing)
	api.HandleFunc("GET /api/matches", s.signedIn(s.matches))
	api.HandleFunc("GET /api/needs", s.signedIn(s.needs))
	api.HandleFunc("POST /api/needs", s.signedIn(s.createNeed))
	api.HandleFunc("PUT /api/needs/{id}", s.signedIn(s.updateNeed))
	api.HandleFunc("POST /api/needs/{id}/close", s.signedIn(s.closeNeed))
	api.HandleFunc("GET /api/transfers", s.signedIn(s.transfers))
	api.HandleFunc("POST /api/transfers", s.signedIn(s.requestListing))
	api.HandleFunc("POST /api/transfers/{id}/{action}", s.signedIn(s.advance))
	mux.Handle("/api/", s.limited(api))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte("ok")) })
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeMsg(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

// writeErr maps domain and store errors to HTTP statuses. Validation,
// permission and conflict messages are shown to the user; anything else is
// logged and hidden.
func writeErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalid):
		writeMsg(w, http.StatusBadRequest, cleanMsg(err))
	case errors.Is(err, domain.ErrForbidden):
		writeMsg(w, http.StatusForbidden, cleanMsg(err))
	case errors.Is(err, domain.ErrConflict):
		writeMsg(w, http.StatusConflict, cleanMsg(err))
	case errors.Is(err, store.ErrNotFound):
		writeMsg(w, http.StatusNotFound, "That item no longer exists.")
	default:
		log.Printf("error: %v", err)
		writeMsg(w, http.StatusInternalServerError, "Something went wrong on the server. Try again.")
	}
}

// cleanMsg drops the sentinel prefix and capitalises the rest for display.
func cleanMsg(err error) string {
	m := err.Error()
	if i := strings.Index(m, ": "); i >= 0 {
		m = m[i+2:]
	}
	if m == "" {
		return m
	}
	return strings.ToUpper(m[:1]) + m[1:] + "."
}

// decode reads a JSON body of at most 1 MB and rejects unknown fields.
func decode(r *http.Request, v any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return errors.Join(domain.ErrInvalid, errors.New("request body: the form data could not be read"))
	}
	return nil
}

// ---- Sessions ----

const sessionCookie = "rl_session"

type handler func(w http.ResponseWriter, r *http.Request, u domain.User)

// signedIn loads the user from the session cookie and answers 401 when
// there is none, so handlers behind it always have a user.
func (s *Server) signedIn(h handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil {
			writeMsg(w, http.StatusUnauthorized, "Log in to continue.")
			return
		}
		u, err := s.store.UserForSession(r.Context(), c.Value)
		if errors.Is(err, store.ErrNotFound) {
			writeMsg(w, http.StatusUnauthorized, "Your session has ended. Log in again.")
			return
		} else if err != nil {
			writeErr(w, err)
			return
		}
		h(w, r, u)
	}
}

// startSession logs u in by setting the session cookie. The cookie is
// HttpOnly so scripts cannot read it, and SameSite=Lax so other sites
// cannot send it with a form post.
func (s *Server) startSession(w http.ResponseWriter, r *http.Request, u domain.User) error {
	token, expires, err := s.store.CreateSession(r.Context(), u.ID)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name: sessionCookie, Value: token, Path: "/", Expires: expires,
		HttpOnly: true, SameSite: http.SameSiteLaxMode,
		Secure: r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https",
	})
	return nil
}

func (s *Server) signup(w http.ResponseWriter, r *http.Request) {
	var in domain.Signup
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	if err := domain.ValidateSignup(&in); err != nil {
		writeErr(w, err)
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		writeErr(w, err)
		return
	}
	u := domain.User{Name: in.Name, Email: in.Email, Role: in.Role, Area: in.Area}
	if err := s.store.CreateUser(r.Context(), &u, hash); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.startSession(w, r, u); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, u)
}

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	// Also limit by account, so guessing one password from many IPs is slow too.
	if !s.limits.allow(w, s.limits.account, "login:"+domain.NormalizeEmail(in.Email)) {
		return
	}
	u, hash, err := s.store.UserByEmail(r.Context(), in.Email)
	if errors.Is(err, store.ErrNotFound) {
		auth.CheckPassword(s.dummyHash, in.Password)
		writeMsg(w, http.StatusUnauthorized, "That email and password don't match an account.")
		return
	} else if err != nil {
		writeErr(w, err)
		return
	}
	if !auth.CheckPassword(hash, in.Password) {
		writeMsg(w, http.StatusUnauthorized, "That email and password don't match an account.")
		return
	}
	if err := s.startSession(w, r, u); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(sessionCookie); err == nil {
		_ = s.store.DeleteSession(r.Context(), c.Value)
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) me(w http.ResponseWriter, _ *http.Request, u domain.User) {
	writeJSON(w, http.StatusOK, u)
}

// ---- Public ----

// meta returns the catalog the front end needs to build its forms and labels.
func (s *Server) meta(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"materials":  domain.Materials,
		"routes":     domain.Routes,
		"areas":      domain.Areas,
		"roles":      domain.Roles,
		"outcomes":   domain.Outcomes,
		"maxPhotos":  domain.MaxPhotos,
		"demoLogins": len(s.demo) > 0,
	})
}

// stats returns community-wide totals for the landing page.
func (s *Server) stats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ls, err := s.store.Listings(ctx)
	if err != nil {
		writeErr(w, err)
		return
	}
	ts, err := s.store.Transfers(ctx)
	if err != nil {
		writeErr(w, err)
		return
	}
	needs, err := s.store.OpenNeeds(ctx)
	if err != nil {
		writeErr(w, err)
		return
	}
	open := 0
	for _, l := range ls {
		if l.Status == domain.ListingOpen {
			open++
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"openListings": open,
		"openNeeds":    len(needs),
		"totals":       domain.ComputeTotals(ts, byID(ls), ""),
	})
}

func byID(ls []domain.Listing) map[string]domain.Listing {
	m := make(map[string]domain.Listing, len(ls))
	for _, l := range ls {
		m[l.ID] = l
	}
	return m
}

// ---- Listings ----

// createListing publishes a listing for the signed-in user. Only fields a
// giver may set are copied, so a client cannot choose its own id, owner or
// status. The answer says how many people have an open need for the material.
func (s *Server) createListing(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in domain.Listing
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	l := domain.Listing{
		OwnerID: u.ID, ProviderName: u.Name, Contact: in.Contact, Item: in.Item, Material: in.Material,
		Quantity: in.Quantity, Pieces: in.Pieces, EstKg: in.EstKg, Condition: in.Condition, Route: in.Route,
		Area: in.Area, Notes: in.Notes,
	}
	if err := domain.ValidateListing(&l); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.store.CreateListing(r.Context(), &l); err != nil {
		writeErr(w, err)
		return
	}
	needs, err := s.store.OpenNeeds(r.Context())
	if err != nil {
		writeErr(w, err)
		return
	}
	waiting := 0
	for _, n := range needs {
		if n.Wants(l) {
			waiting++
		}
	}
	writeJSON(w, http.StatusCreated, map[string]any{"listing": l, "waiting": waiting})
}

// updateListing changes the details of one of the user's open listings.
// As with createListing, only the fields a giver may set are copied.
func (s *Server) updateListing(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in domain.Listing
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	l := domain.Listing{
		ID: r.PathValue("id"), ProviderName: u.Name, Contact: in.Contact, Item: in.Item, Material: in.Material,
		Quantity: in.Quantity, Pieces: in.Pieces, EstKg: in.EstKg, Condition: in.Condition, Route: in.Route,
		Area: in.Area, Notes: in.Notes,
	}
	if err := domain.ValidateListing(&l); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.store.UpdateListing(r.Context(), u.ID, &l); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func (s *Server) withdrawListing(w http.ResponseWriter, r *http.Request, u domain.User) {
	if err := s.store.WithdrawListing(r.Context(), u.ID, r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// uploadPhoto adds a reference photo to one of the user's listings, so
// people looking can see what it is. The body is the image itself; only
// JPEG, PNG and WebP are kept.
func (s *Server) uploadPhoto(w http.ResponseWriter, r *http.Request, u domain.User) {
	data, err := io.ReadAll(io.LimitReader(r.Body, maxListingPhotoBytes+1))
	if err != nil || len(data) == 0 || len(data) > maxListingPhotoBytes {
		writeMsg(w, http.StatusBadRequest, "Attach a photo under 2 MB.")
		return
	}
	ct := http.DetectContentType(data)
	switch ct {
	case "image/jpeg", "image/png", "image/webp":
	default:
		writeMsg(w, http.StatusBadRequest, "Use a JPEG, PNG or WebP photo.")
		return
	}
	n, err := s.store.AddPhoto(r.Context(), u.ID, r.PathValue("id"), ct, data)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]int{"n": n})
}

func (s *Server) deletePhoto(w http.ResponseWriter, r *http.Request, u domain.User) {
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		writeErr(w, store.ErrNotFound)
		return
	}
	if err := s.store.DeletePhoto(r.Context(), u.ID, r.PathValue("id"), n); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// photo serves one of a listing's photos. Photos are public, like the
// listing's share page, so link previews in chat apps can show them; a
// withdrawn listing's photos are not served.
func (s *Server) photo(w http.ResponseWriter, r *http.Request) {
	n, err := strconv.Atoi(r.PathValue("n"))
	if err != nil {
		writeErr(w, store.ErrNotFound)
		return
	}
	l, err := s.store.Listing(r.Context(), r.PathValue("id"))
	if err == nil && l.Status == domain.ListingWithdrawn {
		err = store.ErrNotFound
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	ct, data, err := s.store.Photo(r.Context(), l.ID, n)
	if err != nil {
		writeErr(w, err)
		return
	}
	// Photos move down a place when an earlier one is removed, so the same
	// URL can change: browsers keep a copy but check it with the ETag.
	sum := sha256.Sum256(data)
	etag := `"` + hex.EncodeToString(sum[:8]) + `"`
	w.Header().Set("ETag", etag)
	w.Header().Set("Cache-Control", "public, no-cache")
	if r.Header.Get("If-None-Match") == etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	w.Header().Set("Content-Type", ct)
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Write(data)
}

// ---- Matching and needs ----

// othersListings returns every listing except the user's own, which they
// cannot request.
func (s *Server) othersListings(ctx context.Context, userID string) ([]domain.Listing, error) {
	ls, err := s.store.Listings(ctx)
	if err != nil {
		return nil, err
	}
	out := ls[:0]
	for _, l := range ls {
		if l.OwnerID != userID {
			out = append(out, l)
		}
	}
	return out, nil
}

// matches ranks open listings against the need given in the query string
// (material, kg or pieces, area). The role is the signed-in user's.
func (s *Server) matches(w http.ResponseWriter, r *http.Request, u domain.User) {
	q := r.URL.Query()
	kg, _ := strconv.ParseFloat(q.Get("kg"), 64)
	pieces, _ := strconv.Atoi(q.Get("pieces"))
	ls, err := s.othersListings(r.Context(), u.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	need := domain.Need{Material: q.Get("material"), Kg: kg, Pieces: pieces, Area: q.Get("area"), Role: u.Role}
	writeJSON(w, http.StatusOK, domain.Rank(ls, need))
}

// needView is a saved need with its current matches, worked out on every
// read so it always reflects the listings open right now.
type needView struct {
	domain.SavedNeed
	Matches []domain.Match `json:"matches"`
}

func (s *Server) needViews(ctx context.Context, u domain.User) ([]needView, error) {
	needs, err := s.store.NeedsBy(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	ls, err := s.othersListings(ctx, u.ID)
	if err != nil {
		return nil, err
	}
	out := make([]needView, 0, len(needs))
	for _, n := range needs {
		out = append(out, needView{n, domain.Rank(ls, n.AsNeed(u.Role))})
	}
	return out, nil
}

func (s *Server) needs(w http.ResponseWriter, r *http.Request, u domain.User) {
	out, err := s.needViews(r.Context(), u)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) createNeed(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in domain.SavedNeed
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	n := domain.SavedNeed{UserID: u.ID, Material: in.Material, Kg: in.Kg, Pieces: in.Pieces, Area: in.Area, Note: in.Note}
	if err := domain.ValidateNeed(&n); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.store.CreateNeed(r.Context(), &n); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, n)
}

func (s *Server) updateNeed(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in domain.SavedNeed
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	n := domain.SavedNeed{ID: r.PathValue("id"), Material: in.Material, Kg: in.Kg, Pieces: in.Pieces, Area: in.Area, Note: in.Note}
	if err := domain.ValidateNeed(&n); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.store.UpdateNeed(r.Context(), u.ID, &n); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, n)
}

func (s *Server) closeNeed(w http.ResponseWriter, r *http.Request, u domain.User) {
	if err := s.store.CloseNeed(r.Context(), u.ID, r.PathValue("id")); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---- Transfers ----

// transferView adds the listing details the hand-over screens show, and
// which side of the hand-over the signed-in user is on.
type transferView struct {
	domain.Transfer
	Item         string `json:"item"`
	Material     string `json:"material"`
	Route        string `json:"route"`
	ProviderName string `json:"providerName"`
	Contact      string `json:"contact"`
	Area         string `json:"area"`
	// The listing's pieces and estimated weight, so the hand-over form can
	// ask for the amount in the listing's unit.
	ListingPieces int     `json:"listingPieces"`
	ListingEstKg  float64 `json:"listingEstKg"`
	// Side is "giving" when the user listed the material, "receiving" when
	// they asked for it.
	Side string `json:"side"`
}

func (s *Server) transferViews(ctx context.Context, u domain.User) ([]transferView, []domain.Transfer, error) {
	ts, err := s.store.TransfersFor(ctx, u.ID)
	if err != nil {
		return nil, nil, err
	}
	ls, err := s.store.Listings(ctx)
	if err != nil {
		return nil, nil, err
	}
	m := byID(ls)
	out := make([]transferView, 0, len(ts))
	for _, t := range ts {
		l := m[t.ListingID]
		side := "receiving"
		if l.OwnerID == u.ID {
			side = "giving"
		}
		// The giver's phone number is only shared once they accept.
		contact := ""
		if side == "giving" || t.Status != domain.TransferRequested && t.Status != domain.TransferDeclined {
			contact = l.Contact
		}
		out = append(out, transferView{t, l.Item, l.Material, l.Route, l.ProviderName, contact, l.Area, l.Pieces, l.EstKg, side})
	}
	return out, ts, nil
}

func (s *Server) transfers(w http.ResponseWriter, r *http.Request, u domain.User) {
	out, _, err := s.transferViews(r.Context(), u)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// requestListing creates a hand-over in the requested state for the
// signed-in user.
func (s *Server) requestListing(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in struct {
		ListingID   string `json:"listingId"`
		IntendedUse string `json:"intendedUse"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	t := domain.Transfer{
		ListingID: in.ListingID, ReceiverID: u.ID, ReceiverName: u.Name, ReceiverRole: u.Role,
		IntendedUse: strings.TrimSpace(in.IntendedUse),
	}
	if len(t.IntendedUse) > 140 {
		writeMsg(w, http.StatusBadRequest, "Keep the intended use under 140 characters.")
		return
	}
	if err := s.store.RequestListing(r.Context(), &t); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

// advance applies one lifecycle action (accept, decline, fail, collect,
// outcome) to a hand-over. The action comes from the path; collect and
// outcome also send their details in the body. The store checks the user
// is the right side of the hand-over for the action.
func (s *Server) advance(w http.ResponseWriter, r *http.Request, u domain.User) {
	step := domain.Step{Action: r.PathValue("action")}
	if r.ContentLength != 0 {
		var body domain.Step
		if err := decode(r, &body); err != nil {
			writeErr(w, err)
			return
		}
		body.Action = step.Action
		step = body
	}
	t, err := s.store.Advance(r.Context(), u.ID, r.PathValue("id"), step)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, t)
}

// ---- Dashboard ----

// dashboard returns everything the signed-in user's home screen shows:
// their listings, saved needs with live matches, hand-overs, and how much
// they have given and claimed.
func (s *Server) dashboard(w http.ResponseWriter, r *http.Request, u domain.User) {
	ctx := r.Context()
	listings, err := s.store.ListingsBy(ctx, u.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	needs, err := s.needViews(ctx, u)
	if err != nil {
		writeErr(w, err)
		return
	}
	views, ts, err := s.transferViews(ctx, u)
	if err != nil {
		writeErr(w, err)
		return
	}
	all, err := s.store.Listings(ctx)
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"listings":  listings,
		"needs":     needs,
		"transfers": views,
		"totals":    domain.ComputeTotals(ts, byID(all), u.ID),
	})
}
