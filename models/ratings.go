package model

import (
	"time"

	"github.com/google/uuid"
)

// Rating represents a student's rating of AI feedback
type Rating struct {
	Id        uuid.UUID  `json:"id"`
	UserId    uuid.UUID  `json:"user_id"`
	TaskId    uuid.UUID  `json:"task_id"`
	PatientId *uuid.UUID `json:"patient_id,omitempty"`
	TaskType  string     `json:"task_type"`
	Rating    int        `json:"rating"`
	CreatedAt time.Time  `json:"created_at"`
}

// RatingStats holds aggregated rating statistics for a user
type RatingStats struct {
	TotalRatings       int                       `json:"totalRatings"`
	AverageRating      float64                   `json:"averageRating"`
	RatingDistribution [5]int                    `json:"ratingDistribution"` // counts for 1-5 stars
	RatingsByTaskType  map[string]TaskTypeRating `json:"ratingsByTaskType"`
	RecentRatings      []RecentRating            `json:"recentRatings"`
}

// TaskTypeRating holds stats for a specific task type
type TaskTypeRating struct {
	Average float64 `json:"average"`
	Count   int     `json:"count"`
}

// RecentRating represents a single rating with patient info
type RecentRating struct {
	TaskType    string `json:"taskType"`
	Rating      int    `json:"rating"`
	Timestamp   string `json:"timestamp"`
	PatientName string `json:"patientName"`
}
