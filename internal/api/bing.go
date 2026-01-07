package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

const bingWallpaperTTL = 6 * time.Hour

var bingMarketPattern = regexp.MustCompile(`^[a-z]{2}-[A-Z]{2}$`)

type bingWallpaperPayload struct {
	Provider  string `json:"provider"`
	Market    string `json:"mkt"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	Copyright string `json:"copyright"`
}

type bingWallpaperResponse struct {
	Images []bingWallpaperImage `json:"images"`
}

type bingWallpaperImage struct {
	URL       string `json:"url"`
	Title     string `json:"title"`
	Copyright string `json:"copyright"`
}

type bingWallpaperCacheEntry struct {
	FetchedAt time.Time
	Payload   bingWallpaperPayload
}

var (
	bingWallpaperCacheMu sync.Mutex
	bingWallpaperCache   = map[string]bingWallpaperCacheEntry{}
)

func (s *Server) bingWallpaperHandler(c *gin.Context) {
	market := sanitizeBingMarket(c.Query("mkt"))
	payload, err := getBingWallpaper(c.Request.Context(), market)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"error":   "Failed to fetch Bing wallpaper",
			"details": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, payload)
}

func sanitizeBingMarket(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "pt-BR"
	}
	if bingMarketPattern.MatchString(trimmed) {
		return trimmed
	}
	return "pt-BR"
}

func getBingWallpaper(ctx context.Context, market string) (bingWallpaperPayload, error) {
	now := time.Now()
	bingWallpaperCacheMu.Lock()
	entry, ok := bingWallpaperCache[market]
	bingWallpaperCacheMu.Unlock()
	if ok && now.Sub(entry.FetchedAt) < bingWallpaperTTL {
		return entry.Payload, nil
	}

	payload, err := fetchBingWallpaper(ctx, market)
	if err != nil {
		if ok {
			return entry.Payload, nil
		}
		return bingWallpaperPayload{}, err
	}

	bingWallpaperCacheMu.Lock()
	bingWallpaperCache[market] = bingWallpaperCacheEntry{
		FetchedAt: now,
		Payload:   payload,
	}
	bingWallpaperCacheMu.Unlock()
	return payload, nil
}

func fetchBingWallpaper(ctx context.Context, market string) (bingWallpaperPayload, error) {
	endpoint := fmt.Sprintf(
		"https://www.bing.com/HPImageArchive.aspx?format=js&idx=0&n=1&mkt=%s",
		market,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return bingWallpaperPayload{}, fmt.Errorf("bing request: %w", err)
	}
	req.Header.Set("User-Agent", "SungrowMonitor/1.0 (+https://github.com/mathiasvinicius/sungrow-monitor.local)")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return bingWallpaperPayload{}, fmt.Errorf("bing request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return bingWallpaperPayload{}, fmt.Errorf("bing bad status: %s", resp.Status)
	}

	var payload bingWallpaperResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return bingWallpaperPayload{}, fmt.Errorf("bing decode: %w", err)
	}
	if len(payload.Images) == 0 || strings.TrimSpace(payload.Images[0].URL) == "" {
		return bingWallpaperPayload{}, fmt.Errorf("bing image URL is missing")
	}

	image := payload.Images[0]
	url := strings.TrimSpace(image.URL)
	if !strings.HasPrefix(url, "http") {
		url = "https://www.bing.com" + url
	}

	return bingWallpaperPayload{
		Provider:  "bing",
		Market:    market,
		URL:       url,
		Title:     strings.TrimSpace(image.Title),
		Copyright: strings.TrimSpace(image.Copyright),
	}, nil
}
