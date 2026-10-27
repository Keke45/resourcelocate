package api

import (
	"bytes"
	"fmt"
	"html"
	"net/http"
	"strings"

	"github.com/KeKe45/resourcelocate/internal/domain"
)

// publicView is what anyone with a listing's link may see. The giver's
// phone number is left out; it is only shared after they accept a request.
func publicView(l domain.Listing) domain.Listing {
	l.Contact = ""
	return l
}

// publicListing returns one listing for its share page, signed in or not.
func (s *Server) publicListing(w http.ResponseWriter, r *http.Request) {
	l, err := s.store.Listing(r.Context(), r.PathValue("id"))
	if err == nil && l.Status == domain.ListingWithdrawn {
		err = errWithdrawn
	}
	if err != nil {
		writeErr(w, err)
		return
	}
	writeJSON(w, http.StatusOK, publicView(l))
}

var errWithdrawn = fmt.Errorf("%w: the giver took this listing down", domain.ErrConflict)

// SharePage serves the app for a listing's share link, /l/{id}, with link
// preview tags (Open Graph) filled in, so WhatsApp, Facebook, X, Telegram
// and others show the item, amount, area and first photo. index is the
// built index.html; the app itself then shows the listing.
func (s *Server) SharePage(index []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page := index
		if l, err := s.store.Listing(r.Context(), r.PathValue("id")); err == nil && l.Status != domain.ListingWithdrawn {
			page = withPreview(index, l, origin(r))
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(page)
	}
}

// origin is the scheme and host the visitor used, for absolute URLs in
// preview tags.
func origin(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// withPreview puts the listing's title, description and photo into the
// page head as Open Graph and Twitter card tags.
func withPreview(index []byte, l domain.Listing, base string) []byte {
	amount := fmt.Sprintf("about %g kg", l.EstKg)
	if l.Pieces > 0 {
		amount = fmt.Sprintf("%d pieces", l.Pieces)
		if l.Pieces == 1 {
			amount = "1 piece"
		}
	}
	title := "Free: " + l.Item
	desc := fmt.Sprintf("%s in %s, given away by %s. Ask for it on ResourceLocate.", amount, l.Area, l.ProviderName)
	if l.Status != domain.ListingOpen {
		desc = "Already promised to someone. Find more materials on ResourceLocate."
	}
	tags := []string{
		`<meta property="og:type" content="website" />`,
		`<meta property="og:site_name" content="ResourceLocate" />`,
		`<meta property="og:title" content="` + html.EscapeString(title) + `" />`,
		`<meta property="og:description" content="` + html.EscapeString(desc) + `" />`,
		`<meta property="og:url" content="` + html.EscapeString(base+"/l/"+l.ID) + `" />`,
	}
	card := "summary"
	if l.Photos > 0 {
		img := base + "/api/listings/" + l.ID + "/photos/0"
		tags = append(tags, `<meta property="og:image" content="`+html.EscapeString(img)+`" />`)
		card = "summary_large_image"
	}
	tags = append(tags, `<meta name="twitter:card" content="`+card+`" />`)

	page := bytes.Replace(index, []byte("<title>ResourceLocate</title>"),
		[]byte("<title>"+html.EscapeString(title)+" · ResourceLocate</title>"), 1)
	return bytes.Replace(page, []byte("</head>"), []byte(strings.Join(tags, "\n    ")+"\n  </head>"), 1)
}
