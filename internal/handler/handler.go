package handler

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/Pirathikaran/web-analyzer/internal/analyzer"
	"github.com/Pirathikaran/web-analyzer/internal/metrics"
)

type pageData struct {
	URL    string
	Result *analyzer.Result
	Error  string
}

type Handler struct {
	pool      *analyzer.Pool
	templates *template.Template
	metrics   *metrics.Metrics
	logger    *slog.Logger
}

func New(
	pool *analyzer.Pool,
	tmpl *template.Template,
	m *metrics.Metrics,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		pool:      pool,
		templates: tmpl,
		metrics:   m,
		logger:    logger,
	}
}

func (h *Handler) Index(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	rawURL := r.URL.Query().Get("url")
	if rawURL == "" {
		h.render(w, r, http.StatusOK, pageData{})
		return
	}

	h.runAnalysis(w, r, rawURL)
}

func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
		http.Error(w, "invalid content type", http.StatusUnsupportedMediaType)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)

	rawURL := r.FormValue("url")

	if err := analyzer.ValidateURL(rawURL); err != nil {
		h.metrics.RequestsTotal.WithLabelValues("validation_error").Inc()
		h.render(w, r, http.StatusUnprocessableEntity, pageData{
			URL:   rawURL,
			Error: err.Error(),
		})
		return
	}

	http.Redirect(w, r, "/?url="+url.QueryEscape(rawURL), http.StatusSeeOther)
}

func (h *Handler) runAnalysis(w http.ResponseWriter, r *http.Request, rawURL string) {
	start := time.Now()

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := h.pool.Submit(ctx, rawURL)
	duration := time.Since(start).Seconds()

	if errors.Is(err, analyzer.ErrQueueFull) {
		h.metrics.RequestsTotal.WithLabelValues("queue_full").Inc()
		http.Error(w, "server busy, try again later", http.StatusServiceUnavailable)
		return
	}

	if err != nil {
		h.metrics.AnalysisErrors.Inc()
		h.metrics.RequestsTotal.WithLabelValues("error").Inc()
		h.metrics.RequestDuration.WithLabelValues("error").Observe(duration)

		h.logger.WarnContext(r.Context(), "analysis failed", "url", rawURL, "error", err)
		h.render(w, r, http.StatusBadGateway, pageData{URL: rawURL, Error: friendlyError(err)})
		return
	}

	h.metrics.RequestsTotal.WithLabelValues("success").Inc()
	h.metrics.RequestDuration.WithLabelValues("success").Observe(duration)
	h.render(w, r, http.StatusOK, pageData{URL: rawURL, Result: result})
}

func friendlyError(err error) string {
	var httpErr *analyzer.HTTPError
	if errors.As(err, &httpErr) {
		switch httpErr.Code {
		case 999:
			return "This website blocks automated access (bot protection). It cannot be analyzed."
		case 401, 403:
			return "Access denied. This website requires authentication or blocks external requests."
		case 404:
			return "Page not found (404). Please check the URL and try again."
		case 429:
			return "Too many requests. This website is rate-limiting access. Try again later."
		case 500, 502, 503, 504:
			return "The target website is currently unavailable (server error). Try again later."
		default:
			return httpErr.Error()
		}
	}

	raw := err.Error()

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "Website not found. The domain does not exist or cannot be resolved. Please check the URL."
	}

	switch {
	case strings.Contains(raw, "no such host"):
		return "Website not found. The domain does not exist or cannot be resolved. Please check the URL."
	case strings.Contains(raw, "connection refused"):
		return "Connection refused. The website is not accepting connections on that address."
	case strings.Contains(raw, "i/o timeout"), strings.Contains(raw, "context deadline exceeded"):
		return "Request timed out. The website took too long to respond."
	case strings.Contains(raw, "connection reset"):
		return "The connection was reset by the target server. The website may be blocking requests."
	case strings.Contains(raw, "no route to host"):
		return "Unable to reach the website. The host is unreachable from this server."
	case strings.Contains(raw, "certificate"), strings.Contains(raw, "tls"):
		return "SSL/TLS error. The website has an invalid or untrusted certificate."
	}

	return "Unable to analyze the URL. Please check it is correct and the website is publicly accessible."
}

func (h *Handler) render(w http.ResponseWriter, _ *http.Request, status int, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		h.logger.Error("template render failed", "error", err)
	}
}
