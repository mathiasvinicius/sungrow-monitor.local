package weather

import (
    "context"
    "encoding/json"
    "fmt"
    "math"
    "net/http"
    "net/url"
    "strings"
    "time"
)

type OpenMeteoClient struct {
    city      string
    country   string
    latitude  float64
    longitude float64
    units     string
    client    *http.Client
}

func NewOpenMeteoClient(city, country string, latitude, longitude float64, units string) *OpenMeteoClient {
    if units == "" {
        units = "metric"
    }
    return &OpenMeteoClient{
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

type openMeteoResponse struct {
    Timezone string `json:"timezone"`
    Current  struct {
        Time          string  `json:"time"`
        WeatherCode   int     `json:"weather_code"`
        CloudCover    float64 `json:"cloud_cover"`
        Precipitation float64 `json:"precipitation"`
        Rain          float64 `json:"rain"`
        Showers       float64 `json:"showers"`
    } `json:"current"`
    Hourly struct {
        Time          []string  `json:"time"`
        Precipitation []float64 `json:"precipitation"`
        Rain          []float64 `json:"rain"`
        Showers       []float64 `json:"showers"`
    } `json:"hourly"`
    Daily struct {
        Sunrise []string `json:"sunrise"`
        Sunset  []string `json:"sunset"`
    } `json:"daily"`
}

type openMeteoGeoResponse struct {
    Results []struct {
        Latitude  float64 `json:"latitude"`
        Longitude float64 `json:"longitude"`
    } `json:"results"`
}

func (c *OpenMeteoClient) Get(ctx context.Context) (*Data, error) {
    lat, lon, err := c.resolveLocation(ctx)
    if err != nil {
        return nil, err
    }

    query := url.Values{}
    query.Set("latitude", fmt.Sprintf("%.6f", lat))
    query.Set("longitude", fmt.Sprintf("%.6f", lon))
    query.Set("current", "weather_code,cloud_cover,precipitation,rain,showers")
    query.Set("hourly", "precipitation,rain,showers")
    query.Set("daily", "sunrise,sunset")
    query.Set("timezone", "auto")
    query.Set("forecast_days", "1")
    query.Set("past_days", "1")
    query.Set("precipitation_unit", "mm")

    endpoint := url.URL{
        Scheme:   "https",
        Host:     "api.open-meteo.com",
        Path:     "/v1/forecast",
        RawQuery: query.Encode(),
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
    if err != nil {
        return nil, fmt.Errorf("open-meteo request: %w", err)
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return nil, fmt.Errorf("open-meteo request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return nil, fmt.Errorf("open-meteo bad status: %s", resp.Status)
    }

    var payload openMeteoResponse
    if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
        return nil, fmt.Errorf("open-meteo decode: %w", err)
    }

    if strings.TrimSpace(payload.Current.Time) == "" {
        return nil, fmt.Errorf("open-meteo current data missing")
    }

    observed, loc := parseOpenMeteoTime(payload.Current.Time, payload.Timezone)

    sunrise, sunset := pickOpenMeteoSunTimes(observed, payload.Timezone, payload.Daily.Sunrise, payload.Daily.Sunset)

    rain1h := payload.Current.Precipitation
    if rain1h == 0 {
        rain1h = payload.Current.Rain + payload.Current.Showers
    }

    rain3h := sumOpenMeteoPrecip(observed, payload.Hourly.Time, payload.Hourly.Precipitation, loc)
    if rain3h == 0 {
        rain3h = sumOpenMeteoPrecip(observed, payload.Hourly.Time, payload.Hourly.Rain, loc)
    }
    if rain3h == 0 {
        rain3h = sumOpenMeteoPrecip(observed, payload.Hourly.Time, payload.Hourly.Showers, loc)
    }

    condition, description := openMeteoDescribe(payload.Current.WeatherCode)

    return &Data{
        Provider:    "openmeteo",
        Condition:   condition,
        Description: description,
        Clouds:      int(math.Round(payload.Current.CloudCover)),
        Rain1h:      rain1h,
        Rain3h:      rain3h,
        Sunrise:     sunrise,
        Sunset:      sunset,
        ObservedAt:  observed,
    }, nil
}

func (c *OpenMeteoClient) resolveLocation(ctx context.Context) (float64, float64, error) {
    if c.latitude != 0 || c.longitude != 0 {
        return c.latitude, c.longitude, nil
    }

    if strings.TrimSpace(c.city) == "" {
        return 0, 0, fmt.Errorf("open-meteo location is empty")
    }

    query := url.Values{}
    query.Set("name", c.city)
    query.Set("count", "1")
    query.Set("language", "pt")
    query.Set("format", "json")
    if strings.TrimSpace(c.country) != "" {
        query.Set("country", c.country)
    }

    endpoint := url.URL{
        Scheme:   "https",
        Host:     "geocoding-api.open-meteo.com",
        Path:     "/v1/search",
        RawQuery: query.Encode(),
    }

    req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
    if err != nil {
        return 0, 0, fmt.Errorf("open-meteo geocoding request: %w", err)
    }

    resp, err := c.client.Do(req)
    if err != nil {
        return 0, 0, fmt.Errorf("open-meteo geocoding request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode < 200 || resp.StatusCode >= 300 {
        return 0, 0, fmt.Errorf("open-meteo geocoding bad status: %s", resp.Status)
    }

    var payload openMeteoGeoResponse
    if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
        return 0, 0, fmt.Errorf("open-meteo geocoding decode: %w", err)
    }

    if len(payload.Results) == 0 {
        return 0, 0, fmt.Errorf("open-meteo geocoding found no results")
    }

    c.latitude = payload.Results[0].Latitude
    c.longitude = payload.Results[0].Longitude

    return c.latitude, c.longitude, nil
}

func parseOpenMeteoTime(value, timezone string) (time.Time, *time.Location) {
    loc := time.UTC
    if strings.TrimSpace(timezone) != "" {
        if parsed, err := time.LoadLocation(timezone); err == nil {
            loc = parsed
        }
    }

    if t, err := time.ParseInLocation("2006-01-02T15:04", value, loc); err == nil {
        return t, loc
    }
    if t, err := time.ParseInLocation(time.RFC3339, value, loc); err == nil {
        return t, loc
    }
    return time.Time{}, loc
}

func pickOpenMeteoSunTimes(observed time.Time, timezone string, sunrises, sunsets []string) (time.Time, time.Time) {
    if len(sunrises) == 0 || len(sunsets) == 0 {
        return time.Time{}, time.Time{}
    }

    count := len(sunrises)
    if len(sunsets) < count {
        count = len(sunsets)
    }

    observedDate := time.Date(observed.Year(), observed.Month(), observed.Day(), 0, 0, 0, 0, observed.Location())
    closestDiff := time.Duration(1<<63 - 1)
    var closestSunrise time.Time
    var closestSunset time.Time

    for i := 0; i < count; i++ {
        sunrise, _ := parseOpenMeteoTime(sunrises[i], timezone)
        sunset, _ := parseOpenMeteoTime(sunsets[i], timezone)
        if sunrise.IsZero() || sunset.IsZero() {
            continue
        }
        if sameOpenMeteoDate(observed, sunrise) || sameOpenMeteoDate(observed, sunset) {
            return sunrise, sunset
        }

        sunriseDate := time.Date(sunrise.Year(), sunrise.Month(), sunrise.Day(), 0, 0, 0, 0, sunrise.Location())
        diff := observedDate.Sub(sunriseDate)
        if diff < 0 {
            diff = -diff
        }
        if diff < closestDiff {
            closestDiff = diff
            closestSunrise = sunrise
            closestSunset = sunset
        }
    }

    return closestSunrise, closestSunset
}

func sameOpenMeteoDate(a, b time.Time) bool {
    return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
}

func sumOpenMeteoPrecip(now time.Time, times []string, values []float64, loc *time.Location) float64 {
    if len(times) == 0 || len(times) != len(values) {
        return 0
    }
    if loc == nil {
        loc = time.UTC
    }

    windowStart := now.Add(-3 * time.Hour)
    sum := 0.0

    for i, value := range values {
        t, _ := time.ParseInLocation("2006-01-02T15:04", times[i], loc)
        if t.IsZero() {
            t, _ = time.ParseInLocation(time.RFC3339, times[i], loc)
        }
        if t.IsZero() {
            continue
        }
        if t.After(windowStart) && (t.Before(now) || t.Equal(now)) {
            sum += value
        }
    }

    return sum
}

func openMeteoDescribe(code int) (string, string) {
    switch code {
    case 0:
        return "Clear", "ceu limpo"
    case 1:
        return "Clouds", "principalmente limpo"
    case 2:
        return "Clouds", "poucas nuvens"
    case 3:
        return "Clouds", "ceu encoberto"
    case 45, 48:
        return "Fog", "nevoeiro"
    case 51, 53, 55, 56, 57:
        return "Drizzle", "garoa"
    case 61, 63, 65, 66, 67:
        return "Rain", "chuva"
    case 71, 73, 75, 77:
        return "Snow", "neve"
    case 80, 81, 82:
        return "Rain", "pancadas de chuva"
    case 85, 86:
        return "Snow", "pancadas de neve"
    case 95:
        return "Thunderstorm", "trovoada"
    case 96, 99:
        return "Thunderstorm", "trovoada com granizo"
    default:
        return "Unknown", "condicao desconhecida"
    }
}
