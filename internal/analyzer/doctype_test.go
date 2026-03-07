package analyzer

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func docFromSrc(t *testing.T, src string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}
	return doc
}

func TestClassifyDoctype_AllBranches(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			"xhtml11",
			`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd"><html></html>`,
			"XHTML 1.1",
		},
		{
			"xhtml10strict",
			`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Strict//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-strict.dtd"><html></html>`,
			"XHTML 1.0 Strict",
		},
		{
			"xhtml10frameset",
			`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Frameset//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-frameset.dtd"><html></html>`,
			"XHTML 1.0 Frameset",
		},
		{
			"html401transitional",
			`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Transitional//EN" "http://www.w3.org/TR/html4/loose.dtd"><html></html>`,
			"HTML 4.01 Transitional",
		},
		{
			"html401frameset",
			`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Frameset//EN" "http://www.w3.org/TR/html4/frameset.dtd"><html></html>`,
			"HTML 4.01 Frameset",
		},
		{
			"html40",
			`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.0//EN"><html></html>`,
			"HTML 4.0",
		},
		{
			"html32",
			`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 3.2 Final//EN"><html></html>`,
			"HTML 3.2",
		},
		{
			"html20",
			`<!DOCTYPE HTML PUBLIC "-//IETF//DTD HTML 2.0//EN"><html></html>`,
			"HTML 2.0",
		},
		{
			"html_version_unknown",
			`<!DOCTYPE html PUBLIC "-//Unknown//DTD//EN"><html></html>`,
			"HTML (version unknown)",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := docFromSrc(t, tc.src)
			got := detectHTMLVersion(doc)
			if got != tc.want {
				t.Errorf("detectHTMLVersion() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestHTTPError_Error(t *testing.T) {
	e := &HTTPError{Code: 404, Message: "Not Found"}
	want := "HTTP 404: Not Found"
	if got := e.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestResolveURL_TelAndExtra(t *testing.T) {
	base := mustBase(t, "https://example.com/")
	cases := []struct {
		href   string
		wantOK bool
	}{
		{"tel:+1234567890", false},
		{"  ", false},
	}
	for _, tc := range cases {
		_, ok := resolveURL(base, tc.href)
		if ok != tc.wantOK {
			t.Errorf("resolveURL(%q) ok=%v, want %v", tc.href, ok, tc.wantOK)
		}
	}
}
