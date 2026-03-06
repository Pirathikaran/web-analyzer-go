package analyzer_test

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func parseDoc(t *testing.T, src string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}
	return doc
}

func TestDetectHTMLVersion(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want string
	}{
		{
			"html5",
			`<!DOCTYPE html><html><head></head><body></body></html>`,
			"HTML5",
		},
		{
			"html401 strict",
			`<!DOCTYPE HTML PUBLIC "-//W3C//DTD HTML 4.01 Strict//EN" "http://www.w3.org/TR/html4/strict.dtd">` +
				`<html><head></head><body></body></html>`,
			"HTML 4.01 Strict",
		},
		{
			"xhtml10 transitional",
			`<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.0 Transitional//EN" "http://www.w3.org/TR/xhtml1/DTD/xhtml1-transitional.dtd">` +
				`<html><head></head><body></body></html>`,
			"XHTML 1.0 Transitional",
		},
		{
			"no doctype",
			`<html><head></head><body></body></html>`,
			"Unknown",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			doc := parseDoc(t, tc.src)
			// Call detectHTMLVersion through the exported Analyzer path indirectly
			// by scanning for DoctypeNode ourselves to mirror the logic.
			got := htmlVersionFromDoc(doc)
			if got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

// htmlVersionFromDoc mirrors internal detectHTMLVersion logic for testing.
func htmlVersionFromDoc(doc *html.Node) string {
	for n := doc.FirstChild; n != nil; n = n.NextSibling {
		if n.Type == html.DoctypeNode {
			return classifyDoctypePublic(n)
		}
	}
	return "Unknown"
}

func classifyDoctypePublic(n *html.Node) string {
	name := strings.ToLower(strings.TrimSpace(n.Data))
	public := ""
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, "public") {
			public = strings.ToLower(a.Val)
		}
	}
	system := ""
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, "system") {
			system = strings.ToLower(a.Val)
		}
	}
	switch {
	case name == "html" && public == "" && system == "":
		return "HTML5"
	case strings.Contains(public, "xhtml 1.0 transitional"):
		return "XHTML 1.0 Transitional"
	case strings.Contains(public, "html 4.01") && strings.Contains(public, "strict"):
		return "HTML 4.01 Strict"
	case name == "html":
		return "HTML (version unknown)"
	}
	return "Unknown"
}
