package main

import (
	"html/template"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/Pirathikaran/web-analyzer/internal/analyzer"
	"github.com/Pirathikaran/web-analyzer/internal/handler"
	"github.com/Pirathikaran/web-analyzer/internal/metrics"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	tmpl := template.Must(
		template.New("").
			Funcs(template.FuncMap{"upper": strings_ToUpper}).
			ParseGlob("web/templates/*.html"),
	)

	httpClient := &http.Client{
		Timeout: 20 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	workers  := getEnvInt("POOL_WORKERS", 20)
	queueSize := getEnvInt("POOL_QUEUE", 1000)
	semSize  := getEnvInt("LINK_SEM", 50)

	globalSem := make(chan struct{}, semSize)
	a := analyzer.New(httpClient, logger, globalSem)
	pool := analyzer.NewPool(a, workers, queueSize)
	m := metrics.New()
	h := handler.New(pool, tmpl, m, logger)

	mux := http.NewServeMux()
	mux.HandleFunc("/", h.Index)
	mux.HandleFunc("/analyze", h.Analyze)
	mux.Handle("/metrics", promhttp.HandlerFor(m.Registry, promhttp.HandlerOpts{}))

	go func() {
		debugAddr := "127.0.0.1:6060"
		logger.Info("pprof server listening", "addr", debugAddr)
		if err := http.ListenAndServe(debugAddr, http.DefaultServeMux); err != nil {
			logger.Error("pprof server error", "error", err)
		}
	}()

	logged := handler.Logging(logger, handler.Recover(logger, handler.RateLimit(mux)))

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      logged,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 35 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logger.Info("server listening", "addr", srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		logger.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func getEnvInt(key string, defaultVal int) int {
	if s := os.Getenv(key); s != "" {
		if v, err := strconv.Atoi(s); err == nil && v > 0 {
			return v
		}
	}
	return defaultVal
}

func strings_ToUpper(s string) string {
	result := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'a' && c <= 'z' {
			c -= 32
		}
		result[i] = c
	}
	return string(result)
}
