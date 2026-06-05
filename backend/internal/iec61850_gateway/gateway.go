package iec61850_gateway

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net"
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/repository"
	"sync"
	"time"
)

type Gateway struct {
	Port           int
	Devices        []DeviceInfo
	TelemetryOut   chan model.DeviceTelemetry
	sqliteRepo     *repository.SQLiteRepo
	latestTelemetry sync.Map
}

type DeviceInfo struct {
	DeviceID   string
	DeviceType string
	LineID     string
}

func NewGateway(port int, sqliteRepo *repository.SQLiteRepo) *Gateway {
	g := &Gateway{
		Port:         port,
		TelemetryOut: make(chan model.DeviceTelemetry, 2000),
		sqliteRepo:   sqliteRepo,
	}
	g.loadDevices()
	return g
}

func (g *Gateway) loadDevices() {
	subs, err := g.sqliteRepo.GetSubstations()
	if err != nil {
		log.Printf("load substations error: %v", err)
		return
	}
	for _, sub := range subs {
		g.Devices = append(g.Devices, DeviceInfo{
			DeviceID:   sub.ID,
			DeviceType: "substation",
			LineID:     sub.LineID,
		})
	}

	rows, err := g.sqliteRepo.DB.Query("SELECT id, substation_id FROM rectifiers")
	if err != nil {
		log.Printf("load rectifiers error: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, subID string
		rows.Scan(&id, &subID)
		g.Devices = append(g.Devices, DeviceInfo{
			DeviceID:   id,
			DeviceType: "rectifier",
			LineID:     "",
		})
	}

	rows2, err := g.sqliteRepo.DB.Query("SELECT id, substation_id FROM dc_switchgears")
	if err != nil {
		log.Printf("load switchgears error: %v", err)
		return
	}
	defer rows2.Close()
	for rows2.Next() {
		var id, subID string
		rows2.Scan(&id, &subID)
		g.Devices = append(g.Devices, DeviceInfo{
			DeviceID:   id,
			DeviceType: "dc_switchgear",
			LineID:     "",
		})
	}

	log.Printf("Gateway loaded %d devices", len(g.Devices))
}

func (g *Gateway) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", g.Port))
	if err != nil {
		log.Printf("gateway listen error: %v", err)
		return
	}
	log.Printf("IEC 61850 gateway listening on :%d", g.Port)

	go g.generateLoop()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("gateway accept error: %v", err)
			continue
		}
		go g.handleConnection(conn)
	}
}

func (g *Gateway) generateLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		for _, dev := range g.Devices {
			t := g.GenerateTelemetry(dev.DeviceID, dev.DeviceType)
			g.latestTelemetry.Store(dev.DeviceID, t)
			select {
			case g.TelemetryOut <- t:
			default:
			}
		}
	}
}

func (g *Gateway) handleConnection(conn net.Conn) {
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, dev := range g.Devices {
			t := g.GenerateTelemetry(dev.DeviceID, dev.DeviceType)
			data, err := json.Marshal(t)
			if err != nil {
				continue
			}
			writer.Write(data)
			writer.WriteByte('\n')
		}
		writer.Flush()
	}
}

func (g *Gateway) GenerateTelemetry(deviceID, deviceType string) model.DeviceTelemetry {
	var voltage, maxCurrent, temperature, loadRate float64

	faultChance := rand.Float64()
	if faultChance < 0.002 {
		voltage = 1200 + rand.Float64()*100
	} else {
		voltage = 1400 + rand.Float64()*200
	}

	switch deviceType {
	case "substation":
		maxCurrent = 2000
	case "rectifier":
		maxCurrent = 1500
	case "dc_switchgear":
		maxCurrent = 1000
	default:
		maxCurrent = 1000
	}

	spikeChance := rand.Float64()
	if spikeChance < 0.005 {
		loadRate = 105 + rand.Float64()*15
	} else {
		loadRate = 40 + rand.Float64()*30
	}

	current := maxCurrent * loadRate / 100.0
	power := voltage * current
	temperature = 30 + rand.Float64()*50

	return model.DeviceTelemetry{
		DeviceID:    deviceID,
		DeviceType:  deviceType,
		Voltage:     math.Round(voltage*100) / 100,
		Current:     math.Round(current*100) / 100,
		Power:       math.Round(power*100) / 100,
		Temperature: math.Round(temperature*100) / 100,
		LoadRate:    math.Round(loadRate*100) / 100,
		Timestamp:   time.Now(),
	}
}

func (g *Gateway) GetLatestTelemetry() map[string]model.DeviceTelemetry {
	result := make(map[string]model.DeviceTelemetry)
	g.latestTelemetry.Range(func(key, value interface{}) bool {
		if k, ok := key.(string); ok {
			if v, ok := value.(model.DeviceTelemetry); ok {
				result[k] = v
			}
		}
		return true
	})
	return result
}
