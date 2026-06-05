package alarm

import (
	"fmt"
	"power-twin-backend/internal/model"
	"power-twin-backend/internal/mqtt"
	"power-twin-backend/internal/repository"
	"sync"
	"time"
)

type Broadcaster interface {
	BroadcastMessage(msgType string, payload interface{})
}

type AlarmEngine struct {
	overloadTrackers map[string]*model.OverloadTracker
	sqliteRepo       *repository.SQLiteRepo
	mqttPublisher    *mqtt.MQTTPublisher
	broadcaster      Broadcaster
	mu               sync.Mutex
	alarmCounter     int
}

func NewAlarmEngine(sqliteRepo *repository.SQLiteRepo, mqttPublisher *mqtt.MQTTPublisher, broadcaster Broadcaster) *AlarmEngine {
	return &AlarmEngine{
		overloadTrackers: make(map[string]*model.OverloadTracker),
		sqliteRepo:       sqliteRepo,
		mqttPublisher:    mqttPublisher,
		broadcaster:      broadcaster,
	}
}

func (e *AlarmEngine) CheckTelemetry(telemetry model.DeviceTelemetry) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if telemetry.LoadRate > 100 {
		tracker, exists := e.overloadTrackers[telemetry.DeviceID]
		if !exists {
			e.overloadTrackers[telemetry.DeviceID] = &model.OverloadTracker{
				DeviceID:      telemetry.DeviceID,
				OverloadStart: time.Now(),
				Notified:      false,
			}
			return
		}
		if !tracker.Notified && time.Since(tracker.OverloadStart) >= 10*time.Second {
			tracker.Notified = true
			alarm := model.Alarm{
				ID:           fmt.Sprintf("alarm_%d", e.alarmCounter+1),
				Level:        1,
				Type:         "overload",
				DeviceID:     telemetry.DeviceID,
				Message:      fmt.Sprintf("设备 %s 过载持续超过10秒，当前负载率 %.1f%%", telemetry.DeviceID, telemetry.LoadRate),
				Timestamp:    time.Now(),
				Acknowledged: false,
			}
			e.alarmCounter++
			e.TriggerAlarm(alarm)
		}
	} else {
		delete(e.overloadTrackers, telemetry.DeviceID)
	}
}

func (e *AlarmEngine) CheckN1Results(n1Results []model.N1Result) {
	e.mu.Lock()
	defer e.mu.Unlock()

	for _, result := range n1Results {
		if !result.Safe {
			alarm := model.Alarm{
				ID:           fmt.Sprintf("alarm_%d", e.alarmCounter+1),
				Level:        2,
				Type:         "n1_violation",
				DeviceID:     result.FaultBranch,
				Message:      fmt.Sprintf("N-1校验不通过：支路 %s 故障后存在过载风险，建议 %s", result.FaultBranch, result.TransferSuggestion),
				Timestamp:    time.Now(),
				Acknowledged: false,
			}
			e.alarmCounter++
			e.TriggerAlarm(alarm)
		}
	}
}

func (e *AlarmEngine) TriggerAlarm(alarm model.Alarm) {
	err := e.sqliteRepo.InsertAlarm(alarm)
	if err != nil {
		fmt.Printf("insert alarm error: %v\n", err)
	}

	if e.mqttPublisher != nil {
		err = e.mqttPublisher.PublishAlarm(alarm)
		if err != nil {
			fmt.Printf("mqtt publish alarm error: %v\n", err)
		}
	}

	if e.broadcaster != nil {
		e.broadcaster.BroadcastMessage("alarm", alarm)
	}
}
