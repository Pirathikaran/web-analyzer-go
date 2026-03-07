package integration_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestIntegration_IndexPage(t *testing.T) {
	app := buildApp(t)
	defer app.Close()

	resp, err := http.Get(app.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(body, "Web Page Analyzer") {
		t.Errorf("expected page title in body, got:\n%s", body)
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/html") {
		t.Errorf("Content-Type = %q, want text/html", ct)
	}
}

func TestIntegration_NotFound(t *testing.T) {
	app := buildApp(t)
	defer app.Close()

	resp, err := http.Get(app.URL + "/no-such-route")
	if err != nil {
		t.Fatalf("GET /no-such-route: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("status = %d, want 404", resp.StatusCode)
	}
}
