package api

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	backgroundConfigPath   = "/data/background.json"
	unsplashWallpaperTTL   = 2 * time.Hour
	defaultBackgroundQuery = "sky landscape"
)

type backgroundConfig struct {
	UnsplashAccessKey string `json:"unsplash_access_key"`
}

type backgroundConfigResponse struct {
	HasUnsplashKey bool   `json:"has_unsplash_key"`
	Provider       string `json:"provider"`
}

type backgroundConfigRequest struct {
	UnsplashAccessKey *string `json:"unsplash_access_key"`
	ClearUnsplashKey  bool    `json:"clear_unsplash_key"`
}

type backgroundWallpaperPayload struct {
	Provider string `json:"provider"`
	URL      string `json:"url"`
	Title    string `json:"title,omitempty"`
	Credit   string `json:"credit,omitempty"`
	Query    string `json:"query,omitempty"`
}

type unsplashResponse struct {
	Urls struct {
		Regular string `json:"regular"`
		Full    string `json:"full"`
	} `json:"urls"`
	User struct {
		Name  string `json:"name"`
		Links struct {
			HTML string `json:"html"`
		} `json:"links"`
	} `json:"user"`
	Description    string `json:"description"`
	AltDescription string `json:"alt_description"`
}

type unsplashCacheEntry struct {
	FetchedAt time.Time
	Payload   backgroundWallpaperPayload
}

type backgroundChoice struct {
	UnsplashQuery string
	BingIndex     int
}

var (
	backgroundConfigMu sync.Mutex
	unsplashCacheMu    sync.Mutex
	unsplashCache      = map[string]unsplashCacheEntry{}
)

func (s *Server) getBackgroundConfigHandler(c *gin.Context) {
	cfg, err := loadBackgroundConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := backgroundConfigResponse{
		HasUnsplashKey: strings.TrimSpace(cfg.UnsplashAccessKey) != "",
		Provider:       "bing",
	}
	if response.HasUnsplashKey {
		response.Provider = "unsplash"
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) updateBackgroundConfigHandler(c *gin.Context) {
	var req backgroundConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg, err := loadBackgroundConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if req.ClearUnsplashKey {
		cfg.UnsplashAccessKey = ""
	} else if req.UnsplashAccessKey != nil {
		cfg.UnsplashAccessKey = strings.TrimSpace(*req.UnsplashAccessKey)
	}

	if err := saveBackgroundConfig(cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := backgroundConfigResponse{
		HasUnsplashKey: strings.TrimSpace(cfg.UnsplashAccessKey) != "",
		Provider:       "bing",
	}
	if response.HasUnsplashKey {
		response.Provider = "unsplash"
	}

	c.JSON(http.StatusOK, response)
}

func (s *Server) backgroundWallpaperHandler(c *gin.Context) {
	cfg, err := loadBackgroundConfig()
	if err != nil {
		log.Printf("Background config load failed: %v", err)
	}

	label := ""
	if weather := s.getWeather(time.Now()); weather != nil {
		label = classifyWeather(weather)
		if label == "" {
			label = weather.Description
		}
	}

	choice := pickBackgroundChoice(label)

	if strings.TrimSpace(cfg.UnsplashAccessKey) != "" {
		payload, err := getUnsplashWallpaper(c.Request.Context(), cfg.UnsplashAccessKey, choice.UnsplashQuery)
		if err == nil {
			c.JSON(http.StatusOK, payload)
			return
		}
		log.Printf("Unsplash fetch failed, falling back to Bing: %v", err)
	}

	market := sanitizeBingMarket(c.Query("mkt"))
	bingPayload, err := getBingWallpaper(c.Request.Context(), market, choice.BingIndex)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": "Failed to fetch wallpaper", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, backgroundWallpaperPayload{
		Provider: "bing",
		URL:      bingPayload.URL,
		Title:    bingPayload.Title,
		Credit:   bingPayload.Copyright,
		Query:    choice.UnsplashQuery,
	})
}

func loadBackgroundConfig() (backgroundConfig, error) {
	backgroundConfigMu.Lock()
	defer backgroundConfigMu.Unlock()

	data, err := os.ReadFile(backgroundConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			return backgroundConfig{}, nil
		}
		return backgroundConfig{}, err
	}

	var cfg backgroundConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return backgroundConfig{}, err
	}
	return cfg, nil
}

func saveBackgroundConfig(cfg backgroundConfig) error {
	backgroundConfigMu.Lock()
	defer backgroundConfigMu.Unlock()

	if err := os.MkdirAll("/data", 0755); err != nil {
		return err
	}
	payload, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(backgroundConfigPath, payload, 0600)
}

func getUnsplashWallpaper(ctx context.Context, accessKey, query string) (backgroundWallpaperPayload, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		query = defaultBackgroundQuery
	}

	cacheKey := query
	now := time.Now()
	unsplashCacheMu.Lock()
	entry, ok := unsplashCache[cacheKey]
	unsplashCacheMu.Unlock()
	if ok && now.Sub(entry.FetchedAt) < unsplashWallpaperTTL {
		return entry.Payload, nil
	}

	payload, err := fetchUnsplashWallpaper(ctx, accessKey, query)
	if err != nil {
		if ok {
			return entry.Payload, nil
		}
		return backgroundWallpaperPayload{}, err
	}

	unsplashCacheMu.Lock()
	unsplashCache[cacheKey] = unsplashCacheEntry{
		FetchedAt: now,
		Payload:   payload,
	}
	unsplashCacheMu.Unlock()
	return payload, nil
}

func fetchUnsplashWallpaper(ctx context.Context, accessKey, query string) (backgroundWallpaperPayload, error) {
	params := url.Values{}
	params.Set("query", query)
	params.Set("orientation", "landscape")

	endpoint := url.URL{
		Scheme:   "https",
		Host:     "api.unsplash.com",
		Path:     "/photos/random",
		RawQuery: params.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return backgroundWallpaperPayload{}, fmt.Errorf("unsplash request: %w", err)
	}
	req.Header.Set("Authorization", "Client-ID "+strings.TrimSpace(accessKey))
	req.Header.Set("Accept-Version", "v1")
	req.Header.Set("User-Agent", "SungrowMonitor/1.0 (+https://github.com/mathiasvinicius/sungrow-monitor.local)")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return backgroundWallpaperPayload{}, fmt.Errorf("unsplash request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return backgroundWallpaperPayload{}, fmt.Errorf("unsplash bad status: %s", resp.Status)
	}

	var payload unsplashResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return backgroundWallpaperPayload{}, fmt.Errorf("unsplash decode: %w", err)
	}

	imageURL := strings.TrimSpace(payload.Urls.Regular)
	if imageURL == "" {
		imageURL = strings.TrimSpace(payload.Urls.Full)
	}
	if imageURL == "" {
		return backgroundWallpaperPayload{}, fmt.Errorf("unsplash image URL is missing")
	}

	title := strings.TrimSpace(payload.Description)
	if title == "" {
		title = strings.TrimSpace(payload.AltDescription)
	}

	credit := ""
	author := strings.TrimSpace(payload.User.Name)
	if author != "" {
		credit = fmt.Sprintf("Foto por %s / Unsplash", author)
	}

	return backgroundWallpaperPayload{
		Provider: "unsplash",
		URL:      imageURL,
		Title:    title,
		Credit:   credit,
		Query:    query,
	}, nil
}

func pickBackgroundChoice(label string) backgroundChoice {
	normalized := normalizeBackgroundLabel(label)
	if normalized == "" {
		return backgroundChoice{UnsplashQuery: defaultBackgroundQuery, BingIndex: 0}
	}

	if strings.Contains(normalized, "temporal") || strings.Contains(normalized, "trovoada") {
		return backgroundChoice{UnsplashQuery: "thunderstorm sky", BingIndex: 6}
	}
	if strings.Contains(normalized, "chuva forte") {
		return backgroundChoice{UnsplashQuery: "heavy rain clouds", BingIndex: 5}
	}
	if strings.Contains(normalized, "chuva") {
		return backgroundChoice{UnsplashQuery: "rainy sky", BingIndex: 4}
	}
	if strings.Contains(normalized, "nevoeiro") || strings.Contains(normalized, "neblina") || strings.Contains(normalized, "fog") {
		return backgroundChoice{UnsplashQuery: "foggy landscape", BingIndex: 7}
	}
	if strings.Contains(normalized, "encoberto") || strings.Contains(normalized, "nublado") {
		return backgroundChoice{UnsplashQuery: "overcast sky", BingIndex: 3}
	}
	if strings.Contains(normalized, "poucas nuvens") || strings.Contains(normalized, "parcialmente") {
		return backgroundChoice{UnsplashQuery: "partly cloudy sky", BingIndex: 2}
	}
	if strings.Contains(normalized, "limpo") || strings.Contains(normalized, "clear") {
		return backgroundChoice{UnsplashQuery: "clear sky", BingIndex: 1}
	}

	return backgroundChoice{UnsplashQuery: defaultBackgroundQuery, BingIndex: 0}
}

func normalizeBackgroundLabel(label string) string {
	normalized := strings.TrimSpace(strings.ToLower(label))
	if normalized == "" {
		return ""
	}
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "â", "a", "ã", "a", "ä", "a",
		"é", "e", "ê", "e", "è", "e", "ë", "e",
		"í", "i", "î", "i", "ì", "i", "ï", "i",
		"ó", "o", "ô", "o", "õ", "o", "ò", "o", "ö", "o",
		"ú", "u", "û", "u", "ù", "u", "ü", "u",
		"ç", "c",
	)
	return replacer.Replace(normalized)
}
