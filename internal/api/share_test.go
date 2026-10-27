package api

import (
	"strings"
	"testing"

	"github.com/KeKe45/resourcelocate/internal/domain"
)

func TestPreviewTagsAreEscaped(t *testing.T) {
	index := []byte("<html><head><title>ResourceLocate</title></head><body></body></html>")
	l := domain.Listing{ID: "abc", Item: `Chairs "<script>"`, Pieces: 6, Area: "Kabwata", ProviderName: "A & B", Status: domain.ListingOpen, Photos: 1}
	page := string(withPreview(index, l, "https://example.org"))
	for _, want := range []string{
		`<title>Free: Chairs &#34;&lt;script&gt;&#34; · ResourceLocate</title>`,
		`content="6 pieces in Kabwata, given away by A &amp; B. Ask for it on ResourceLocate."`,
		`<meta property="og:image" content="https://example.org/api/listings/abc/photos/0" />`,
		`content="summary_large_image"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("page lacks %s\n%s", want, page)
		}
	}
	if strings.Contains(page, "<script>") {
		t.Error("item text was not escaped")
	}
}
