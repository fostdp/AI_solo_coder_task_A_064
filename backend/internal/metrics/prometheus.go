package metrics

import "github.com/prometheus/client_golang/prometheus"

var (
	TelemetryReceived = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "power_twin_telemetry_received_total",
			Help: "Total number of telemetry messages received",
		},
		[]string{"device_type"},
	)

	TelemetryProcessed = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "power_twin_telemetry_processed_total",
			Help: "Total number of telemetry messages processed",
		},
		[]string{"device_type"},
	)

	InfluxDBWriteDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "power_twin_influxdb_write_duration_seconds",
			Help:    "Duration of InfluxDB batch writes",
			Buckets: prometheus.DefBuckets,
		},
	)

	InfluxDBWriteErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "power_twin_influxdb_write_errors_total",
			Help: "Total number of InfluxDB write errors",
		},
	)

	PowerFlowCalcDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "power_twin_powerflow_calc_duration_seconds",
			Help:    "Duration of power flow calculations",
			Buckets: prometheus.DefBuckets,
		},
	)

	PowerFlowConverged = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "power_twin_powerflow_converged_total",
			Help: "Total power flow calculations by convergence status",
		},
		[]string{"converged"},
	)

	N1AnalysisDuration = prometheus.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "power_twin_n1_analysis_duration_seconds",
			Help:    "Duration of N-1 analysis",
			Buckets: prometheus.DefBuckets,
		},
	)

	N1Violations = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "power_twin_n1_violations_current",
			Help: "Current number of N-1 violations",
		},
	)

	AlarmsTriggered = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "power_twin_alarms_triggered_total",
			Help: "Total alarms triggered by level",
		},
		[]string{"level"},
	)

	AlarmsActive = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "power_twin_alarms_active_current",
			Help: "Current active alarms by level",
		},
		[]string{"level"},
	)

	MQTTMessagesPublished = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "power_twin_mqtt_published_total",
			Help: "Total MQTT messages published",
		},
		[]string{"topic"},
	)

	MQTTPublishErrors = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "power_twin_mqtt_publish_errors_total",
			Help: "Total MQTT publish errors",
		},
	)

	WebSocketClients = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "power_twin_websocket_clients_current",
			Help: "Current number of WebSocket clients",
		},
	)

	ChannelBacklog = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "power_twin_channel_backlog_current",
			Help: "Current backlog of Go channels",
		},
		[]string{"channel"},
	)
)

func init() {
	prometheus.MustRegister(
		TelemetryReceived,
		TelemetryProcessed,
		InfluxDBWriteDuration,
		InfluxDBWriteErrors,
		PowerFlowCalcDuration,
		PowerFlowConverged,
		N1AnalysisDuration,
		N1Violations,
		AlarmsTriggered,
		AlarmsActive,
		MQTTMessagesPublished,
		MQTTPublishErrors,
		WebSocketClients,
		ChannelBacklog,
	)
}
