package metrics_test

import (
	"testing"

	"github.com/Pirathikaran/web-analyzer/internal/metrics"
)

func TestNew_ReturnsValidMetrics(t *testing.T) {
	m := metrics.New()
	if m == nil {
		t.Fatal("New() returned nil")
	}
	if m.RequestsTotal == nil {
		t.Error("RequestsTotal is nil")
	}
	if m.RequestDuration == nil {
		t.Error("RequestDuration is nil")
	}
	if m.AnalysisErrors == nil {
		t.Error("AnalysisErrors is nil")
	}
	if m.Registry == nil {
		t.Error("Registry is nil")
	}
}
