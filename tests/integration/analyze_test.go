package integration_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIntegration_AnalyzeMethodNotAllowed(t *testing.T) {
	app := buildApp(t)
	defer app.Close()

	resp, err := http.Get(app.URL + "/analyze")
	if err != nil {
		t.Fatalf("GET /analyze: %v", err)
	}
	resp.Body.Close()

	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", resp.StatusCode)
	}
}

func TestIntegration_AnalyzeEmptyURL(t *testing.T) {
	app := buildApp(t)
	defer app.Close()

	resp := postForm(t, app.URL, "")
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", resp.StatusCode)
	}
	if !strings.Contains(body, "URL must not be empty") {
		t.Errorf("expected validation error in body, got:\n%s", body)
	}
}

func TestIntegration_AnalyzeInvalidURL(t *testing.T) {
	app := buildApp(t)
	defer app.Close()

	resp := postForm(t, app.URL, "not-a-url")
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", resp.StatusCode)
	}
	if !strings.Contains(body, "invalid URL") {
		t.Errorf("expected invalid URL error in body, got:\n%s", body)
	}
}

func TestIntegration_AnalyzeSuccess(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head><title>Integration Test Page</title></head>
<body>
  <h1>Hello</h1>
  <h2>Sub</h2>
  <a href="/internal">internal</a>
  <a href="https://external.example.com">external</a>
</body>
</html>`)
	}))
	defer backend.Close()

	app := buildApp(t)
	defer app.Close()

	resp := postForm(t, app.URL, backend.URL)
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200\nbody: %s", resp.StatusCode, body)
	}
	if !strings.Contains(body, "Integration Test Page") {
		t.Errorf("expected page title in results, got:\n%s", body)
	}
	if !strings.Contains(body, "HTML5") {
		t.Errorf("expected HTML version in results, got:\n%s", body)
	}
	if !strings.Contains(body, "H1") {
		t.Errorf("expected H1 heading chip, got:\n%s", body)
	}
}

func TestIntegration_AnalyzeLoginFormDetected(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Login</title></head>
<body>
  <form method="POST" action="/login">
    <input type="text" name="username">
    <input type="password" name="password">
    <button>Sign in</button>
  </form>
</body></html>`)
	}))
	defer backend.Close()

	app := buildApp(t)
	defer app.Close()

	resp := postForm(t, app.URL, backend.URL)
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(body, "badge-yes") {
		t.Errorf("expected login-form badge-yes in body, got:\n%s", body)
	}
}

func TestIntegration_AnalyzeBackendReturns4xx(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer backend.Close()

	app := buildApp(t)
	defer app.Close()

	resp := postForm(t, app.URL, backend.URL)
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", resp.StatusCode)
	}
	if !strings.Contains(body, "HTTP 403") {
		t.Errorf("expected HTTP 403 error in body, got:\n%s", body)
	}
}

func TestIntegration_AnalyzeUnreachableURL(t *testing.T) {
	app := buildApp(t)
	defer app.Close()

	resp := postForm(t, app.URL, "http://127.0.0.1:19999/no-such-server")
	body := readBody(t, resp)

	if !strings.Contains(body, "class=\"error\"") {
		t.Errorf("expected error div in body for unreachable URL, got:\n%s", body)
	}
}

func TestIntegration_HeadingsCountedCorrectly(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		fmt.Fprint(w, `<!DOCTYPE html>
<html><head><title>Headings Page</title></head>
<body>
  <h1>One</h1>
  <h1>Two</h1>
  <h2>Sub A</h2>
  <h3>Deep</h3>
</body></html>`)
	}))
	defer backend.Close()

	app := buildApp(t)
	defer app.Close()

	resp := postForm(t, app.URL, backend.URL)
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(body, "H1: 2") {
		t.Errorf("expected 'H1: 2' heading chip, got:\n%s", body)
	}
	if !strings.Contains(body, "H2: 1") {
		t.Errorf("expected 'H2: 1' heading chip, got:\n%s", body)
	}
}
