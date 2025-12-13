package api

import (
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	"sungrow-monitor/internal/collector"
	"sungrow-monitor/internal/storage"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router    *gin.Engine
	server    *http.Server
	collector *collector.Collector
	db        *storage.Database
	port      int
	webPath   string
}

type ServerConfig struct {
	Port      int
	Collector *collector.Collector
	Database  *storage.Database
	WebPath   string
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
		router:    router,
		collector: cfg.Collector,
		db:        cfg.Database,
		port:      cfg.Port,
		webPath:   webPath,
	}

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

	// Health check
	s.router.GET("/health", s.healthHandler)

	// API routes
	api := s.router.Group("/api/v1")
	{
		api.GET("/status", s.statusHandler)
		api.GET("/readings", s.readingsHandler)
		api.GET("/readings/latest", s.latestReadingHandler)
		api.GET("/energy/daily", s.dailyEnergyHandler)
		api.GET("/energy/total", s.totalEnergyHandler)
		api.GET("/stats/daily", s.dailyStatsHandler)
	}
}

func (s *Server) dashboardHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "Sungrow Monitor",
	})
}

func (s *Server) historyHandler(c *gin.Context) {
	c.HTML(http.StatusOK, "history.html", gin.H{
		"title": "Sungrow Monitor - Historico",
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
