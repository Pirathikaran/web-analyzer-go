package integration_test

import (
	"net/http"
	"strings"
	"testing"
)

func TestIntegration_MetricsEndpoint(t *testing.T) {
	app := buildApp(t)
	defer app.Close()

	postForm(t, app.URL, "not-a-url")

	resp, err := http.Get(app.URL + "/metrics")
	if err != nil {
		t.Fatalf("GET /metrics: %v", err)
	}
	body := readBody(t, resp)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
	if !strings.Contains(body, "web_analyzer_requests_total") {
		t.Errorf("expected prometheus metric in body, got:\n%s", body)
	}
}
