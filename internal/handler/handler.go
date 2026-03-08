package handler

import (
	"context"
	"errors"
	"html/template"
	"log/slog"
	"net/http"
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
	analyzer  *analyzer.Analyzer
	templates *template.Template
	metrics   *metrics.Metrics
	logger    *slog.Logger
}

func New(
	a *analyzer.Analyzer,
	tmpl *template.Template,
	m *metrics.Metrics,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		analyzer:  a,
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
	h.render(w, r, http.StatusOK, pageData{})
}

func (h *Handler) Analyze(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

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

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	result, err := h.analyzer.Analyze(ctx, rawURL)
	duration := time.Since(start).Seconds()

	if err != nil {
		h.metrics.AnalysisErrors.Inc()
		h.metrics.RequestsTotal.WithLabelValues("error").Inc()
		h.metrics.RequestDuration.WithLabelValues("error").Observe(duration)

		var httpErr *analyzer.HTTPError
		status := http.StatusBadGateway
		msg := err.Error()
		if errors.As(err, &httpErr) {
			msg = httpErr.Error()
			status = http.StatusBadGateway
		}

		h.logger.WarnContext(r.Context(), "analysis failed", "url", rawURL, "error", err)
		h.render(w, r, status, pageData{URL: rawURL, Error: msg})
		return
	}

	h.metrics.RequestsTotal.WithLabelValues("success").Inc()
	h.metrics.RequestDuration.WithLabelValues("success").Observe(duration)
	h.render(w, r, http.StatusOK, pageData{URL: rawURL, Result: result})
}

func (h *Handler) render(w http.ResponseWriter, _ *http.Request, status int, data pageData) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	if err := h.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		h.logger.Error("template render failed", "error", err)
	}
}
