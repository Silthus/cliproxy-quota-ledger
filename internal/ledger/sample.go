package ledger

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"
	"unicode/utf8"
)

const maxFutureSkew = 10 * time.Minute

type Sample struct {
	At        int64   `json:"at"`
	Auth      string  `json:"auth"`
	Provider  string  `json:"provider"`
	Window    string  `json:"window"`
	Remaining float64 `json:"remaining"`
	ResetsAt  *int64  `json:"resets_at"`
}

type series struct {
	auth, provider, window string
}

func (s Sample) series() series {
	return series{auth: s.Auth, provider: s.Provider, window: s.Window}
}

func (s *Sample) UnmarshalJSON(data []byte) error {
	type fields Sample
	var decoded struct {
		fields
		Remaining *float64 `json:"remaining"`
	}
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	if decoded.Remaining == nil {
		return errors.New("remaining is required")
	}
	*s = Sample(decoded.fields)
	s.Remaining = *decoded.Remaining
	return nil
}

func (s Sample) Validate(now time.Time) error {
	if s.At <= 0 {
		return errors.New("at must be a positive unix ms timestamp")
	}
	if s.At > now.Add(maxFutureSkew).UnixMilli() {
		return errors.New("at is more than 10 minutes in the future")
	}
	if err := checkLength("auth", s.Auth, 256); err != nil {
		return err
	}
	if err := checkLength("provider", s.Provider, 64); err != nil {
		return err
	}
	if err := checkLength("window", s.Window, 128); err != nil {
		return err
	}
	if math.IsNaN(s.Remaining) || s.Remaining < 0 || s.Remaining > 100 {
		return errors.New("remaining must be a percentage between 0 and 100")
	}
	if s.ResetsAt != nil && *s.ResetsAt <= 0 {
		return errors.New("resets_at must be a positive unix ms timestamp or null")
	}
	return nil
}

func checkLength(field, value string, maxChars int) error {
	if count := utf8.RuneCountInString(value); count < 1 || count > maxChars {
		return fmt.Errorf("%s must be 1..%d characters", field, maxChars)
	}
	return nil
}
