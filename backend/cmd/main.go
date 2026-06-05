package main

import (
	"log"
	"net/http"
	"power-twin-backend/internal/alarm_broker"
	"power-twin-backend/internal/config"
	"power-twin-backend/internal/handler"
	"power-twin-backend/internal/iec61850_gateway"
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/mqtt"
	"power-twin-backend/internal/powerflow_engine"
	"power-twin-backend/internal/reliability_analyzer"
	"power-twin-backend/internal/repository"
	"power-twin-backend/internal/service"
)

func main() {
	log.Println("Starting Urban Rail Transit Power Supply Digital Twin Platform...")

	cfg, err := config.LoadN1Config("./config.json")
	if err != nil {
		log.Printf("Config load error, using defaults: %v", err)
	}

	influxRepo := repository.NewInfluxDBRepo(
		"http://localhost:8086",
		"my-token",
		"power-twin",
		"telemetry",
	)

	sqliteRepo, err := repository.NewSQLiteRepo("./power_twin.db")
	if err != nil {
		log.Fatalf("Failed to initialize SQLite: %v", err)
	}
	log.Println("SQLite database initialized")

	mqttPublisher := mqtt.NewMQTTPublisher("tcp://localhost:1883", "power-twin-backend")

	wsHub := handler.NewHub()
	go wsHub.Run()
	log.Println("WebSocket hub started")

	telemetryCh := make(chan model.DeviceTelemetry, 2000)
	telemetryToPF := make(chan model.DeviceTelemetry, 2000)
	telemetryToBroker := make(chan model.DeviceTelemetry, 2000)

	flowResultFromEngine := make(chan powerflow_engine.PowerFlowResultMsg, 64)
	flowResultToRA := make(chan powerflow_engine.PowerFlowResultMsg, 64)

	n1ResultFromRA := make(chan reliability_analyzer.N1AnalysisMsg, 64)
	n1ResultToBroker := make(chan reliability_analyzer.N1AnalysisMsg, 64)

	go func() {
		for t := range telemetryCh {
			select {
			case telemetryToPF <- t:
			default:
			}
			select {
			case telemetryToBroker <- t:
			default:
			}
		}
	}()

	go func() {
		for msg := range flowResultFromEngine {
			select {
			case flowResultToRA <- msg:
			default:
			}
			wsHub.BroadcastMessage("powerflow_result", msg.Result)
		}
	}()

	go func() {
		for msg := range n1ResultFromRA {
			select {
			case n1ResultToBroker <- msg:
			default:
			}
			wsHub.BroadcastMessage("n1_result", msg.N1Results)
		}
	}()

	gateway := iec61850_gateway.NewGateway(61850, sqliteRepo)
	gateway.TelemetryOut = telemetryCh

	pfEngine := powerflow_engine.NewEngine(telemetryToPF, sqliteRepo, influxRepo, cfg)
	pfEngine.ResultOut = flowResultFromEngine

	ra := reliability_analyzer.NewAnalyzer(flowResultToRA, sqliteRepo, cfg)
	ra.AnalysisOut = n1ResultFromRA

	broker := alarm_broker.NewBroker(telemetryToBroker, n1ResultToBroker, sqliteRepo, mqttPublisher, wsHub, cfg)

	go gateway.Start()
	go pfEngine.Start()
	go ra.Start()
	go broker.Start()

	log.Println("IEC 61850 gateway started on port 61850")

	topologySvc := service.NewTopologyService(sqliteRepo, influxRepo)
	apiHandler := handler.NewAPIHandler(topologySvc, pfEngine, ra, broker, wsHub, sqliteRepo)

	mux := http.NewServeMux()
	apiHandler.SetupRoutes(mux)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
