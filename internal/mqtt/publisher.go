package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"sungrow-monitor/internal/inverter"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Publisher struct {
	client      mqtt.Client
	topicPrefix string
	enabled     bool
}

type PublisherConfig struct {
	Broker      string
	ClientID    string
	Username    string
	Password    string
	TopicPrefix string
	Enabled     bool
}

func NewPublisher(cfg PublisherConfig) (*Publisher, error) {
	if !cfg.Enabled {
		return &Publisher{enabled: false}, nil
	}

	opts := mqtt.NewClientOptions().
		AddBroker(cfg.Broker).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(5 * time.Second).
		SetConnectionLostHandler(func(c mqtt.Client, err error) {
			log.Printf("MQTT connection lost: %v", err)
		}).
		SetOnConnectHandler(func(c mqtt.Client) {
			log.Println("MQTT connected")
		})

	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	client := mqtt.NewClient(opts)
	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		return nil, fmt.Errorf("failed to connect to MQTT broker: %w", token.Error())
	}

	return &Publisher{
		client:      client,
		topicPrefix: cfg.TopicPrefix,
		enabled:     true,
	}, nil
}

func (p *Publisher) Publish(data *inverter.InverterData) error {
	if !p.enabled {
		return nil
	}

	// Publish individual values
	topics := map[string]interface{}{
		"power":           data.TotalActivePower,
		"energy_daily":    data.DailyEnergy,
		"energy_total":    data.TotalEnergy,
		"temperature":     data.Temperature,
		"mppt1_voltage":   data.MPPT1Voltage,
		"mppt1_current":   data.MPPT1Current,
		"mppt2_voltage":   data.MPPT2Voltage,
		"mppt2_current":   data.MPPT2Current,
		"dc_power":        data.TotalDCPower,
		"grid_voltage":    data.GridVoltage,
		"grid_frequency":  data.GridFrequency,
		"grid_current":    data.GridCurrent,
		"power_factor":    data.PowerFactor,
		"running_state":   data.RunningStateString,
		"is_online":       data.IsOnline,
	}

	for name, value := range topics {
		topic := fmt.Sprintf("%s/%s/%s", p.topicPrefix, "SG5.0RS-S", name)
		payload := fmt.Sprintf("%v", value)
		token := p.client.Publish(topic, 0, false, payload)
		token.Wait()
		if token.Error() != nil {
			log.Printf("Failed to publish to %s: %v", topic, token.Error())
		}
	}

	// Publish full status as JSON
	statusJSON, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal status: %w", err)
	}

	statusTopic := fmt.Sprintf("%s/%s/status", p.topicPrefix, "SG5.0RS-S")
	token := p.client.Publish(statusTopic, 0, true, statusJSON)
	token.Wait()
	if token.Error() != nil {
		return fmt.Errorf("failed to publish status: %w", token.Error())
	}

	return nil
}

func (p *Publisher) PublishHomeAssistantDiscovery() error {
	if !p.enabled {
		return nil
	}

	sensors := []struct {
		Name       string
		ID         string
		Unit       string
		DeviceClass string
		StateTopic string
	}{
		{"Power", "power", "W", "power", "power"},
		{"Daily Energy", "energy_daily", "kWh", "energy", "energy_daily"},
		{"Total Energy", "energy_total", "kWh", "energy", "energy_total"},
		{"Temperature", "temperature", "°C", "temperature", "temperature"},
		{"MPPT1 Voltage", "mppt1_voltage", "V", "voltage", "mppt1_voltage"},
		{"MPPT1 Current", "mppt1_current", "A", "current", "mppt1_current"},
		{"Grid Voltage", "grid_voltage", "V", "voltage", "grid_voltage"},
		{"Grid Frequency", "grid_frequency", "Hz", "frequency", "grid_frequency"},
		{"Power Factor", "power_factor", "", "power_factor", "power_factor"},
	}

	for _, sensor := range sensors {
		discoveryTopic := fmt.Sprintf("homeassistant/sensor/sungrow/%s/config", sensor.ID)

		config := map[string]interface{}{
			"name":                fmt.Sprintf("Sungrow %s", sensor.Name),
			"unique_id":           fmt.Sprintf("sungrow_%s", sensor.ID),
			"state_topic":         fmt.Sprintf("%s/SG5.0RS-S/%s", p.topicPrefix, sensor.StateTopic),
			"unit_of_measurement": sensor.Unit,
			"device": map[string]interface{}{
				"identifiers":  []string{"sungrow_sg5rs"},
				"name":         "Sungrow SG5.0RS-S",
				"manufacturer": "Sungrow",
				"model":        "SG5.0RS-S",
			},
		}

		if sensor.DeviceClass != "" {
			config["device_class"] = sensor.DeviceClass
		}

		payload, _ := json.Marshal(config)
		token := p.client.Publish(discoveryTopic, 0, true, payload)
		token.Wait()
	}

	return nil
}

func (p *Publisher) IsConnected() bool {
	if !p.enabled {
		return false
	}
	return p.client.IsConnected()
}

func (p *Publisher) Close() {
	if p.enabled && p.client != nil {
		p.client.Disconnect(1000)
	}
}
