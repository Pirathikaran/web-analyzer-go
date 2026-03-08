package integration_test

import (
	"html/template"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Pirathikaran/web-analyzer/internal/analyzer"
	"github.com/Pirathikaran/web-analyzer/internal/handler"
	"github.com/Pirathikaran/web-analyzer/internal/metrics"
)

func buildApp(t *testing.T) *httptest.Server {
	t.Helper()

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	tmpl := template.Must(
		template.New("").
			Funcs(template.FuncMap{"upper": strings.ToUpper}).
			ParseGlob("../../web/templates/*.html"),
	)

	client := &http.Client{Timeout: 10 * time.Second}
	a := analyzer.New(client, logger)
	pool := analyzer.NewPool(a, 5, 100)
	m := metrics.New()
	h := handler.New(pool, tmpl, m, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.Index)
	mux.HandleFunc("/analyze", h.Analyze)
	mux.Handle("/metrics", promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))

	stack := handler.Logging(logger, handler.Recover(logger, mux))
	return httptest.NewServer(stack)
}

func postForm(t *testing.T, appURL, targetURL string) *http.Response {
	t.Helper()
	form := url.Values{"url": {targetURL}}
	resp, err := http.PostForm(appURL+"/analyze", form)
	if err != nil {
		t.Fatalf("POST /analyze: %v", err)
	}
	return resp
}

func readBody(t *testing.T, resp *http.Response) string {
	t.Helper()
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	return string(b)
}
