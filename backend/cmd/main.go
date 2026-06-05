package main

import (
	"log"
	"net/http"
	"power-twin-backend/internal/alarm"
	"power-twin-backend/internal/handler"
	"power-twin-backend/internal/mqtt"
	"power-twin-backend/internal/repository"
	"power-twin-backend/internal/service"
	"power-twin-backend/internal/simulator"
)

func main() {
	log.Println("Starting Urban Rail Transit Power Supply Digital Twin Platform...")

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

	alarmEngine := alarm.NewAlarmEngine(sqliteRepo, mqttPublisher, wsHub)
	log.Println("Alarm engine initialized")

	topologySvc := service.NewTopologyService(sqliteRepo, influxRepo)
	telemetrySvc := service.NewTelemetryService(influxRepo, sqliteRepo, alarmEngine)
	powerFlowSvc := handler.NewPowerFlowService(sqliteRepo, influxRepo, alarmEngine, wsHub)
	alarmSvc := handler.NewAlarmService(sqliteRepo)

	apiHandler := handler.NewAPIHandler(topologySvc, telemetrySvc, powerFlowSvc, alarmSvc, wsHub)

	sim := simulator.NewSimulator(61850, sqliteRepo)
	go sim.Start()
	log.Println("IEC 61850 simulator started on port 61850")

	go func() {
		for telemetry := range sim.TelemetryChan {
			err := telemetrySvc.ProcessTelemetry(telemetry)
			if err != nil {
				log.Printf("Process telemetry error: %v", err)
			}
			wsHub.BroadcastMessage("telemetry_update", telemetry)
		}
	}()

	go powerFlowSvc.RunPeriodicCalculation()
	log.Println("Periodic power flow calculation started (every 30s)")

	mux := http.NewServeMux()
	apiHandler.SetupRoutes(mux)

	log.Println("Server starting on :8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
