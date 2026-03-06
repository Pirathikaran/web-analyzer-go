package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Metrics struct {
	RequestsTotal   *prometheus.CounterVec
	RequestDuration *prometheus.HistogramVec
	AnalysisErrors  prometheus.Counter
	Registry        *prometheus.Registry
}

func New() *Metrics {
	reg := prometheus.NewRegistry()

	requestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "web_analyzer_requests_total",
			Help: "Total number of analysis requests by status.",
		},
		[]string{"status"},
	)

	requestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "web_analyzer_request_duration_seconds",
			Help:    "Histogram of analysis request latencies.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)

	analysisErrors := prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "web_analyzer_errors_total",
			Help: "Total number of analysis errors.",
		},
	)

	reg.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		requestsTotal,
		requestDuration,
		analysisErrors,
	)

	return &Metrics{
		RequestsTotal:   requestsTotal,
		RequestDuration: requestDuration,
		AnalysisErrors:  analysisErrors,
		Registry:        reg,
	}
}
