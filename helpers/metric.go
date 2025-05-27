package helpers

import (
	"github.com/prometheus/client_golang/prometheus"
)

// HttpRequestsTotal is a counter for HTTP requests total
var HttpRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_requests_total",
		Help: "Total number of HTTP requests",
		ConstLabels: map[string]string{
			"service": "Auth Service",
		},
	},
	[]string{"path", "method"},
)

// HttpRequestErrors is a counter for HTTP requests errors
var HttpRequestErrors = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Name: "http_request_errors_total",
		Help: "Total number of HTTP requests errors",
		ConstLabels: map[string]string{
			"service": "Auth Service",
		},
	},
	[]string{"path", "method"},
)

// CollectHttpMetrics collects metrics from the HTTP requests
func CollectHttpMetrics() {
	prometheus.MustRegister(HttpRequestsTotal, HttpRequestErrors)
}
