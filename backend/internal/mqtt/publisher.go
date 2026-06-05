package mqtt

import (
	"encoding/json"
	"fmt"
	"power-twin-backend/internal/model"
	"time"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"
)

type MQTTPublisher struct {
	client pahomqtt.Client
}

func NewMQTTPublisher(broker string, clientID string) *MQTTPublisher {
	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(broker)
	opts.SetClientID(clientID)
	opts.SetConnectTimeout(5 * time.Second)
	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)

	client := pahomqtt.NewClient(opts)
	token := client.Connect()
	token.Wait()

	return &MQTTPublisher{client: client}
}

func (p *MQTTPublisher) PublishAlarm(alarm model.Alarm) error {
	data, err := json.Marshal(alarm)
	if err != nil {
		return err
	}
	token := p.client.Publish("power-twin/alarms", 1, false, data)
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("mqtt publish alarm error: %w", token.Error())
	}
	return nil
}

func (p *MQTTPublisher) PublishTelemetry(telemetry model.DeviceTelemetry) error {
	data, err := json.Marshal(telemetry)
	if err != nil {
		return err
	}
	token := p.client.Publish("power-twin/telemetry", 0, false, data)
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("mqtt publish telemetry error: %w", token.Error())
	}
	return nil
}
