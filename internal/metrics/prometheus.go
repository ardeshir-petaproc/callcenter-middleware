package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

// PrometheusMetrics encapsulates application metrics collectors.
type PrometheusMetrics struct {
	wsConnected   prometheus.Gauge
	activeCalls   prometheus.Gauge
	dtmfTotal     *prometheus.CounterVec
	handoffsTotal *prometheus.CounterVec
	ariLatency    *prometheus.HistogramVec
	outcomes      *prometheus.CounterVec
	waitSeconds   *prometheus.HistogramVec
	queueWaiting  *prometheus.GaugeVec
	agents        *prometheus.GaugeVec
	aiSessions    prometheus.Gauge
}

// NewPrometheusMetrics initializes and registers Prometheus collectors.
func NewPrometheusMetrics() *PrometheusMetrics {
	return NewPrometheusMetricsWith(prometheus.DefaultRegisterer)
}

// NewPrometheusMetricsWith registers collectors on a custom registerer.
func NewPrometheusMetricsWith(reg prometheus.Registerer) *PrometheusMetrics {
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
		outcomes: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "callcenter_call_outcomes_total",
			Help: "Finished calls by outcome.",
		}, []string{"outcome"}),
		waitSeconds: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name:    "callcenter_queue_wait_seconds",
			Help:    "Time callers waited in queue before an agent answered.",
			Buckets: []float64{5, 10, 20, 30, 60, 120, 300, 600},
		}, []string{"queue"}),
		queueWaiting: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "callcenter_queue_waiting_calls",
			Help: "Callers currently waiting per queue.",
		}, []string{"queue"}),
		agents: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "callcenter_agents",
			Help: "Agents by presence status.",
		}, []string{"status"}),
		aiSessions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "callcenter_ai_sessions",
			Help: "Active AI voice agent sessions.",
		}),
	}
	reg.MustRegister(m.wsConnected, m.activeCalls, m.dtmfTotal, m.handoffsTotal, m.ariLatency,
		m.outcomes, m.waitSeconds, m.queueWaiting, m.agents, m.aiSessions)
	return m
}

func (m *PrometheusMetrics) SetWSConnected(connected bool) {
	if connected {
		m.wsConnected.Set(1)
	} else {
		m.wsConnected.Set(0)
	}
}

func (m *PrometheusMetrics) IncActiveCalls() { m.activeCalls.Inc() }

func (m *PrometheusMetrics) DecActiveCalls() { m.activeCalls.Dec() }

func (m *PrometheusMetrics) RecordDTMF(digit string, outcome string) {
	m.dtmfTotal.WithLabelValues(digit, outcome).Inc()
}

func (m *PrometheusMetrics) RecordHandoff(queueExt string, reason string) {
	m.handoffsTotal.WithLabelValues(queueExt, reason).Inc()
}

func (m *PrometheusMetrics) RecordARILatency(operation string, durationSeconds float64) {
	m.ariLatency.WithLabelValues(operation).Observe(durationSeconds)
}

func (m *PrometheusMetrics) RecordOutcome(outcome string) { m.outcomes.WithLabelValues(outcome).Inc() }

func (m *PrometheusMetrics) ObserveWait(queue string, seconds float64) {
	m.waitSeconds.WithLabelValues(queue).Observe(seconds)
}

func (m *PrometheusMetrics) SetQueueWaiting(queue string, n int) {
	m.queueWaiting.WithLabelValues(queue).Set(float64(n))
}

func (m *PrometheusMetrics) SetAgentsByStatus(counts map[string]int) {
	for _, st := range []string{"offline", "available", "ringing", "busy", "wrapup", "paused"} {
		m.agents.WithLabelValues(st).Set(float64(counts[st]))
	}
}

func (m *PrometheusMetrics) SetAISessions(n int) { m.aiSessions.Set(float64(n)) }
