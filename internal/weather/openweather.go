package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

type OpenWeatherClient struct {
	apiKey    string
	city      string
	country   string
	latitude  float64
	longitude float64
	units     string
	client    *http.Client
}

func NewOpenWeatherClient(apiKey, city, country string, latitude, longitude float64, units string) *OpenWeatherClient {
	if units == "" {
		units = "metric"
	}
	return &OpenWeatherClient{
		apiKey:    apiKey,
		city:      city,
		country:   country,
		latitude:  latitude,
		longitude: longitude,
		units:     units,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

type openWeatherResponse struct {
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
	Clouds struct {
		All int `json:"all"`
	} `json:"clouds"`
	Rain struct {
		OneHour  float64 `json:"1h"`
		ThreeHr  float64 `json:"3h"`
	} `json:"rain"`
	Dt       int64 `json:"dt"`
	Timezone int64 `json:"timezone"`
	Sys      struct {
		Sunrise int64 `json:"sunrise"`
		Sunset  int64 `json:"sunset"`
	} `json:"sys"`
}

func (c *OpenWeatherClient) Get(ctx context.Context) (*Data, error) {
	if c.apiKey == "" {
		return nil, fmt.Errorf("openweather api key is empty")
	}

	query := url.Values{}
	query.Set("appid", c.apiKey)
	query.Set("units", c.units)
	query.Set("lang", "pt_br")

	if c.latitude != 0 || c.longitude != 0 {
		query.Set("lat", fmt.Sprintf("%.6f", c.latitude))
		query.Set("lon", fmt.Sprintf("%.6f", c.longitude))
	} else if c.city != "" {
		if c.country != "" {
			query.Set("q", fmt.Sprintf("%s,%s", c.city, c.country))
		} else {
			query.Set("q", c.city)
		}
	} else {
		return nil, fmt.Errorf("openweather location is empty")
	}

	endpoint := url.URL{
		Scheme:   "https",
		Host:     "api.openweathermap.org",
		Path:     "/data/2.5/weather",
		RawQuery: query.Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("openweather request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("openweather request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("openweather bad status: %s", resp.Status)
	}

	var payload openWeatherResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("openweather decode: %w", err)
	}

	condition := ""
	description := ""
	if len(payload.Weather) > 0 {
		condition = payload.Weather[0].Main
		description = payload.Weather[0].Description
	}

	offset := time.Duration(payload.Timezone) * time.Second
	observed := time.Unix(payload.Dt, 0).UTC().Add(offset)
	sunrise := time.Unix(payload.Sys.Sunrise, 0).UTC().Add(offset)
	sunset := time.Unix(payload.Sys.Sunset, 0).UTC().Add(offset)

	return &Data{
		Provider:    "openweather",
		Condition:   condition,
		Description: description,
		Clouds:      payload.Clouds.All,
		Rain1h:      payload.Rain.OneHour,
		Rain3h:      payload.Rain.ThreeHr,
		Sunrise:     sunrise,
		Sunset:      sunset,
		ObservedAt:  observed,
	}, nil
}
