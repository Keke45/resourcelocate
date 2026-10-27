package api

import (
	"context"
	"net/http"

	"github.com/KeKe45/resourcelocate/internal/auth"
	"github.com/KeKe45/resourcelocate/internal/domain"
)

// demoAccounts are the two shared accounts judges can use to try both sides
// of a hand-over without signing up. They start empty: nothing is listed
// for them.
var demoAccounts = map[string]domain.User{
	"giver":  {Name: "Demo giver (school)", Email: "demo-giver@resourcelocate.example", Role: domain.RoleBusiness, Area: "Chilenje"},
	"seeker": {Name: "Demo seeker (maker)", Email: "demo-seeker@resourcelocate.example", Role: domain.RoleIndividual, Area: "Kabwata"},
}

// EnableDemoLogins creates the demo accounts if needed and turns on
// one-click login for them. Their password is random and never shown, so
// the only way in is the demo login button.
func (s *Server) EnableDemoLogins(ctx context.Context) error {
	s.demo = map[string]domain.User{}
	for key, u := range demoAccounts {
		pw, _, err := auth.NewToken()
		if err != nil {
			return err
		}
		hash, err := auth.HashPassword(pw)
		if err != nil {
			return err
		}
		if err := s.store.EnsureUser(ctx, &u, hash); err != nil {
			return err
		}
		s.demo[key] = u
	}
	return nil
}

// demoLogin signs in as one of the demo accounts. It answers 404 unless
// the server was started with DEMO_LOGINS=1.
func (s *Server) demoLogin(w http.ResponseWriter, r *http.Request) {
	var in struct {
		As string `json:"as"`
	}
	if err := decode(r, &in); err != nil {
		writeErr(w, err)
		return
	}
	u, ok := s.demo[in.As]
	if !ok {
		writeMsg(w, http.StatusNotFound, "Demo logins are turned off.")
		return
	}
	if err := s.startSession(w, r, u); err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, u)
}

// demoKey reports whether u is one of the demo accounts.
func (s *Server) demoKey(u domain.User) (string, bool) {
	for k, d := range s.demo {
		if d.ID == u.ID {
			return k, true
		}
	}
	return "", false
}
