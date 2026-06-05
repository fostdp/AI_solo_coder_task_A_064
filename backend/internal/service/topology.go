package service

import (
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/repository"
)

type TopologyService struct {
	sqliteRepo *repository.SQLiteRepo
	influxRepo *repository.InfluxDBRepo
}

func NewTopologyService(sqliteRepo *repository.SQLiteRepo, influxRepo *repository.InfluxDBRepo) *TopologyService {
	return &TopologyService{
		sqliteRepo: sqliteRepo,
		influxRepo: influxRepo,
	}
}

func (s *TopologyService) GetTopology() ([]model.TopologyNode, []model.TopologyEdge, error) {
	nodes, edges, err := s.sqliteRepo.GetTopology()
	if err != nil {
		return nil, nil, err
	}
	return nodes, edges, nil
}

func (s *TopologyService) GetDeviceTelemetry(deviceID, rangeStr string) ([]model.DeviceTelemetry, error) {
	return s.influxRepo.QueryTelemetry(deviceID, rangeStr)
}

func (s *TopologyService) GetDeviceHistory(deviceID string) ([]map[string]interface{}, error) {
	return s.sqliteRepo.GetOperationHistory(deviceID)
}

func (s *TopologyService) GetKPIMetrics() (*model.KPIMetrics, error) {
	return s.influxRepo.QueryLatestMetrics()
}
