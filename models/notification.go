package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Notification struct {
	Id        uuid.UUID      `json:"id"`
	UserId    uuid.UUID      `json:"userId"`
	Type      string         `json:"type"` //"instructor_assignment", "instructor_removed", "general", etc.
	Title     string         `json:"title"`
	Message   string         `json:"message"`
	IsRead    bool           `json:"isRead"`
	CreatedAt FlexibleTime   `json:"createdAt"`
	ActionUrl *string        `json:"actionUrl,omitempty"` // Optional URL for action button
}

// FlexibleTime is a custom time type that can parse timestamps with or without timezone
type FlexibleTime struct {
	time.Time
}

// UnmarshalJSON implements custom JSON unmarshaling for FlexibleTime
func (ft *FlexibleTime) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	// Try multiple timestamp formats
	formats := []string{
		time.RFC3339,                  // "2006-01-02T15:04:05Z07:00"
		time.RFC3339Nano,              // "2006-01-02T15:04:05.999999999Z07:00"
		"2006-01-02T15:04:05.999999",  // Without timezone
		"2006-01-02T15:04:05",         // Without timezone and microseconds
		"2006-01-02 15:04:05.999999",  // Space separator without timezone
		"2006-01-02 15:04:05",         // Space separator without timezone and microseconds
	}

	var parseErr error
	for _, format := range formats {
		t, err := time.Parse(format, s)
		if err == nil {
			ft.Time = t
			return nil
		}
		parseErr = err
	}

	return parseErr
}

// MarshalJSON implements custom JSON marshaling for FlexibleTime
func (ft FlexibleTime) MarshalJSON() ([]byte, error) {
	return json.Marshal(ft.Time)
}
