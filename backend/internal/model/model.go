package model

import (
	"encoding/json"
	"time"
)

type Line struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	SubstationCount int   `json:"substation_count"`
}

type Substation struct {
	ID     string  `json:"id"`
	LineID string  `json:"line_id"`
	Name   string  `json:"name"`
	PosX   float64 `json:"pos_x"`
	PosY   float64 `json:"pos_y"`
	PosZ   float64 `json:"pos_z"`
}

type Rectifier struct {
	ID           string  `json:"id"`
	SubstationID string  `json:"substation_id"`
	Name         string  `json:"name"`
	RatedPower   float64 `json:"rated_power"`
}

type DCSwitchgear struct {
	ID            string  `json:"id"`
	SubstationID  string  `json:"substation_id"`
	Name          string  `json:"name"`
	RatedCurrent  float64 `json:"rated_current"`
}

type Feeder struct {
	ID           string  `json:"id"`
	SourceID     string  `json:"source_id"`
	TargetID     string  `json:"target_id"`
	ImpedanceR   float64 `json:"impedance_r"`
	ImpedanceX   float64 `json:"impedance_x"`
	RatedCurrent float64 `json:"rated_current"`
}

type DeviceTelemetry struct {
	DeviceID    string    `json:"device_id"`
	DeviceType  string    `json:"device_type"`
	Voltage     float64   `json:"voltage"`
	Current     float64   `json:"current"`
	Power       float64   `json:"power"`
	Temperature float64   `json:"temperature"`
	LoadRate    float64   `json:"load_rate"`
	Timestamp   time.Time `json:"timestamp"`
}

type TopologyNode struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	LineID   string  `json:"line_id"`
	Position [3]float64 `json:"position"`
	LoadRate float64 `json:"load_rate"`
	Status   string  `json:"status"`
}

type TopologyEdge struct {
	ID       string  `json:"id"`
	Source   string  `json:"source"`
	Target   string  `json:"target"`
	Type     string  `json:"type"`
	LoadRate float64 `json:"load_rate"`
	Status   string  `json:"status"`
}

type PowerFlowResult struct {
	Timestamp     time.Time              `json:"timestamp"`
	Converged     bool                   `json:"converged"`
	Iterations    int                    `json:"iterations"`
	NodeVoltages  map[string]float64     `json:"node_voltages"`
	BranchPowers  map[string]float64     `json:"branch_powers"`
	Losses        float64                `json:"losses"`
	N1Results     []N1Result             `json:"n1_results"`
}

type N1Result struct {
	FaultBranch         string   `json:"fault_branch"`
	Overloads           []string `json:"overloads"`
	Safe                bool     `json:"safe"`
	TransferSuggestion  string   `json:"transfer_suggestion"`
}

type Alarm struct {
	ID           string    `json:"id"`
	Level        int       `json:"level"`
	Type         string    `json:"type"`
	DeviceID     string    `json:"device_id"`
	Message      string    `json:"message"`
	Timestamp    time.Time `json:"timestamp"`
	Acknowledged bool      `json:"acknowledged"`
}

type KPIMetrics struct {
	TotalPowerMW        float64   `json:"total_power_mw"`
	LineLossMW          float64   `json:"line_loss_mw"`
	VoltageQualifiedRate float64  `json:"voltage_qualified_rate"`
	Timestamp           time.Time `json:"timestamp"`
}

type OverloadTracker struct {
	DeviceID     string    `json:"device_id"`
	OverloadStart time.Time `json:"overload_start"`
	Notified     bool      `json:"notified"`
}

type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}
