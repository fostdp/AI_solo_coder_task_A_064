package handler

import (
	"encoding/json"
	"log"
	"net/http"
	"power-twin-backend/internal/alarm_broker"
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/powerflow_engine"
	"power-twin-backend/internal/reliability_analyzer"
	"power-twin-backend/internal/repository"
	"power-twin-backend/internal/service"
	"strings"
	"time"
)

type APIHandler struct {
	topologySvc         *service.TopologyService
	pfEngine            *powerflow_engine.Engine
	reliabilityAnalyzer *reliability_analyzer.Analyzer
	alarmBroker         *alarm_broker.Broker
	wsHub               *Hub
	sqliteRepo          *repository.SQLiteRepo
}

func NewAPIHandler(topologySvc *service.TopologyService, pfEngine *powerflow_engine.Engine, ra *reliability_analyzer.Analyzer, broker *alarm_broker.Broker, wsHub *Hub, sqliteRepo *repository.SQLiteRepo) *APIHandler {
	return &APIHandler{
		topologySvc:         topologySvc,
		pfEngine:            pfEngine,
		reliabilityAnalyzer: ra,
		alarmBroker:         broker,
		wsHub:               wsHub,
		sqliteRepo:          sqliteRepo,
	}
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

	result, err := h.pfEngine.RunCalculation()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.wsHub != nil {
		h.wsHub.BroadcastMessage("powerflow_result", result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (h *APIHandler) handleN1(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	n1Results, err := h.reliabilityAnalyzer.RunN1Analysis()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if h.wsHub != nil {
		h.wsHub.BroadcastMessage("n1_result", n1Results)
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
	alarms, err := h.sqliteRepo.GetAlarms(acknowledged)
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
	err := h.sqliteRepo.AcknowledgeAlarm(alarmID)
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
