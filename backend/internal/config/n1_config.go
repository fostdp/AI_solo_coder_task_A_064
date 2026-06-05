package config

import (
	"encoding/json"
	"os"
)

type N1Config struct {
	MaxIterations        int     `json:"max_iterations"`
	Tolerance            float64 `json:"tolerance"`
	OverloadThreshold    float64 `json:"overload_threshold"`
	OverloadDurationSec  int     `json:"overload_duration_sec"`
	ScanIntervalSec      int     `json:"scan_interval_sec"`
	PowerFlowIntervalSec int     `json:"powerflow_interval_sec"`
	BaseVoltage          float64 `json:"base_voltage"`
	VoltageMinPU         float64 `json:"voltage_min_pu"`
	VoltageMaxPU         float64 `json:"voltage_max_pu"`
	FeederCapacityRatio  float64 `json:"feeder_capacity_ratio"`
}

func DefaultN1Config() *N1Config {
	return &N1Config{
		MaxIterations:        50,
		Tolerance:            1e-6,
		OverloadThreshold:    100.0,
		OverloadDurationSec:  10,
		ScanIntervalSec:      30,
		PowerFlowIntervalSec: 30,
		BaseVoltage:          1500.0,
		VoltageMinPU:         0.9,
		VoltageMaxPU:         1.1,
		FeederCapacityRatio:  0.8,
	}
}

func LoadN1Config(path string) (*N1Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return DefaultN1Config(), nil
	}
	cfg := DefaultN1Config()
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return DefaultN1Config(), nil
	}
	return cfg, nil
}
