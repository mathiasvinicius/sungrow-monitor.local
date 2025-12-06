package collector

import (
	"context"
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

	if err := c.client.Connect(); err != nil {
		return err
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
	data, err := c.sungrow.ReadAllData()
	if err != nil {
		log.Printf("Error reading inverter data: %v", err)
		// Try to reconnect
		if reconnErr := c.client.Reconnect(); reconnErr != nil {
			log.Printf("Failed to reconnect: %v", reconnErr)
		}
		return
	}

	c.mu.Lock()
	c.latestData = data
	c.mu.Unlock()

	// Save to database
	if c.db != nil {
		if err := c.db.SaveReading(data); err != nil {
			log.Printf("Error saving reading: %v", err)
		}
	}

	// Publish to MQTT
	if c.publisher != nil {
		if err := c.publisher.Publish(data); err != nil {
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

func (c *Collector) Stop() {
	c.client.Close()
	if c.publisher != nil {
		c.publisher.Close()
	}
	if c.db != nil {
		c.db.Close()
	}
}
