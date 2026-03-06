package analyzer

import (
	"net/url"
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func mustBase(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	return u
}

func parseHTML(t *testing.T, src string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}
	return doc
}

func TestCollectLinks_InternalExternal(t *testing.T) {
	const src = `<!DOCTYPE html><html><body>
		<a href="/about">About</a>
		<a href="https://example.com/contact">Contact</a>
		<a href="https://other.org/page">External</a>
		<a href="mailto:x@y.com">Mail</a>
		<a href="#section">Fragment</a>
	</body></html>`

	base := mustBase(t, "https://example.com")
	doc := parseHTML(t, src)
	internal, external := collectLinks(doc, base)

	if len(internal) != 2 {
		t.Errorf("internal = %d, want 2 (got %v)", len(internal), internal)
	}
	if len(external) != 1 {
		t.Errorf("external = %d, want 1 (got %v)", len(external), external)
	}
}

func TestCollectLinks_Deduplication(t *testing.T) {
	const src = `<!DOCTYPE html><html><body>
		<a href="/page">Page</a>
		<a href="/page">Page again</a>
		<a href="/page#anchor">Page with anchor</a>
	</body></html>`

	base := mustBase(t, "https://example.com")
	doc := parseHTML(t, src)
	internal, external := collectLinks(doc, base)

	if len(internal) != 1 {
		t.Errorf("internal after dedup = %d, want 1 (got %v)", len(internal), internal)
	}
	if len(external) != 0 {
		t.Errorf("external = %d, want 0", len(external))
	}
}

func TestResolveURL(t *testing.T) {
	base := mustBase(t, "https://example.com/dir/")
	cases := []struct {
		href    string
		wantOK  bool
		wantAbs string
	}{
		{"/page", true, "https://example.com/page"},
		{"./rel", true, "https://example.com/dir/rel"},
		{"https://other.com/x", true, "https://other.com/x"},
		{"", false, ""},
		{"#frag", false, ""},
		{"mailto:a@b.com", false, ""},
		{"javascript:void(0)", false, ""},
	}
	for _, tc := range cases {
		abs, ok := resolveURL(base, tc.href)
		if ok != tc.wantOK {
			t.Errorf("resolveURL(%q) ok=%v, want %v", tc.href, ok, tc.wantOK)
			continue
		}
		if ok && abs != tc.wantAbs {
			t.Errorf("resolveURL(%q) = %q, want %q", tc.href, abs, tc.wantAbs)
		}
	}
}

func TestIsInternal(t *testing.T) {
	base := mustBase(t, "https://example.com")
	cases := []struct {
		target string
		want   bool
	}{
		{"https://example.com/page", true},
		{"https://EXAMPLE.COM/other", true},
		{"https://sub.example.com/", false},
		{"https://other.org/", false},
	}
	for _, tc := range cases {
		got := isInternal(base, tc.target)
		if got != tc.want {
			t.Errorf("isInternal(%q) = %v, want %v", tc.target, got, tc.want)
		}
	}
}
