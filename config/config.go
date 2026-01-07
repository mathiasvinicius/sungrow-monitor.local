package config

import (
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Inverter  InverterConfig  `mapstructure:"inverter"`
	Collector CollectorConfig `mapstructure:"collector"`
	API       APIConfig       `mapstructure:"api"`
	MQTT      MQTTConfig      `mapstructure:"mqtt"`
	Database  DatabaseConfig  `mapstructure:"database"`
	Weather   WeatherConfig   `mapstructure:"weather"`
}

type InverterConfig struct {
	IP      string        `mapstructure:"ip"`
	Port    int           `mapstructure:"port"`
	SlaveID uint8         `mapstructure:"slave_id"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type CollectorConfig struct {
	Interval time.Duration `mapstructure:"interval"`
	Enabled  bool          `mapstructure:"enabled"`
}

type APIConfig struct {
	Port    int    `mapstructure:"port"`
	Enabled bool   `mapstructure:"enabled"`
	WebPath string `mapstructure:"web_path"`
}

type MQTTConfig struct {
	Enabled     bool   `mapstructure:"enabled"`
	Broker      string `mapstructure:"broker"`
	TopicPrefix string `mapstructure:"topic_prefix"`
	ClientID    string `mapstructure:"client_id"`
	Username    string `mapstructure:"username"`
	Password    string `mapstructure:"password"`
}

type DatabaseConfig struct {
	Path string `mapstructure:"path"`
}

type WeatherConfig struct {
	Enabled   bool    `mapstructure:"enabled"`
	Provider  string  `mapstructure:"provider"`
	APIKey    string  `mapstructure:"api_key"`
	City      string  `mapstructure:"city"`
	Country   string  `mapstructure:"country"`
	Latitude  float64 `mapstructure:"latitude"`
	Longitude float64 `mapstructure:"longitude"`
	Units     string  `mapstructure:"units"`
}

func Load(configPath string) (*Config, error) {
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/sungrow-monitor")
	}

	// Set defaults
	viper.SetDefault("inverter.ip", "172.16.0.120")
	viper.SetDefault("inverter.port", 502)
	viper.SetDefault("inverter.slave_id", 1)
	viper.SetDefault("inverter.timeout", "10s")
	viper.SetDefault("collector.interval", "30s")
	viper.SetDefault("collector.enabled", true)
	viper.SetDefault("api.port", 8045)
	viper.SetDefault("api.enabled", true)
	viper.SetDefault("api.web_path", "./web")
	viper.SetDefault("mqtt.enabled", true)
	viper.SetDefault("mqtt.broker", "tcp://localhost:1883")
	viper.SetDefault("mqtt.topic_prefix", "sungrow")
	viper.SetDefault("mqtt.client_id", "sungrow-monitor")
	viper.SetDefault("database.path", "./sungrow.db")
	viper.SetDefault("weather.enabled", false)
	viper.SetDefault("weather.provider", "openweather")
	viper.SetDefault("weather.api_key", "")
	viper.SetDefault("weather.city", "")
	viper.SetDefault("weather.country", "")
	viper.SetDefault("weather.latitude", 0)
	viper.SetDefault("weather.longitude", 0)
	viper.SetDefault("weather.units", "metric")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
