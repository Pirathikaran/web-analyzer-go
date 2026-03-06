package analyzer

import (
	"strings"
	"testing"

	"golang.org/x/net/html"
)

func parse(t *testing.T, src string) *html.Node {
	t.Helper()
	doc, err := html.Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("html.Parse: %v", err)
	}
	return doc
}

func TestExtractTitle(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`<html><head><title>Hello World</title></head><body></body></html>`, "Hello World"},
		{`<html><head><title>  Trim Me  </title></head><body></body></html>`, "Trim Me"},
		{`<html><head></head><body></body></html>`, ""},
	}
	for _, tc := range cases {
		got := extractTitle(parse(t, tc.src))
		if got != tc.want {
			t.Errorf("extractTitle = %q, want %q", got, tc.want)
		}
	}
}

func TestCollectHeadings(t *testing.T) {
	const src = `<html><body>
		<h1>One</h1>
		<h2>Two A</h2><h2>Two B</h2>
		<h3>Three</h3>
	</body></html>`

	counts := make(map[string]int)
	collectHeadings(parse(t, src), counts)

	expected := map[string]int{"h1": 1, "h2": 2, "h3": 1}
	for tag, want := range expected {
		if counts[tag] != want {
			t.Errorf("%s count = %d, want %d", tag, counts[tag], want)
		}
	}
}

func TestHasLoginForm(t *testing.T) {
	cases := []struct {
		name string
		src  string
		want bool
	}{
		{
			"password field present",
			`<html><body><form><input type="text"><input type="password"></form></body></html>`,
			true,
		},
		{
			"password field absent",
			`<html><body><form><input type="text"><button>Search</button></form></body></html>`,
			false,
		},
		{
			"no form at all",
			`<html><body><p>Nothing here</p></body></html>`,
			false,
		},
		{
			"password outside form",
			`<html><body><input type="password"><form><input type="text"></form></body></html>`,
			false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := hasLoginForm(parse(t, tc.src))
			if got != tc.want {
				t.Errorf("hasLoginForm = %v, want %v", got, tc.want)
			}
		})
	}
}
