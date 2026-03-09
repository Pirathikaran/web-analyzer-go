package analyzer_test

import (
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Pirathikaran/web-analyzer/internal/analyzer"
)

// newTestAnalyzer creates an Analyzer whose HTTP client points at the given test server.
func newTestAnalyzer(t *testing.T) (*analyzer.Analyzer, *http.Client) {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	client := &http.Client{}
	globalSem := make(chan struct{}, 10)
	return analyzer.New(client, logger, globalSem), client
}

// ------- ValidateURL ---------------------------------------------------------

func TestValidateURL(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid http", "http://example.com", false},
		{"valid https", "https://example.com/path?q=1", false},
		{"empty string", "", true},
		{"missing scheme", "example.com", true},
		{"ftp scheme", "ftp://example.com", true},
		{"whitespace only", "   ", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := analyzer.ValidateURL(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tc.input, err, tc.wantErr)
			}
		})
	}
}

// ------- Analyze end-to-end via httptest ------------------------------------

const html5Page = `<!DOCTYPE html>
<html><head><title>Test Page</title></head>
<body>
  <h1>Heading One</h1>
  <h2>Heading Two A</h2>
  <h2>Heading Two B</h2>
  <a href="/internal">internal</a>
  <a href="https://external.example.org/page">external</a>
  <form>
    <input type="text" name="user">
    <input type="password" name="pass">
  </form>
</body></html>`

func TestAnalyze_HTML5Page(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// HEAD requests for link checking
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(html5Page))
	}))
	defer srv.Close()

	a, _ := newTestAnalyzer(t)
	result, err := a.Analyze(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.HTMLVersion != "HTML5" {
		t.Errorf("HTMLVersion = %q, want HTML5", result.HTMLVersion)
	}
	if result.Title != "Test Page" {
		t.Errorf("Title = %q, want 'Test Page'", result.Title)
	}
	if result.Headings["h1"] != 1 {
		t.Errorf("h1 count = %d, want 1", result.Headings["h1"])
	}
	if result.Headings["h2"] != 2 {
		t.Errorf("h2 count = %d, want 2", result.Headings["h2"])
	}
	if !result.HasLoginForm {
		t.Error("HasLoginForm = false, want true")
	}
	// 1 internal (/internal resolved to srv.URL/internal) + 1 external
	if result.InternalLinks != 1 {
		t.Errorf("InternalLinks = %d, want 1", result.InternalLinks)
	}
	if result.ExternalLinks != 1 {
		t.Errorf("ExternalLinks = %d, want 1", result.ExternalLinks)
	}
}

func TestAnalyze_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "gone", http.StatusGone)
	}))
	defer srv.Close()

	a, _ := newTestAnalyzer(t)
	_, err := a.Analyze(context.Background(), srv.URL)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	var httpErr *analyzer.HTTPError
	if !isHTTPError(err, &httpErr) {
		t.Fatalf("expected *HTTPError, got %T: %v", err, err)
	}
	if httpErr.Code != http.StatusGone {
		t.Errorf("HTTPError.Code = %d, want %d", httpErr.Code, http.StatusGone)
	}
}

// isHTTPError is a simple errors.As replacement to avoid importing "errors" above.
func isHTTPError(err error, target **analyzer.HTTPError) bool {
	if e, ok := err.(*analyzer.HTTPError); ok {
		*target = e
		return true
	}
	return false
}

func TestAnalyze_NoLoginForm(t *testing.T) {
	const page = `<!DOCTYPE html><html><head><title>No Login</title></head>
<body><form><input type="text" name="search"><button>Go</button></form></body></html>`

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte(page))
	}))
	defer srv.Close()

	a, _ := newTestAnalyzer(t)
	result, err := a.Analyze(context.Background(), srv.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.HasLoginForm {
		t.Error("HasLoginForm = true, want false")
	}
}
