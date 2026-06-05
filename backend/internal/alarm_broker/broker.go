package alarm_broker

import (
	"fmt"
	"power-twin-backend/internal/config"
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/mqtt"
	"power-twin-backend/internal/reliability_analyzer"
	"power-twin-backend/internal/repository"
	"sync"
	"time"
)

type Broadcaster interface {
	BroadcastMessage(msgType string, payload interface{})
}

type Broker struct {
	TelemetryIn      chan model.DeviceTelemetry
	N1ResultIn       chan reliability_analyzer.N1AnalysisMsg
	sqliteRepo       *repository.SQLiteRepo
	mqttPublisher    *mqtt.MQTTPublisher
	wsHub            Broadcaster
	config           *config.N1Config
	overloadTrackers map[string]*model.OverloadTracker
	n1Tracker        map[string]bool
	mu               sync.Mutex
	alarmCounter     int
}

func NewBroker(telemetryIn chan model.DeviceTelemetry, n1ResultIn chan reliability_analyzer.N1AnalysisMsg, sqliteRepo *repository.SQLiteRepo, mqttPub *mqtt.MQTTPublisher, wsHub Broadcaster, cfg *config.N1Config) *Broker {
	return &Broker{
		TelemetryIn:      telemetryIn,
		N1ResultIn:       n1ResultIn,
		sqliteRepo:       sqliteRepo,
		mqttPublisher:    mqttPub,
		wsHub:            wsHub,
		config:           cfg,
		overloadTrackers: make(map[string]*model.OverloadTracker),
		n1Tracker:        make(map[string]bool),
	}
}

func (b *Broker) Start() {
	go b.telemetryAlarmChecker()
	go b.n1AlarmChecker()
}

func (b *Broker) telemetryAlarmChecker() {
	for t := range b.TelemetryIn {
		b.checkTelemetry(t)
	}
}

func (b *Broker) checkTelemetry(telemetry model.DeviceTelemetry) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if telemetry.LoadRate > b.config.OverloadThreshold {
		tracker, exists := b.overloadTrackers[telemetry.DeviceID]
		if !exists {
			b.overloadTrackers[telemetry.DeviceID] = &model.OverloadTracker{
				DeviceID:      telemetry.DeviceID,
				OverloadStart: time.Now(),
				Notified:      false,
			}
			return
		}
		if !tracker.Notified && time.Since(tracker.OverloadStart) >= time.Duration(b.config.OverloadDurationSec)*time.Second {
			tracker.Notified = true
			alarm := model.Alarm{
				ID:           fmt.Sprintf("alarm_%d", b.alarmCounter+1),
				Level:        1,
				Type:         "overload",
				DeviceID:     telemetry.DeviceID,
				Message:      fmt.Sprintf("设备 %s 过载持续超过%d秒，当前负载率 %.1f%%", telemetry.DeviceID, b.config.OverloadDurationSec, telemetry.LoadRate),
				Timestamp:    time.Now(),
				Acknowledged: false,
			}
			b.alarmCounter++
			b.TriggerAlarm(alarm)
		}
	} else {
		delete(b.overloadTrackers, telemetry.DeviceID)
	}
}

func (b *Broker) n1AlarmChecker() {
	for msg := range b.N1ResultIn {
		b.checkN1Results(msg.N1Results)
	}
}

func (b *Broker) checkN1Results(n1Results []model.N1Result) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for _, result := range n1Results {
		if !result.Safe {
			if !b.n1Tracker[result.FaultBranch] {
				b.n1Tracker[result.FaultBranch] = true
				alarm := model.Alarm{
					ID:           fmt.Sprintf("alarm_%d", b.alarmCounter+1),
					Level:        2,
					Type:         "n1_violation",
					DeviceID:     result.FaultBranch,
					Message:      fmt.Sprintf("N-1校验不通过：支路 %s 故障后存在过载风险，建议 %s", result.FaultBranch, result.TransferSuggestion),
					Timestamp:    time.Now(),
					Acknowledged: false,
				}
				b.alarmCounter++
				b.TriggerAlarm(alarm)
			}
		} else {
			delete(b.n1Tracker, result.FaultBranch)
		}
	}
}

func (b *Broker) TriggerAlarm(alarm model.Alarm) {
	err := b.sqliteRepo.InsertAlarm(alarm)
	if err != nil {
		fmt.Printf("insert alarm error: %v\n", err)
	}

	if b.mqttPublisher != nil {
		err = b.mqttPublisher.PublishAlarm(alarm)
		if err != nil {
			fmt.Printf("mqtt publish alarm error: %v\n", err)
		}
	}

	if b.wsHub != nil {
		b.wsHub.BroadcastMessage("alarm", alarm)
	}
}
