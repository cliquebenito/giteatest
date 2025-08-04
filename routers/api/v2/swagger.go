package v2

import (
	"time"
)

// swagger:response Hook
type swaggerResponseHook struct {
	// in:body
	Body Hook
}
type Hook struct {
	ID                  int64             `json:"id"`
	Type                string            `json:"type,omitempty"`
	URL                 string            `json:"-"`
	Config              map[string]string `json:"config"`
	Events              []string          `json:"events"`
	AuthorizationHeader string            `json:"authorization_header"`
	Active              bool              `json:"active"`
	// swagger:strfmt date-time
	Updated time.Time `json:"updated_at"`
	// swagger:strfmt date-time
	Created time.Time `json:"created_at"`
}
