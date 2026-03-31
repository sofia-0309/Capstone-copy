package model

import (
	"github.com/google/uuid"
)

type User struct {
	Id              uuid.UUID   `json:"id"`
	UserUniqueId    string      `json:"userId"`
	Nickname        string      `json:"nickname"`
	Name            string      `json:"name"`
	IsAdmin         bool        `json:"isAdmin"`
	PasswordHash    string      `json:"passwordHash"`
	StudentStanding *string     `json:"studentStanding"`
	Students        []uuid.UUID `json:"students"`
	IsAssigned      bool        `json:"isAssigned"`
	TimeOnline      int         `json:"timeOnline"`
	//profile fields
	Icon             *string  `json:"icon,omitempty"`
	Rotation         *string  `json:"rotation,omitempty"`
	Mode             bool     `json:"isDarkMode,omitempty"`
	ImprovementAreas []string `json:"improvementAreas,omitempty"`
	Border           string   `json:"border,omitempty"`
	FeedbackType	 string	  `json:"feedback_type"`
}
