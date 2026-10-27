package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics encapsulates application metrics collectors.
type PrometheusMetrics struct {
	wsConnected     prometheus.Gauge
	activeCalls     prometheus.Gauge
	dtmfTotal       *prometheus.CounterVec
	handoffsTotal   *prometheus.CounterVec
	ariLatency      *prometheus.HistogramVec
}

// NewPrometheusMetrics initializes and registers Prometheus collectors.
func NewPrometheusMetrics() *PrometheusMetrics {
	m := &PrometheusMetrics{
		wsConnected: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "callcenter_ari_ws_connected",
			Help: "1 if connected to Asterisk ARI WebSocket, 0 if disconnected.",
		}),
		activeCalls: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "callcenter_active_stasis_calls",
			Help: "Current number of calls actively controlled by Go middleware.",
		}),
		dtmfTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "callcenter_ivr_dtmf_total",
			Help: "Counts key presses received in IVR.",
		}, []string{"digit", "outcome"}),
		handoffsTotal: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "callcenter_queue_handoffs_total",
			Help: "Tracks queue handoffs by target queue and reason.",
		}, []string{"queue", "reason"}),
		ariLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "callcenter_ari_rest_duration_seconds",
			Help:    "Latency of ARI HTTP REST operations.",
			Buckets: []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
		}, []string{"operation"}),
	}

	prometheus.MustRegister(m.wsConnected, m.activeCalls, m.dtmfTotal, m.handoffsTotal, m.ariLatency)
	return m
}

func (m *PrometheusMetrics) SetWSConnected(connected bool) {
	if connected {
		m.wsConnected.Set(1)
	} else {
		m.wsConnected.Set(0)
	}
}

func (m *PrometheusMetrics) IncActiveCalls() {
	m.activeCalls.Inc()
}

func (m *PrometheusMetrics) DecActiveCalls() {
	m.activeCalls.Dec()
}

func (m *PrometheusMetrics) RecordDTMF(digit string, outcome string) {
	m.dtmfTotal.WithLabelValues(digit, outcome).Inc()
}

func (m *PrometheusMetrics) RecordHandoff(queueExt string, reason string) {
	m.handoffsTotal.WithLabelValues(queueExt, reason).Inc()
}

func (m *PrometheusMetrics) RecordARILatency(operation string, durationSeconds float64) {
	m.ariLatency.WithLabelValues(operation).Observe(durationSeconds)
}
