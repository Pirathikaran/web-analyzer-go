package integration_test

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Pirathikaran/web-analyzer/internal/handler"
)

func TestIntegration_PanicRecovery(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: slog.LevelError,
	}))

	panicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("deliberate test panic")
	})
	srv := httptest.NewServer(handler.Recover(logger, panicHandler))
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/")
	if err != nil {
		t.Fatalf("GET /: %v", err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Logf("failed to close response body: %v", err)
	}

	if resp.StatusCode != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500 after panic recovery", resp.StatusCode)
	}
}
