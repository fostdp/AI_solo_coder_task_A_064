package service

import (
	"power-twin-backend/internal/alarm"
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/repository"
)

type TelemetryService struct {
	influxRepo   *repository.InfluxDBRepo
	sqliteRepo   *repository.SQLiteRepo
	alarmEngine  *alarm.AlarmEngine
}

func NewTelemetryService(influxRepo *repository.InfluxDBRepo, sqliteRepo *repository.SQLiteRepo, alarmEngine *alarm.AlarmEngine) *TelemetryService {
	return &TelemetryService{
		influxRepo:  influxRepo,
		sqliteRepo:  sqliteRepo,
		alarmEngine: alarmEngine,
	}
}

func (s *TelemetryService) ProcessTelemetry(telemetry model.DeviceTelemetry) error {
	err := s.influxRepo.WriteTelemetry(telemetry)
	if err != nil {
		return err
	}
	s.alarmEngine.CheckTelemetry(telemetry)
	return nil
}

func (s *TelemetryService) GetTelemetryHistory(deviceID, rangeStr string) ([]model.DeviceTelemetry, error) {
	return s.influxRepo.QueryTelemetry(deviceID, rangeStr)
}
