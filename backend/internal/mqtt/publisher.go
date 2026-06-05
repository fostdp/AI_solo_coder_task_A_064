package mqtt

import (
	"encoding/json"
	"fmt"
	"power-twin-backend/internal/model"
	"sync"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTPublisher struct {
	client       pahomqtt.Client
	broker       string
	clientID     string
	alarmStore   []model.Alarm
	storeMutex   sync.Mutex
	maxStoreSize int
}

func NewMQTTPublisher(broker string, clientID string) *MQTTPublisher {
	p := &MQTTPublisher{
		broker:       broker,
		clientID:     clientID,
		alarmStore:   make([]model.Alarm, 0),
		maxStoreSize: 1000,
	}

	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetConnectRetryInterval(5 * time.Second)
	opts.SetCleanSession(false)
	opts.SetOrderMatters(false)

	opts.SetWill("power-twin/status", mustMarshal(map[string]string{
		"client_id": clientID,
		"status":    "offline",
		"timestamp": time.Now().Format(time.RFC3339),
	}), 1, true)

	opts.SetOnConnectHandler(func(c pahomqtt.Client) {
		c.Publish("power-twin/status", 1, true, mustMarshal(map[string]string{
			"client_id": clientID,
			"status":    "online",
			"timestamp": time.Now().Format(time.RFC3339),
		}))
		p.flushPendingAlarms(c)
	})

	client := pahomqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	return p
}

func (p *MQTTPublisher) PublishAlarm(alarm model.Alarm) error {
	data, err := json.Marshal(alarm)
	if err != nil {
		p.storePendingAlarm(alarm)
		return err
	}

	if !p.client.IsConnected() {
		p.storePendingAlarm(alarm)
		return fmt.Errorf("mqtt not connected, alarm stored for later delivery")
	}

	token := p.client.Publish("power-twin/alarms", 1, false, data)
	token.Wait()
	if token.Error() != nil {
		p.storePendingAlarm(alarm)
		return fmt.Errorf("mqtt publish alarm error: %w", token.Error())
	}

	token = p.client.Publish("power-twin/alarms/retain", 1, true, data)
	token.Wait()

	return nil
}

func (p *MQTTPublisher) PublishTelemetry(telemetry model.DeviceTelemetry) error {
	data, err := json.Marshal(telemetry)
	if err != nil {
		return err
	}
	if !p.client.IsConnected() {
		return fmt.Errorf("mqtt not connected")
	}
	token := p.client.Publish("power-twin/telemetry", 0, false, data)
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("mqtt publish telemetry error: %w", token.Error())
	}
	return nil
}

func (p *MQTTPublisher) storePendingAlarm(alarm model.Alarm) {
	p.storeMutex.Lock()
	defer p.storeMutex.Unlock()
	if len(p.alarmStore) >= p.maxStoreSize {
		p.alarmStore = p.alarmStore[1:]
	}
	p.alarmStore = append(p.alarmStore, alarm)
}

func (p *MQTTPublisher) flushPendingAlarms(c pahomqtt.Client) {
	p.storeMutex.Lock()
	pending := make([]model.Alarm, len(p.alarmStore))
	copy(pending, p.alarmStore)
	p.alarmStore = p.alarmStore[:0]
	p.storeMutex.Unlock()

	for _, alarm := range pending {
		data, err := json.Marshal(alarm)
		if err != nil {
			continue
		}
		token := c.Publish("power-twin/alarms", 1, false, data)
		token.Wait()
		if token.Error() != nil {
			p.storePendingAlarm(alarm)
			return
		}
	}
}

func mustMarshal(v interface{}) []byte {
	data, _ := json.Marshal(v)
	return data
}
