package repository

import (
	"context"
	"fmt"
	"power-twin-backend/internal/model"
	"time"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
)

type InfluxDBRepo struct {
	client   influxdb2.Client
	writeAPI api.WriteAPIBlocking
	queryAPI api.QueryAPI
	org      string
	bucket   string
}

func NewInfluxDBRepo(url, token, org, bucket string) *InfluxDBRepo {
	client := influxdb2.NewClient(url, token)
	return &InfluxDBRepo{
		client:   client,
		writeAPI: client.WriteAPIBlocking(org, bucket),
		queryAPI: client.QueryAPI(org),
		org:      org,
		bucket:   bucket,
	}
}

func (r *InfluxDBRepo) WriteTelemetry(telemetry model.DeviceTelemetry) error {
	p := influxdb2.NewPoint(
		"device_telemetry",
		map[string]string{
			"device_id":   telemetry.DeviceID,
			"device_type": telemetry.DeviceType,
		},
		map[string]interface{}{
			"voltage":     telemetry.Voltage,
			"current":     telemetry.Current,
			"power":       telemetry.Power,
			"temperature": telemetry.Temperature,
			"load_rate":   telemetry.LoadRate,
		},
		telemetry.Timestamp,
	)
	return r.writeAPI.WritePoint(context.Background(), p)
}

func (r *InfluxDBRepo) QueryTelemetry(deviceID, rangeStr string) ([]model.DeviceTelemetry, error) {
	query := fmt.Sprintf(`from(bucket: "%s")
  |> range(start: -%s)
  |> filter(fn: (r) => r._measurement == "device_telemetry")
  |> filter(fn: (r) => r.device_id == "%s")
  |> pivot(rowKey: ["_time"], columnKey: ["_field"], valueColumn: "_value")`, r.bucket, rangeStr, deviceID)

	result, err := r.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	var telemetryList []model.DeviceTelemetry
	for result.Next() {
		record := result.Record()
		t := model.DeviceTelemetry{
			DeviceID:    deviceID,
			DeviceType:  "",
			Timestamp:   record.Time(),
		}
		if v, ok := record.ValueByKey("voltage").(float64); ok {
			t.Voltage = v
		}
		if v, ok := record.ValueByKey("current").(float64); ok {
			t.Current = v
		}
		if v, ok := record.ValueByKey("power").(float64); ok {
			t.Power = v
		}
		if v, ok := record.ValueByKey("temperature").(float64); ok {
			t.Temperature = v
		}
		if v, ok := record.ValueByKey("load_rate").(float64); ok {
			t.LoadRate = v
		}
		if v, ok := record.ValueByKey("device_type").(string); ok {
			t.DeviceType = v
		}
		telemetryList = append(telemetryList, t)
	}
	return telemetryList, nil
}

func (r *InfluxDBRepo) QueryLatestMetrics() (*model.KPIMetrics, error) {
	query := fmt.Sprintf(`from(bucket: "%s")
  |> range(start: -5m)
  |> filter(fn: (r) => r._measurement == "device_telemetry")
  |> filter(fn: (r) => r._field == "power" or r._field == "load_rate" or r._field == "voltage")
  |> last()`, r.bucket)

	result, err := r.queryAPI.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	var totalPower float64
	var voltageCount float64
	var voltageQualified float64

	for result.Next() {
		record := result.Record()
		field, ok := record.ValueByKey("_field").(string)
		if !ok {
			continue
		}
		value, ok := record.Value().(float64)
		if !ok {
			continue
		}
		switch field {
		case "power":
			totalPower += value
		case "voltage":
			voltageCount++
			if value >= 1350 && value <= 1650 {
				voltageQualified++
			}
		}
	}

	var voltageQualifiedRate float64
	if voltageCount > 0 {
		voltageQualifiedRate = voltageQualified / voltageCount * 100
	}

	return &model.KPIMetrics{
		TotalPowerMW:         totalPower / 1e6,
		LineLossMW:           totalPower * 0.03 / 1e6,
		VoltageQualifiedRate: voltageQualifiedRate,
		Timestamp:            time.Now(),
	}, nil
}
