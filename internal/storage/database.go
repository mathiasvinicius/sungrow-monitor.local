package storage

import (
	"fmt"
	"path/filepath"
	"time"

	"sungrow-monitor/internal/inverter"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Database struct {
	db *gorm.DB
}

type powerSample struct {
	Timestamp        time.Time
	TotalActivePower uint32
}

func NewDatabase(path string) (*Database, error) {
	// Ensure directory exists
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		// Directory will be created by SQLite if it doesn't exist
	}

	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Auto-migrate the schema
	if err := db.AutoMigrate(&InverterReading{}); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) SaveReading(data *inverter.InverterData) error {
	reading := &InverterReading{
		Timestamp:          data.Timestamp,
		SerialNumber:       data.SerialNumber,
		DeviceTypeCode:     data.DeviceTypeCode,
		NominalPower:       data.NominalPower,
		OutputType:         data.OutputType,
		DailyEnergy:        data.DailyEnergy,
		TotalEnergy:        data.TotalEnergy,
		Temperature:        data.Temperature,
		MPPT1Voltage:       data.MPPT1Voltage,
		MPPT1Current:       data.MPPT1Current,
		MPPT2Voltage:       data.MPPT2Voltage,
		MPPT2Current:       data.MPPT2Current,
		TotalDCPower:       data.TotalDCPower,
		GridVoltage:        data.GridVoltage,
		GridFrequency:      data.GridFrequency,
		GridCurrent:        data.GridCurrent,
		TotalActivePower:   data.TotalActivePower,
		ReactivePower:      data.ReactivePower,
		PowerFactor:        data.PowerFactor,
		RunningState:       data.RunningState,
		RunningStateString: data.RunningStateString,
		FaultCode:          data.FaultCode,
		IsOnline:           data.IsOnline,
	}

	return d.db.Create(reading).Error
}

func (d *Database) GetLatestReading() (*InverterReading, error) {
	var reading InverterReading
	result := d.db.Order("timestamp desc").First(&reading)
	if result.Error != nil {
		return nil, result.Error
	}
	return &reading, nil
}

func (d *Database) GetReadingsByRange(from, to time.Time) ([]InverterReading, error) {
	var readings []InverterReading
	result := d.db.Where("timestamp BETWEEN ? AND ?", from, to).
		Order("timestamp desc").
		Find(&readings)
	if result.Error != nil {
		return nil, result.Error
	}
	return readings, nil
}

func (d *Database) GetReadingsWithLimit(limit int) ([]InverterReading, error) {
	var readings []InverterReading
	result := d.db.Order("timestamp desc").Limit(limit).Find(&readings)
	if result.Error != nil {
		return nil, result.Error
	}
	return readings, nil
}

func (d *Database) GetDailyEnergy(date time.Time) (float64, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var reading InverterReading
	result := d.db.Where("timestamp BETWEEN ? AND ?", startOfDay, endOfDay).
		Order("timestamp desc").
		First(&reading)
	if result.Error != nil {
		return 0, result.Error
	}
	return reading.DailyEnergy, nil
}

func (d *Database) GetTotalEnergy() (float64, error) {
	var reading InverterReading
	result := d.db.Order("timestamp desc").First(&reading)
	if result.Error != nil {
		return 0, result.Error
	}
	return reading.TotalEnergy, nil
}

func (d *Database) GetDailyStats(date time.Time) (*DailyStats, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var stats DailyStats
	stats.Date = startOfDay

	// Get max power
	var reading InverterReading
	result := d.db.Where("timestamp BETWEEN ? AND ?", startOfDay, endOfDay).
		Order("total_active_power desc").
		First(&reading)
	if result.Error == nil {
		stats.MaxPower = reading.TotalActivePower
	}

	// Get latest daily energy
	result = d.db.Where("timestamp BETWEEN ? AND ?", startOfDay, endOfDay).
		Order("timestamp desc").
		First(&reading)
	if result.Error == nil {
		stats.TotalEnergy = reading.DailyEnergy
	}

	// Get average temperature
	var avgTemp float64
	d.db.Model(&InverterReading{}).
		Where("timestamp BETWEEN ? AND ?", startOfDay, endOfDay).
		Select("AVG(temperature)").
		Scan(&avgTemp)
	stats.AvgTemperature = avgTemp

	// Get readings count
	d.db.Model(&InverterReading{}).
		Where("timestamp BETWEEN ? AND ?", startOfDay, endOfDay).
		Count(&stats.ReadingsCount)

	return &stats, nil
}

func (d *Database) GetAveragePowerForTimeOfDay(now time.Time, days int, bucketMinutes int) (float64, int, error) {
	if days <= 0 {
		days = 30
	}
	if bucketMinutes <= 0 {
		bucketMinutes = 30
	}

	start := now.AddDate(0, 0, -days)

	var samples []powerSample
	result := d.db.Model(&InverterReading{}).
		Select("timestamp, total_active_power").
		Where("timestamp >= ? AND timestamp <= ?", start, now).
		Find(&samples)
	if result.Error != nil {
		return 0, 0, result.Error
	}

	localNow := now.In(time.Local)
	targetMinutes := localNow.Hour()*60 + localNow.Minute()
	bucketStart := (targetMinutes / bucketMinutes) * bucketMinutes
	bucketEnd := bucketStart + bucketMinutes

	var total float64
	count := 0
	for _, sample := range samples {
		ts := sample.Timestamp.In(time.Local)
		minutes := ts.Hour()*60 + ts.Minute()
		if minutes >= bucketStart && minutes < bucketEnd {
			total += float64(sample.TotalActivePower)
			count++
		}
	}

	if count == 0 {
		return 0, 0, nil
	}
	return total / float64(count), count, nil
}

func (d *Database) CleanOldReadings(olderThan time.Duration) error {
	cutoff := time.Now().Add(-olderThan)
	return d.db.Where("timestamp < ?", cutoff).Delete(&InverterReading{}).Error
}

func (d *Database) Close() error {
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
