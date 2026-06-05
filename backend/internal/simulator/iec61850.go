package simulator

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
	"time"
)

type Simulator struct {
	Port          int
	Devices       []DeviceInfo
	TelemetryChan chan model.DeviceTelemetry
	sqliteRepo    *repository.SQLiteRepo
}

type DeviceInfo struct {
	DeviceID   string
	DeviceType string
	LineID     string
}

func NewSimulator(port int, sqliteRepo *repository.SQLiteRepo) *Simulator {
	sim := &Simulator{
		Port:          port,
		TelemetryChan: make(chan model.DeviceTelemetry, 2000),
		sqliteRepo:    sqliteRepo,
	}
	sim.loadDevices()
	return sim
}

func (s *Simulator) loadDevices() {
	subs, err := s.sqliteRepo.GetSubstations()
	if err != nil {
		log.Printf("load substations error: %v", err)
		return
	}
	for _, sub := range subs {
		s.Devices = append(s.Devices, DeviceInfo{
			DeviceID:   sub.ID,
			DeviceType: "substation",
			LineID:     sub.LineID,
		})
	}

	rows, err := s.sqliteRepo.DB.Query("SELECT id, substation_id FROM rectifiers")
	if err != nil {
		log.Printf("load rectifiers error: %v", err)
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, subID string
		rows.Scan(&id, &subID)
		s.Devices = append(s.Devices, DeviceInfo{
			DeviceID:   id,
			DeviceType: "rectifier",
			LineID:     "",
		})
	}

	rows2, err := s.sqliteRepo.DB.Query("SELECT id, substation_id FROM dc_switchgears")
	if err != nil {
		log.Printf("load switchgears error: %v", err)
		return
	}
	defer rows2.Close()
	for rows2.Next() {
		var id, subID string
		rows2.Scan(&id, &subID)
		s.Devices = append(s.Devices, DeviceInfo{
			DeviceID:   id,
			DeviceType: "dc_switchgear",
			LineID:     "",
		})
	}

	log.Printf("Simulator loaded %d devices", len(s.Devices))
}

func (s *Simulator) Start() {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", s.Port))
	if err != nil {
		log.Printf("simulator listen error: %v", err)
		return
	}
	log.Printf("IEC 61850 simulator listening on :%d", s.Port)

	go s.generateLoop()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("simulator accept error: %v", err)
			continue
		}
		go s.handleConnection(conn)
	}
}

func (s *Simulator) generateLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		for _, dev := range s.Devices {
			t := s.GenerateTelemetry(dev.DeviceID, dev.DeviceType)
			select {
			case s.TelemetryChan <- t:
			default:
			}
		}
	}
}

func (s *Simulator) handleConnection(conn net.Conn) {
	defer conn.Close()
	writer := bufio.NewWriter(conn)
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		for _, dev := range s.Devices {
			t := s.GenerateTelemetry(dev.DeviceID, dev.DeviceType)
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

func (s *Simulator) GenerateTelemetry(deviceID, deviceType string) model.DeviceTelemetry {
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
