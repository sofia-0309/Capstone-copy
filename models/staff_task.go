package model

import (
	"time"

	"github.com/google/uuid"
)

type StaffTask struct {
	Id              *uuid.UUID `json:"id,omitempty"`
	CreatedAt       *time.Time `json:"created_at,omitempty"`
	StaffId         uuid.UUID  `json:"staff_id"`
	UserId          uuid.UUID  `json:"user_id"`
	Completed       bool       `json:"completed"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	StudentResponse *string    `json:"student_response,omitempty"`
	LLMResponse     *string    `json:"llm_response,omitempty"`
	LLMFeedback     *string    `json:"llm_feedback,omitempty"`
	StaffQuestion   *string    `json:"staff_question,omitempty"`
}
