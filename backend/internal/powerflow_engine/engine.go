package powerflow_engine

import (
	"log"
	"power-twin-backend/internal/config"
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/powerflow"
	"power-twin-backend/internal/repository"
	"strings"
	"sync"
	"time"
)

type PowerFlowResultMsg struct {
	Result    *model.PowerFlowResult
	Timestamp time.Time
}

type Engine struct {
	TelemetryIn    chan model.DeviceTelemetry
	ResultOut      chan PowerFlowResultMsg
	sqliteRepo     *repository.SQLiteRepo
	influxRepo     *repository.InfluxDBRepo
	config         *config.N1Config
	calculator     *powerflow.PowerFlowCalculator
	latestTelemetry map[string]model.DeviceTelemetry
	mu             sync.Mutex
	substationMap  map[string]string
	batchMutex     sync.Mutex
	pendingBatch   map[string][]model.DeviceTelemetry
}

func NewEngine(telemetryIn chan model.DeviceTelemetry, sqliteRepo *repository.SQLiteRepo, influxRepo *repository.InfluxDBRepo, cfg *config.N1Config) *Engine {
	e := &Engine{
		TelemetryIn:     telemetryIn,
		ResultOut:       make(chan PowerFlowResultMsg, 64),
		sqliteRepo:      sqliteRepo,
		influxRepo:      influxRepo,
		config:          cfg,
		latestTelemetry: make(map[string]model.DeviceTelemetry),
		substationMap:   make(map[string]string),
		pendingBatch:    make(map[string][]model.DeviceTelemetry),
	}
	e.buildSubstationMap()
	return e
}

func (e *Engine) buildSubstationMap() {
	rows, err := e.sqliteRepo.DB.Query("SELECT id, substation_id FROM rectifiers")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, subID string
			if rows.Scan(&id, &subID) == nil {
				e.substationMap[id] = subID
			}
		}
	}

	rows2, err := e.sqliteRepo.DB.Query("SELECT id, substation_id FROM dc_switchgears")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var id, subID string
			if rows2.Scan(&id, &subID) == nil {
				e.substationMap[id] = subID
			}
		}
	}

	subs, err := e.sqliteRepo.GetSubstations()
	if err == nil {
		for _, sub := range subs {
			e.substationMap[sub.ID] = sub.ID
		}
	}
}

func (e *Engine) getSubstationID(deviceID string) string {
	if subID, ok := e.substationMap[deviceID]; ok {
		return subID
	}
	if strings.HasPrefix(deviceID, "sub_") {
		return deviceID
	}
	return "unknown"
}

func (e *Engine) Start() {
	go e.telemetryConsumer()
	go e.flushLoop()
	go e.periodicCalculation()
}

func (e *Engine) telemetryConsumer() {
	for t := range e.TelemetryIn {
		e.mu.Lock()
		e.latestTelemetry[t.DeviceID] = t
		e.mu.Unlock()

		subID := e.getSubstationID(t.DeviceID)
		e.batchMutex.Lock()
		e.pendingBatch[subID] = append(e.pendingBatch[subID], t)
		e.batchMutex.Unlock()
	}
}

func (e *Engine) flushLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		e.flushBatches()
	}
}

func (e *Engine) flushBatches() {
	e.batchMutex.Lock()
	batches := e.pendingBatch
	e.pendingBatch = make(map[string][]model.DeviceTelemetry)
	e.batchMutex.Unlock()

	for subID, batch := range batches {
		if len(batch) == 0 {
			continue
		}
		err := e.influxRepo.WriteTelemetryBatch(batch)
		if err != nil {
			e.batchMutex.Lock()
			existing := e.pendingBatch[subID]
			if len(existing)+len(batch) > 200 {
				batch = batch[:200-len(existing)]
			}
			e.pendingBatch[subID] = append(existing, batch...)
			e.batchMutex.Unlock()
		}
	}
}

func (e *Engine) periodicCalculation() {
	interval := time.Duration(e.config.PowerFlowIntervalSec) * time.Second
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		result, err := e.RunCalculation()
		if err != nil {
			log.Printf("periodic power flow error: %v", err)
			continue
		}
		msg := PowerFlowResultMsg{
			Result:    result,
			Timestamp: time.Now(),
		}
		select {
		case e.ResultOut <- msg:
		default:
		}
	}
}

func (e *Engine) RunCalculation() (*model.PowerFlowResult, error) {
	subs, err := e.sqliteRepo.GetSubstations()
	if err != nil {
		return nil, err
	}
	feeders, err := e.sqliteRepo.GetFeeders()
	if err != nil {
		return nil, err
	}

	e.mu.Lock()
	telemetryMap := make(map[string]model.DeviceTelemetry)
	for k, v := range e.latestTelemetry {
		telemetryMap[k] = v
	}
	e.mu.Unlock()

	calc := powerflow.NewCalculator(subs, feeders)
	calc.SetTelemetry(telemetryMap)

	result, err := calc.Solve(e.config.MaxIterations, e.config.Tolerance)
	if err != nil {
		return nil, err
	}
	result.Timestamp = time.Now()

	return result, nil
}
