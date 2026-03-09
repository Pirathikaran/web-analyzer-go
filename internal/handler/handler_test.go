package handler_test

import (
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/Pirathikaran/web-analyzer/internal/analyzer"
	"github.com/Pirathikaran/web-analyzer/internal/handler"
	"github.com/Pirathikaran/web-analyzer/internal/metrics"
)

const tmplSrc = `{{define "index.html"}}` +
	`{{if .Error}}ERROR:{{.Error}}{{end}}` +
	`{{if .Result}}RESULT:{{.Result.Title}}{{end}}` +
	`{{end}}`

func newHandler(t *testing.T) http.Handler {
	t.Helper()
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	tmpl := template.Must(template.New("").Parse(tmplSrc))
	client := &http.Client{}
	globalSem := make(chan struct{}, 10)
	a := analyzer.New(client, logger, globalSem)
	pool := analyzer.NewPool(a, 5, 100)
	m := metrics.New()
	h := handler.New(pool, tmpl, m, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.Index)
	mux.HandleFunc("/analyze", h.Analyze)
	return mux
}

func TestIndex_GET(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestIndex_NotFound(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/no-such-path", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", rr.Code)
	}
}

func TestAnalyze_InvalidURL(t *testing.T) {
	h := newHandler(t)

	form := url.Values{"url": {"not-a-url"}}
	req := httptest.NewRequest(http.MethodPost, "/analyze", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Errorf("status = %d, want 422", rr.Code)
	}
	if !strings.Contains(rr.Body.String(), "ERROR:") {
		t.Errorf("body should contain error message, got: %s", rr.Body.String())
	}
}

func TestAnalyze_ValidURL(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte(`<!DOCTYPE html><html><head><title>Test</title></head><body></body></html>`))
	}))
	defer backend.Close()

	h := newHandler(t)

	form := url.Values{"url": {backend.URL}}
	req := httptest.NewRequest(http.MethodPost, "/analyze", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusSeeOther {
		t.Errorf("POST status = %d, want 303", rr.Code)
	}

	location := rr.Header().Get("Location")
	req2 := httptest.NewRequest(http.MethodGet, location, nil)
	rr2 := httptest.NewRecorder()
	h.ServeHTTP(rr2, req2)

	if rr2.Code != http.StatusOK {
		t.Errorf("GET status = %d, want 200", rr2.Code)
	}
	if !strings.Contains(rr2.Body.String(), "RESULT:Test") {
		t.Errorf("body should contain result title, got: %s", rr2.Body.String())
	}
}

func TestAnalyze_MethodNotAllowed(t *testing.T) {
	h := newHandler(t)
	req := httptest.NewRequest(http.MethodGet, "/analyze", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rr.Code)
	}
}

func TestAnalyze_UnreachableURL(t *testing.T) {
	h := newHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/?url=http://127.0.0.1:19999/no-such-server", nil)
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)

	if !strings.Contains(rr.Body.String(), "ERROR:") {
		t.Errorf("expected error in body for unreachable URL, got: %s", rr.Body.String())
	}
}
