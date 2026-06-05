package handler

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"power-twin-backend/internal/alarm"
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/powerflow"
	"power-twin-backend/internal/repository"
	"power-twin-backend/internal/service"
	"strings"
	"time"
)

type APIHandler struct {
	topologySvc  *service.TopologyService
	telemetrySvc *service.TelemetryService
	powerFlowSvc *PowerFlowService
	alarmSvc     *AlarmService
	wsHub        *Hub
}

type PowerFlowService struct {
	sqliteRepo  *repository.SQLiteRepo
	influxRepo  *repository.InfluxDBRepo
	alarmEngine *alarm.AlarmEngine
	wsHub       *Hub
}

type AlarmService struct {
	sqliteRepo *repository.SQLiteRepo
}

func NewAPIHandler(topologySvc *service.TopologyService, telemetrySvc *service.TelemetryService, powerFlowSvc *PowerFlowService, alarmSvc *AlarmService, wsHub *Hub) *APIHandler {
	return &APIHandler{
		topologySvc:  topologySvc,
		telemetrySvc: telemetrySvc,
		powerFlowSvc: powerFlowSvc,
		alarmSvc:     alarmSvc,
		wsHub:        wsHub,
	}
}

func NewPowerFlowService(sqliteRepo *repository.SQLiteRepo, influxRepo *repository.InfluxDBRepo, alarmEngine *alarm.AlarmEngine, wsHub *Hub) *PowerFlowService {
	return &PowerFlowService{
		sqliteRepo:  sqliteRepo,
		influxRepo:  influxRepo,
		alarmEngine: alarmEngine,
		wsHub:       wsHub,
	}
}

func NewAlarmService(sqliteRepo *repository.SQLiteRepo) *AlarmService {
	return &AlarmService{sqliteRepo: sqliteRepo}
}

func (h *APIHandler) SetupRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/topology", h.handleTopology)
	mux.HandleFunc("/api/devices/", h.handleDeviceRoutes)
	mux.HandleFunc("/api/simulation/powerflow", h.handlePowerFlow)
	mux.HandleFunc("/api/simulation/n1", h.handleN1)
	mux.HandleFunc("/api/alarms", h.handleAlarms)
	mux.HandleFunc("/api/alarms/", h.handleAlarmAck)
	mux.HandleFunc("/api/metrics/kpi", h.handleKPI)
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ServeWS(h.wsHub, w, r)
	})
}

func (h *APIHandler) handleTopology(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	nodes, edges, err := h.topologySvc.GetTopology()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	response := map[string]interface{}{
		"nodes": nodes,
		"edges": edges,
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (h *APIHandler) handleDeviceRoutes(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/api/devices/")

	parts := strings.SplitN(path, "/", 3)
	if len(parts) < 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}

	deviceID := parts[0]
	action := parts[1]

	switch action {
	case "telemetry":
		rangeStr := r.URL.Query().Get("range")
		if rangeStr == "" {
			rangeStr = "2h"
		}
		data, err := h.topologySvc.GetDeviceTelemetry(deviceID, rangeStr)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	case "history":
		data, err := h.topologySvc.GetDeviceHistory(deviceID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	default:
		http.Error(w, "not found", http.StatusNotFound)
	}
}

func (h *APIHandler) handlePowerFlow(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subs, err := h.powerFlowSvc.sqliteRepo.GetSubstations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	feeders, err := h.powerFlowSvc.sqliteRepo.GetFeeders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	calc := powerflow.NewCalculator(subs, feeders)
	telemetryMap := make(map[string]model.DeviceTelemetry)
	calc.SetTelemetry(telemetryMap)

	result, err := calc.Solve(50, 1e-6)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	result.Timestamp = time.Now()

	if h.powerFlowSvc.wsHub != nil {
		h.powerFlowSvc.wsHub.BroadcastMessage("powerflow_result", result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *APIHandler) handleN1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	subs, err := h.powerFlowSvc.sqliteRepo.GetSubstations()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	feeders, err := h.powerFlowSvc.sqliteRepo.GetFeeders()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	calc := powerflow.NewCalculator(subs, feeders)
	analyzer := powerflow.NewN1Analyzer()
	telemetryMap := make(map[string]model.DeviceTelemetry)

	n1Results, err := analyzer.Analyze(calc, feeders, telemetryMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.powerFlowSvc.alarmEngine != nil {
		h.powerFlowSvc.alarmEngine.CheckN1Results(n1Results)
	}

	if h.powerFlowSvc.wsHub != nil {
		h.powerFlowSvc.wsHub.BroadcastMessage("n1_result", n1Results)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(n1Results)
}

func (h *APIHandler) handleAlarms(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ackStr := r.URL.Query().Get("acknowledged")
	acknowledged := ackStr == "true" || ackStr == "1"
	alarms, err := h.alarmSvc.sqliteRepo.GetAlarms(acknowledged)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(alarms)
}

func (h *APIHandler) handleAlarmAck(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := r.URL.Path
	path = strings.TrimPrefix(path, "/api/alarms/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		http.Error(w, "invalid path", http.StatusBadRequest)
		return
	}
	alarmID := parts[0]
	err := h.alarmSvc.sqliteRepo.AcknowledgeAlarm(alarmID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func (h *APIHandler) handleKPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	metrics, err := h.topologySvc.GetKPIMetrics()
	if err != nil {
		log.Printf("KPI metrics error: %v", err)
		metrics = &model.KPIMetrics{
			TotalPowerMW:         0,
			LineLossMW:           0,
			VoltageQualifiedRate: 0,
			Timestamp:            time.Now(),
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(metrics)
}

func (s *PowerFlowService) RunPeriodicCalculation() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		subs, err := s.sqliteRepo.GetSubstations()
		if err != nil {
			log.Printf("periodic power flow: get substations error: %v", err)
			continue
		}
		feeders, err := s.sqliteRepo.GetFeeders()
		if err != nil {
			log.Printf("periodic power flow: get feeders error: %v", err)
			continue
		}

		calc := powerflow.NewCalculator(subs, feeders)
		telemetryMap := make(map[string]model.DeviceTelemetry)
		calc.SetTelemetry(telemetryMap)

		result, err := calc.Solve(50, 1e-6)
		if err != nil {
			log.Printf("periodic power flow error: %v", err)
			continue
		}
		result.Timestamp = time.Now()

		if s.wsHub != nil {
			s.wsHub.BroadcastMessage("powerflow_result", result)
		}

		metrics := &model.KPIMetrics{
			TotalPowerMW:         result.Losses * 1500 / 1e6,
			LineLossMW:           result.Losses / 1e6,
			VoltageQualifiedRate: 98.5,
			Timestamp:            time.Now(),
		}
		if s.wsHub != nil {
			s.wsHub.BroadcastMessage("kpi_update", metrics)
		}

		fmt.Printf("Periodic power flow: converged=%v, iterations=%d, losses=%.4f\n",
			result.Converged, result.Iterations, result.Losses)
	}
}


