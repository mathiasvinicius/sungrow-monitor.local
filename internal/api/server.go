package api

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"sungrow-monitor/config"
	"sungrow-monitor/internal/collector"
	"sungrow-monitor/internal/inverter"
	"sungrow-monitor/internal/modbus"
	"sungrow-monitor/internal/storage"
	"sungrow-monitor/internal/weather"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
)

type Server struct {
	router      *gin.Engine
	server      *http.Server
	collector   *collector.Collector
	db          *storage.Database
	port        int
	webPath     string
	config      *config.Config
	configPath  string
	configMutex sync.RWMutex
	weatherMu   sync.Mutex
	weather     weather.Provider
	weatherData *weather.Data
	weatherAt   time.Time
}

type ServerConfig struct {
	Port       int
	Collector  *collector.Collector
	Database   *storage.Database
	WebPath    string
	Config     *config.Config
	ConfigPath string
}

func NewServer(cfg ServerConfig) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(gin.Recovery())
	router.Use(gin.Logger())

	// Default web path
	webPath := cfg.WebPath
	if webPath == "" {
		webPath = "./web"
	}

	s := &Server{
		router:     router,
		collector:  cfg.Collector,
		db:         cfg.Database,
		port:       cfg.Port,
		webPath:    webPath,
		config:     cfg.Config,
		configPath: cfg.ConfigPath,
	}

	s.initWeatherProvider()
	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	// Load HTML templates
	tmpl := template.Must(template.ParseGlob(s.webPath + "/templates/*.html"))
	s.router.SetHTMLTemplate(tmpl)

	// Serve static files
	s.router.Static("/static", s.webPath+"/static")

	// Dashboard routes
	s.router.GET("/", s.dashboardHandler)
	s.router.GET("/dashboard", s.dashboardHandler)
	s.router.GET("/history", s.historyHandler)
	s.router.GET("/settings", s.settingsHandler)
	s.router.HEAD("/", s.dashboardHandler)
	s.router.HEAD("/dashboard", s.dashboardHandler)
	s.router.HEAD("/history", s.historyHandler)
	s.router.HEAD("/settings", s.settingsHandler)

	// Health check
	s.router.GET("/health", s.healthHandler)

	// API routes
	api := s.router.Group("/api/v1")
	{
		api.GET("/background/wallpaper", s.backgroundWallpaperHandler)
		api.GET("/bing-wallpaper", s.bingWallpaperHandler)
		api.GET("/status", s.statusHandler)
		api.GET("/readings", s.readingsHandler)
		api.GET("/readings/latest", s.latestReadingHandler)
		api.GET("/energy/daily", s.dailyEnergyHandler)
		api.GET("/energy/total", s.totalEnergyHandler)
		api.GET("/stats/daily", s.dailyStatsHandler)
		api.GET("/insights/production", s.productionInsightsHandler)

		// Config routes
		api.GET("/config/inverter", s.getInverterConfigHandler)
		api.PUT("/config/inverter", s.updateInverterConfigHandler)
		api.POST("/config/inverter/test", s.testInverterConfigHandler)
		api.GET("/config/weather", s.getWeatherConfigHandler)
		api.PUT("/config/weather", s.updateWeatherConfigHandler)
		api.GET("/config/background", s.getBackgroundConfigHandler)
		api.PUT("/config/background", s.updateBackgroundConfigHandler)
	}
}

func (s *Server) dashboardHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Sungrow Monitor",
	})
}

func (s *Server) historyHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "history.html", gin.H{
		"title": "Sungrow Monitor - Histórico",
	})
}

func (s *Server) Start() error {
	s.server = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	log.Printf("API server starting on port %d", s.port)
	return s.server.ListenAndServe()
}

func (s *Server) Stop(ctx context.Context) error {
	if s.server != nil {
		return s.server.Shutdown(ctx)
	}
	return nil
}

func (s *Server) healthHandler(c *gin.Context) {
	status := "healthy"
	inverterOnline := false

	if data := s.collector.GetLatestData(); data != nil {
		inverterOnline = data.IsOnline
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          status,
		"inverter_online": inverterOnline,
		"collecting":      s.collector.IsCollecting(),
		"timestamp":       time.Now(),
	})
}

func (s *Server) statusHandler(c *gin.Context) {
	data := s.collector.GetLatestData()
	if data == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "No data available yet",
		})
		return
	}
	c.JSON(http.StatusOK, data)
}

func (s *Server) readingsHandler(c *gin.Context) {
	fromStr := c.Query("from")
	toStr := c.Query("to")
	limitStr := c.DefaultQuery("limit", "100")

	var limit int
	fmt.Sscanf(limitStr, "%d", &limit)
	if limit <= 0 || limit > 1000 {
		limit = 100
	}

	if fromStr != "" && toStr != "" {
		from, err := time.Parse(time.RFC3339, fromStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' date format"})
			return
		}
		to, err := time.Parse(time.RFC3339, toStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' date format"})
			return
		}

		readings, err := s.db.GetReadingsByRange(from, to)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, readings)
		return
	}

	readings, err := s.db.GetReadingsWithLimit(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, readings)
}

func (s *Server) latestReadingHandler(c *gin.Context) {
	reading, err := s.db.GetLatestReading()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, reading)
}

func (s *Server) dailyEnergyHandler(c *gin.Context) {
	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	energy, err := s.db.GetDailyEnergy(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"date":         dateStr,
		"energy_kwh":   energy,
	})
}

func (s *Server) totalEnergyHandler(c *gin.Context) {
	energy, err := s.db.GetTotalEnergy()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"total_energy_kwh": energy,
	})
}

func (s *Server) dailyStatsHandler(c *gin.Context) {
	dateStr := c.DefaultQuery("date", time.Now().Format("2006-01-02"))
	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid date format"})
		return
	}

	stats, err := s.db.GetDailyStats(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}

const (
	insightHistoryDays  = 30
	insightBucketMinutes = 30
	insightMinSamples   = 20
	insightLowRatio     = 0.4
	weatherCacheTTL     = 10 * time.Minute
)

func (s *Server) productionInsightsHandler(c *gin.Context) {
	latest := s.collector.GetLatestData()
	if latest == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "No data available yet",
		})
		return
	}

	now := time.Now()
	avg, samples, err := s.db.GetAveragePowerForTimeOfDay(now, insightHistoryDays, insightBucketMinutes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	weatherData := s.getWeather(now)
	daylight := s.isDaylight(now, weatherData)

	status := "unknown"
	message := ""
	ratio := 0.0
	weatherLabel := ""

	if !daylight {
		status = "night"
		message = "Noite: comparação desativada"
	} else if samples < insightMinSamples || avg <= 0 {
		status = "insufficient_history"
		message = "Histórico insuficiente para comparar"
	} else {
		ratio = float64(latest.TotalActivePower) / avg
		weatherLabel = classifyWeather(weatherData)
		if ratio < insightLowRatio {
			if weatherLabel != "" {
				status = "low_power_weather"
				message = fmt.Sprintf("Baixa geração compatível com %s", weatherLabel)
			} else {
				status = "low_power_unexpected"
				message = "Baixa geração fora do esperado"
			}
		} else {
			status = "normal"
			message = "Geração dentro do esperado"
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":          status,
		"message":         message,
		"actual_power_w":  latest.TotalActivePower,
		"expected_avg_w":  avg,
		"ratio":           ratio,
		"threshold":       insightLowRatio,
		"samples":         samples,
		"window_days":     insightHistoryDays,
		"bucket_minutes":  insightBucketMinutes,
		"daylight":        daylight,
		"weather":         weatherData,
		"weather_label":   weatherLabel,
		"timestamp":       now,
	})
}

// Settings page handler
func (s *Server) settingsHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "settings.html", gin.H{
		"title": "Sungrow Monitor - Configurações",
	})
}

// InverterConfigResponse represents the inverter configuration
type InverterConfigResponse struct {
	IP             string `json:"ip"`
	Port           int    `json:"port"`
	SlaveID        uint8  `json:"slave_id"`
	TimeoutSeconds int    `json:"timeout_seconds"`
}

// InverterConfigRequest represents a configuration update request
type InverterConfigRequest struct {
	IP             string `json:"ip" binding:"required"`
	Port           int    `json:"port" binding:"required,min=1,max=65535"`
	SlaveID        uint8  `json:"slave_id" binding:"required,min=1,max=255"`
	TimeoutSeconds int    `json:"timeout_seconds" binding:"required,min=1,max=60"`
}

type WeatherConfigResponse struct {
	Enabled   bool    `json:"enabled"`
	Provider  string  `json:"provider"`
	APIKey    string  `json:"api_key"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Units     string  `json:"units"`
}

type WeatherConfigRequest struct {
	Enabled   bool    `json:"enabled"`
	Provider  string  `json:"provider"`
	APIKey    string  `json:"api_key"`
	City      string  `json:"city"`
	Country   string  `json:"country"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Units     string  `json:"units"`
}

// Get current inverter configuration
func (s *Server) getInverterConfigHandler(c *gin.Context) {
	s.configMutex.RLock()
	defer s.configMutex.RUnlock()

	timeoutSec := int(s.config.Inverter.Timeout.Seconds())

	c.JSON(http.StatusOK, InverterConfigResponse{
		IP:             s.config.Inverter.IP,
		Port:           s.config.Inverter.Port,
		SlaveID:        s.config.Inverter.SlaveID,
		TimeoutSeconds: timeoutSec,
	})
}

// Test inverter configuration without saving
func (s *Server) testInverterConfigHandler(c *gin.Context) {
	var req InverterConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
		return
	}

	// Create temporary client to test connection
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	testClient := modbus.NewClient(req.IP, req.Port, req.SlaveID, timeout)

	if err := testClient.Connect(); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Connection failed: %v", err),
		})
		return
	}
	defer testClient.Close()

	// Try to read basic info to verify it's a Sungrow inverter
	testInverter := inverter.NewSungrow(testClient)
	data, err := testInverter.ReadAllData()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   fmt.Sprintf("Failed to read data: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":       true,
		"serial_number": data.SerialNumber,
		"device_type":   data.DeviceTypeCode,
		"message":       "Connection successful",
	})
}

// Update inverter configuration
func (s *Server) updateInverterConfigHandler(c *gin.Context) {
	var req InverterConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// First, test the new configuration
	timeout := time.Duration(req.TimeoutSeconds) * time.Second
	testClient := modbus.NewClient(req.IP, req.Port, req.SlaveID, timeout)

	if err := testClient.Connect(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": fmt.Sprintf("Configuration test failed: %v", err),
		})
		return
	}
	testClient.Close()

	// Update collector with new configuration
	if err := s.collector.UpdateInverterConfig(req.IP, req.Port, req.SlaveID, timeout); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("Failed to apply configuration: %v", err),
		})
		return
	}

	// Update in-memory config
	s.configMutex.Lock()
	s.config.Inverter.IP = req.IP
	s.config.Inverter.Port = req.Port
	s.config.Inverter.SlaveID = req.SlaveID
	s.config.Inverter.Timeout = timeout
	s.configMutex.Unlock()

	// Persist to config file
	if err := s.saveConfigToFile(); err != nil {
		log.Printf("Warning: Failed to save config to file: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Configuration applied but not persisted to file",
			"warning": err.Error(),
		})
		return
	}

	log.Printf("Inverter configuration updated: %s:%d (slave=%d)", req.IP, req.Port, req.SlaveID)

	c.JSON(http.StatusOK, gin.H{
		"message": "Configuration updated successfully",
	})
}

func (s *Server) getWeatherConfigHandler(c *gin.Context) {
	s.configMutex.RLock()
	defer s.configMutex.RUnlock()

	cfg := s.config.Weather
	c.JSON(http.StatusOK, WeatherConfigResponse{
		Enabled:   cfg.Enabled,
		Provider:  cfg.Provider,
		APIKey:    cfg.APIKey,
		City:      cfg.City,
		Country:   cfg.Country,
		Latitude:  cfg.Latitude,
		Longitude: cfg.Longitude,
		Units:     cfg.Units,
	})
}

func (s *Server) updateWeatherConfigHandler(c *gin.Context) {
	var req WeatherConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.TrimSpace(req.Provider) == "" {
		req.Provider = "openweather"
	}
	if strings.TrimSpace(req.Units) == "" {
		req.Units = "metric"
	}

	s.configMutex.Lock()
	s.config.Weather.Enabled = req.Enabled
	s.config.Weather.Provider = req.Provider
	s.config.Weather.APIKey = req.APIKey
	s.config.Weather.City = req.City
	s.config.Weather.Country = req.Country
	s.config.Weather.Latitude = req.Latitude
	s.config.Weather.Longitude = req.Longitude
	s.config.Weather.Units = req.Units
	s.configMutex.Unlock()

	s.initWeatherProvider()

	if err := s.saveConfigToFile(); err != nil {
		log.Printf("Warning: Failed to save config to file: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"message": "Configuration applied but not persisted to file",
			"warning": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Weather configuration updated successfully",
	})
}

// Save configuration to YAML file
func (s *Server) saveConfigToFile() error {
	s.configMutex.RLock()
	defer s.configMutex.RUnlock()

	// Determine config file path
	configPath := s.configPath
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Read existing file to preserve structure and comments
	viper.SetConfigFile(configPath)

	// Update values
	viper.Set("inverter.ip", s.config.Inverter.IP)
	viper.Set("inverter.port", s.config.Inverter.Port)
	viper.Set("inverter.slave_id", s.config.Inverter.SlaveID)
	viper.Set("inverter.timeout", s.config.Inverter.Timeout.String())
	viper.Set("weather.enabled", s.config.Weather.Enabled)
	viper.Set("weather.provider", s.config.Weather.Provider)
	viper.Set("weather.api_key", s.config.Weather.APIKey)
	viper.Set("weather.city", s.config.Weather.City)
	viper.Set("weather.country", s.config.Weather.Country)
	viper.Set("weather.latitude", s.config.Weather.Latitude)
	viper.Set("weather.longitude", s.config.Weather.Longitude)
	viper.Set("weather.units", s.config.Weather.Units)

	// Write back to file
	return viper.WriteConfig()
}

func (s *Server) initWeatherProvider() {
	s.weatherMu.Lock()
	defer s.weatherMu.Unlock()

	s.weather = nil
	s.weatherData = nil
	s.weatherAt = time.Time{}

	s.configMutex.RLock()
	cfg := s.config.Weather
	s.configMutex.RUnlock()

	if !cfg.Enabled {
		return
	}

	switch strings.ToLower(cfg.Provider) {
	case "openweather":
		s.weather = weather.NewOpenWeatherClient(
			cfg.APIKey,
			cfg.City,
			cfg.Country,
			cfg.Latitude,
			cfg.Longitude,
			cfg.Units,
		)
	case "openmeteo", "open-meteo", "open_meteo":
		s.weather = weather.NewOpenMeteoClient(
			cfg.City,
			cfg.Country,
			cfg.Latitude,
			cfg.Longitude,
			cfg.Units,
		)
	default:
		log.Printf("Weather provider not supported: %s", cfg.Provider)
	}
}

func (s *Server) getWeather(now time.Time) *weather.Data {
	s.weatherMu.Lock()
	defer s.weatherMu.Unlock()

	if s.weather == nil {
		return nil
	}

	if s.weatherData != nil && now.Sub(s.weatherAt) < weatherCacheTTL {
		return s.weatherData
	}

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()

	data, err := s.weather.Get(ctx)
	if err != nil {
		log.Printf("Weather fetch failed: %v", err)
		return s.weatherData
	}

	s.weatherData = data
	s.weatherAt = now
	return data
}

func (s *Server) isDaylight(now time.Time, data *weather.Data) bool {
	if data != nil {
		return data.IsDaylight(now)
	}
	hour := now.Hour()
	return hour >= 6 && hour < 18
}

func classifyWeather(data *weather.Data) string {
	if data == nil {
		return ""
	}

	rain := data.Rain1h
	if data.Rain3h/3 > rain {
		rain = data.Rain3h / 3
	}

	condition := strings.ToLower(data.Condition)
	if strings.Contains(condition, "thunder") {
		return "temporal"
	}

	if rain >= 5 {
		return "chuva forte"
	}
	if rain >= 1 {
		return "chuva"
	}
	if data.Clouds >= 80 {
		return "céu encoberto"
	}
	if data.Clouds >= 50 {
		return "nublado"
	}
	if data.Clouds > 0 {
		return "poucas nuvens"
	}
	return "céu limpo"
}
