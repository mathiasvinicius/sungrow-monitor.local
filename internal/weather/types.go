package weather

import (
	"context"
	"time"
)

type Provider interface {
	Get(ctx context.Context) (*Data, error)
}

type Data struct {
	Provider    string    `json:"provider"`
	Condition   string    `json:"condition"`
	Description string    `json:"description"`
	Clouds      int       `json:"clouds"`
	Rain1h      float64   `json:"rain_1h,omitempty"`
	Rain3h      float64   `json:"rain_3h,omitempty"`
	Sunrise     time.Time `json:"sunrise"`
	Sunset      time.Time `json:"sunset"`
	ObservedAt  time.Time `json:"observed_at"`
}

func (d *Data) IsDaylight(at time.Time) bool {
	if d == nil || d.Sunrise.IsZero() || d.Sunset.IsZero() {
		return false
	}
	return at.After(d.Sunrise) && at.Before(d.Sunset)
}
