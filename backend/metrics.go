package main

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	httpRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	httpRequestsInFlight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "http_requests_in_flight",
			Help: "Current number of HTTP requests being served",
		},
	)

	authInitTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "auth_init_total",
			Help: "Total number of auth init requests",
		},
	)

	authCompleteTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_complete_total",
			Help: "Total number of auth complete attempts",
		},
		[]string{"status"},
	)

	authVerifyTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "auth_verify_total",
			Help: "Total number of auth verify requests",
		},
		[]string{"valid"},
	)

	authSessionsActive = promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "auth_sessions_active",
			Help: "Number of active sessions in database",
		},
		func() float64 {
			if sessionsDB == nil {
				return 0
			}
			return float64(sessionCount())
		},
	)

	newsRequestsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "news_requests_total",
			Help: "Total number of news API requests",
		},
	)

	fileDownloadBytesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "file_download_bytes_total",
			Help: "Total bytes of files downloaded",
		},
	)

	fileDownloadsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "file_downloads_total",
			Help: "Total number of file downloads",
		},
	)
)

func endpointFromPath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}
	path = strings.TrimPrefix(path, "/")
	parts := strings.SplitN(path, "/", 2)
	if len(parts) < 2 {
		return "/" + parts[0]
	}
	if parts[0] == "api" {
		return "/api/" + strings.SplitN(parts[1], "/", 2)[0]
	}
	if parts[0] == "files" {
		return "/files"
	}
	return "/" + parts[0]
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/metrics" {
			next.ServeHTTP(w, r)
			return
		}

		httpRequestsInFlight.Inc()
		defer httpRequestsInFlight.Dec()

		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(r.Method, endpointFromPath(r.URL.Path)))
		next.ServeHTTP(rec, r)
		timer.ObserveDuration()

		status := strconv.Itoa(rec.status)
		httpRequestsTotal.WithLabelValues(r.Method, endpointFromPath(r.URL.Path), status).Inc()
	})
}

func metricsHandler() http.Handler {
	return promhttp.Handler()
}
