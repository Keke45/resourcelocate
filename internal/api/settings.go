package api

import (
	"net/http"

	"github.com/KeKe45/resourcelocate/internal/auth"
	"github.com/KeKe45/resourcelocate/internal/domain"
)

// updateProfile changes the signed-in user's name and area. Email and role
// stay as they were at sign-up.
func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in struct {
		Name string `json:"name"`
		Area string `json:"area"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	if err := domain.ValidateProfile(&in.Name, &in.Area); err != nil {
		writeErr(w, err)
		return
	}
	if err := s.store.UpdateProfile(r.Context(), u.ID, in.Name, in.Area); err != nil {
		writeErr(w, err)
		return
	}
	u.Name, u.Area = in.Name, in.Area
	writeJSON(w, http.StatusOK, u)
}

// changePassword sets a new password after checking the current one, then
// logs the account out everywhere except this browser.
func (s *Server) changePassword(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in struct {
		Current string `json:"current"`
		New     string `json:"new"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	if _, demo := s.demoKey(u); demo {
		writeMsg(w, http.StatusForbidden, "The demo accounts are shared, so their password can't be changed.")
		return
	}
	hash, err := s.store.PasswordHash(r.Context(), u.ID)
	if err != nil {
		writeErr(w, err)
		return
	}
	if !auth.CheckPassword(hash, in.Current) {
		writeMsg(w, http.StatusBadRequest, "Your current password isn't right.")
		return
	}
	if err := domain.ValidatePassword(in.New); err != nil {
		writeErr(w, err)
		return
	}
	newHash, err := auth.HashPassword(in.New)
	if err != nil {
		writeErr(w, err)
		return
	}
	c, _ := r.Cookie(sessionCookie) // present: signedIn checked it
	if err := s.store.SetPassword(r.Context(), u.ID, newHash, c.Value); err != nil {
		writeErr(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
