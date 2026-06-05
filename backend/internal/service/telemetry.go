package service

import (
	"power-twin-backend/internal/alarm"
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/repository"
	"strings"
	"sync"
	"time"
)

type TelemetryService struct {
	influxRepo     *repository.InfluxDBRepo
	sqliteRepo     *repository.SQLiteRepo
	alarmEngine    *alarm.AlarmEngine
	substationMap  map[string]string
	batchMutex     sync.Mutex
	pendingBatch   map[string][]model.DeviceTelemetry
	flushInterval  time.Duration
}

func NewTelemetryService(influxRepo *repository.InfluxDBRepo, sqliteRepo *repository.SQLiteRepo, alarmEngine *alarm.AlarmEngine) *TelemetryService {
	svc := &TelemetryService{
		influxRepo:    influxRepo,
		sqliteRepo:    sqliteRepo,
		alarmEngine:   alarmEngine,
		substationMap: make(map[string]string),
		pendingBatch:  make(map[string][]model.DeviceTelemetry),
		flushInterval: 1 * time.Second,
	}
	svc.buildSubstationMap()
	go svc.flushLoop()
	return svc
}

func (s *TelemetryService) buildSubstationMap() {
	type subDevice struct {
		id  string
		sub string
	}

	rows, err := s.sqliteRepo.DB.Query("SELECT id, substation_id FROM rectifiers")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id, subID string
			if rows.Scan(&id, &subID) == nil {
				s.substationMap[id] = subID
			}
		}
	}

	rows2, err := s.sqliteRepo.DB.Query("SELECT id, substation_id FROM dc_switchgears")
	if err == nil {
		defer rows2.Close()
		for rows2.Next() {
			var id, subID string
			if rows2.Scan(&id, &subID) == nil {
				s.substationMap[id] = subID
			}
		}
	}

	subs, err := s.sqliteRepo.GetSubstations()
	if err == nil {
		for _, sub := range subs {
			s.substationMap[sub.ID] = sub.ID
		}
	}
}

func (s *TelemetryService) getSubstationID(deviceID string) string {
	if subID, ok := s.substationMap[deviceID]; ok {
		return subID
	}
	if strings.HasPrefix(deviceID, "sub_") {
		return deviceID
	}
	return "unknown"
}

func (s *TelemetryService) ProcessTelemetry(telemetry model.DeviceTelemetry) error {
	s.alarmEngine.CheckTelemetry(telemetry)

	subID := s.getSubstationID(telemetry.DeviceID)

	s.batchMutex.Lock()
	s.pendingBatch[subID] = append(s.pendingBatch[subID], telemetry)
	s.batchMutex.Unlock()

	return nil
}

func (s *TelemetryService) flushLoop() {
	ticker := time.NewTicker(s.flushInterval)
	defer ticker.Stop()
	for range ticker.C {
		s.flushBatches()
	}
}

func (s *TelemetryService) flushBatches() {
	s.batchMutex.Lock()
	batches := s.pendingBatch
	s.pendingBatch = make(map[string][]model.DeviceTelemetry)
	s.batchMutex.Unlock()

	for subID, batch := range batches {
		if len(batch) == 0 {
			continue
		}
		err := s.influxRepo.WriteTelemetryBatch(batch)
		if err != nil {
			s.batchMutex.Lock()
			existing := s.pendingBatch[subID]
			if len(existing)+len(batch) > 200 {
				batch = batch[:200-len(existing)]
			}
			s.pendingBatch[subID] = append(existing, batch...)
			s.batchMutex.Unlock()
		}
	}
}

func (s *TelemetryService) GetTelemetryHistory(deviceID, rangeStr string) ([]model.DeviceTelemetry, error) {
	return s.influxRepo.QueryTelemetry(deviceID, rangeStr)
}
