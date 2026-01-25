package controllers

import "github.com/prometheus/client_golang/prometheus"

func RegisterMetrics(reg prometheus.Registerer) {
	reg.MustRegister(sshAuthTotal)
	reg.MustRegister(sshSessionDuration)
	reg.MustRegister(sshSessionsInFlight)
}

var (
	sshAuthTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "netcon",
			Subsystem: "nclet",
			Name:      "ssh_auth_total",
			Help:      "Total number of SSH authentication attempts",
		},
		[]string{"result"},
	)

	sshAuthTotalSucceeded = sshAuthTotal.WithLabelValues("succeeded")
	sshAuthTotalFailed    = sshAuthTotal.WithLabelValues("failed")

	sshSessionDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Namespace: "netcon",
			Subsystem: "nclet",
			Name:      "ssh_session_duration_seconds",
			Help:      "Duration of SSH sessions in seconds",
			Buckets: []float64{
				10, 30, 60, 180, 300, 600, 900, 1200, 1800, 2700, 3600, 7200, 10800, 14400, 21600, 28800, 36000, 43200,
			},
		},
	)

	sshSessionsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "netcon",
			Subsystem: "nclet",
			Name:      "ssh_sessions_in_flight",
			Help:      "Current number of in-flight SSH sessions",
		},
	)
)
