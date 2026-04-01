package model

import (
	"time"

	"github.com/google/uuid"
)

type PatientVisit struct {
	ID            uuid.UUID  `json:"id"`
	PatientId     uuid.UUID  `json:"patient_id"`
	ProviderId    *uuid.UUID `json:"provider_id"`
	VisitDate     time.Time  `json:"visit_date"`
	VisitType     string     `json:"visit_type"`
	ClinicalNotes string     `json:"clinical_notes"`
	CreatedAt     time.Time  `json:"created_at"`
}

type PatientVisitWithProvider struct {
	PatientVisit
	ProviderName string `json:"provider_name"`
}
