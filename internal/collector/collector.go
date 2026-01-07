package collector

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"sungrow-monitor/internal/inverter"
	"sungrow-monitor/internal/modbus"
	"sungrow-monitor/internal/mqtt"
	"sungrow-monitor/internal/storage"
)

type Collector struct {
	client    *modbus.Client
	sungrow   *inverter.Sungrow
	db        *storage.Database
	publisher *mqtt.Publisher
	interval  time.Duration
	enabled   bool

	mu          sync.RWMutex
	latestData  *inverter.InverterData
	isCollecting bool
}

type CollectorConfig struct {
	Client    *modbus.Client
	Database  *storage.Database
	Publisher *mqtt.Publisher
	Interval  time.Duration
	Enabled   bool
}

func NewCollector(cfg CollectorConfig) *Collector {
	return &Collector{
		client:    cfg.Client,
		sungrow:   inverter.NewSungrow(cfg.Client),
		db:        cfg.Database,
		publisher: cfg.Publisher,
		interval:  cfg.Interval,
		enabled:   cfg.Enabled,
	}
}

func (c *Collector) Start(ctx context.Context) error {
	if !c.enabled {
		log.Println("Collector is disabled")
		return nil
	}

	c.mu.Lock()
	c.isCollecting = true
	c.mu.Unlock()

	log.Printf("Starting collector with interval %s", c.interval)

	// Initial collection
	c.collect()

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Collector stopped")
			c.mu.Lock()
			c.isCollecting = false
			c.mu.Unlock()
			return nil
		case <-ticker.C:
			c.collect()
		}
	}
}

func (c *Collector) collect() {
	c.mu.RLock()
	client := c.client
	sungrow := c.sungrow
	db := c.db
	publisher := c.publisher

	if client == nil || sungrow == nil {
		c.mu.RUnlock()
		log.Printf("Collector not initialized (client or inverter is nil)")
		return
	}

	if err := client.Connect(); err != nil {
		c.mu.RUnlock()
		log.Printf("Error connecting to inverter: %v", err)
		return
	}

	data, err := sungrow.ReadAllData()
	if err != nil {
		log.Printf("Error reading inverter data: %v", err)
		// Try to reconnect
		if reconnErr := client.Reconnect(); reconnErr != nil {
			c.mu.RUnlock()
			log.Printf("Failed to reconnect: %v", reconnErr)
			return
		}
		// Retry once after successful reconnect
		data, err = sungrow.ReadAllData()
		if err != nil {
			c.mu.RUnlock()
			log.Printf("Error reading inverter data after reconnect: %v", err)
			return
		}
	}
	c.mu.RUnlock()

	c.mu.Lock()
	c.latestData = data
	c.mu.Unlock()

	// Save to database
	if db != nil {
		if err := db.SaveReading(data); err != nil {
			log.Printf("Error saving reading: %v", err)
		}
	}

	// Publish to MQTT
	if publisher != nil {
		if err := publisher.Publish(data); err != nil {
			log.Printf("Error publishing to MQTT: %v", err)
		}
	}

	log.Printf("Collected: Power=%dW, Daily=%.1fkWh, Total=%.1fkWh, Temp=%.1f°C",
		data.TotalActivePower, data.DailyEnergy, data.TotalEnergy, data.Temperature)
}

func (c *Collector) GetLatestData() *inverter.InverterData {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.latestData
}

func (c *Collector) IsCollecting() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isCollecting
}

func (c *Collector) CollectOnce() (*inverter.InverterData, error) {
	if !c.client.IsConnected() {
		if err := c.client.Connect(); err != nil {
			return nil, err
		}
	}

	data, err := c.sungrow.ReadAllData()
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.latestData = data
	c.mu.Unlock()

	return data, nil
}

// UpdateInverterConfig atualiza a configuracao do inversor em runtime
// Para coleta, fecha cliente antigo, cria novo cliente, reinicia
func (c *Collector) UpdateInverterConfig(ip string, port int, slaveID uint8, timeout time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	log.Printf("Updating inverter configuration: %s:%d (slave=%d)", ip, port, slaveID)

	// Guardar cliente antigo para rollback se necessario
	oldClient := c.client

	// Criar novo cliente
	newClient := modbus.NewClient(ip, port, slaveID, timeout)

	// Testar conexao antes de aplicar
	if err := newClient.Connect(); err != nil {
		log.Printf("Failed to connect with new config: %v", err)
		return fmt.Errorf("failed to connect with new configuration: %w", err)
	}

	// Testar leitura de dados
	newSungrow := inverter.NewSungrow(newClient)
	data, err := newSungrow.ReadAllData()
	if err != nil {
		newClient.Close()
		log.Printf("Failed to read data with new config: %v", err)
		return fmt.Errorf("failed to read data with new configuration: %w", err)
	}

	// Sucesso! Fechar cliente antigo e atualizar
	if oldClient != nil {
		oldClient.Close()
	}

	c.client = newClient
	c.sungrow = newSungrow
	c.latestData = data

	log.Printf("Inverter configuration updated successfully")
	return nil
}

func (c *Collector) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.client != nil {
		c.client.Close()
	}
	if c.publisher != nil {
		c.publisher.Close()
	}
	if c.db != nil {
		c.db.Close()
	}
}
