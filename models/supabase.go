package model

import (
	"time"

	"github.com/google/uuid"
)

type Patient struct {
	Id                   uuid.UUID              `json:"id"`
	Name                 string                 `json:"name"`
	DateOfBirth          string                 `json:"date_of_birth"`
	Age                  int                    `json:"age"`
	ChiefConcern         ChiefConcern           `json:"chief_concern"`
	Gender               string                 `json:"gender"`
	MedicalCondition     string                 `json:"medical_condition"`
	MedicalHistory       string                 `json:"medical_history"`
	FamilyMedicalHistory string                 `json:"family_medical_history"`
	SurgicalHistory      string                 `json:"surgical_history"`
	Cholesterol          string                 `json:"cholesterol"`
	Allergies            string                 `json:"allergies"`
	PatientMessage       string                 `json:"patient_message"`
	PDMP                 []PDMPEntry            `json:"pdmp"`
	Immunization         map[string]interface{} `json:"immunization"`
	Height               string                 `json:"height"`
	Weight               string                 `json:"weight"`
	BP                   string                 `json:"last_bp"`
	LastVisitDate        string                 `json:"last_visit_date"`
}

type ChiefConcern struct {
	ChiefComplaint     []string `json:"chief_complaint"`
	ChiefComplaintTags []string `json:"chief_complaint_tags"`
}

type PDMPEntry struct {
	DateFilled  string `json:"date_filled"`
	DateWritten string `json:"date_written"`
	Drug        string `json:"drug"`
	Qty         int    `json:"qty"`
	Days        int    `json:"days"`
	Refill      int    `json:"refill"`
}

type EmbeddedPatient struct {
	Name string `json:"name"`
}

type Prescription struct {
	ID         uuid.UUID       `json:"id"`
	Patient_id uuid.UUID       `json:"patient_id"`
	Medication string          `json:"medication"`
	Dose       string          `json:"dose"`
	Patient    EmbeddedPatient `json:"patient"`
}

type Result struct {
	ID          uuid.UUID       `json:"id"`
	Patient_id  uuid.UUID       `json:"patient_id"`
	Test_name   string          `json:"test_name"`
	Test_date   string          `json:"test_date"`
	Test_result map[string]any  `json:"test_result"`
	Patient     EmbeddedPatient `json:"patient"`
}

type FlaggedPatient struct {
	ID        uuid.UUID         `json:"id"`
	PatientID uuid.UUID         `json:"patient_id"`
	Flaggers  []uuid.UUID       `json:"flaggers"`
	Messages  map[string]string `json:"messages"`
	Patient   struct {
		Name string `json:"name"`
	} `json:"patient"`
}

type Order struct {
	Name string    `json:"name"`
	Type string    `json:"type"`
	ID   uuid.UUID `json:"id"`
}

type OrderedOrders struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Date string `json:"date"`
}

type Question struct {
	QuestionID string     `json:"question_id,omitempty"`
	UserID     uuid.UUID  `json:"user_id"`
	Content    string     `json:"content"`
	Nickname   string     `json:"nickname"`
	CreatedAt  time.Time  `json:"created_at,omitempty"`
	Upvotes    int        `json:"upvotes"`
	Downvotes  int        `json:"downvotes"`
	Tags       []string   `json:"tags,omitempty"`
	PatientID  *uuid.UUID `json:"patient_id,omitempty"`
}

type Reply struct {
	QuestionID string    `json:"question_id"`
	ReplyID    string    `json:"reply_id"`
	UserID     uuid.UUID `json:"user_id"`
	EndorsedBy *string   `json:"endorsed_by,omitempty"`
	Nickname   string    `json:"nickname"`
	Content    string    `json:"content"`
	Upvotes    int       `json:"upvotes"`
	Downvotes  int       `json:"downvotes"`
}

type Vote struct {
	VoteID     string    `json:"vote_id"`
	UserID     uuid.UUID `json:"user_id"`
	TargetID   string    `json:"target_id,omitempty"`
	TargetType string    `json:"target_type"`
	IsUpvote   string    `json:"is_upvote"`
}

type Staff struct {
	Id             uuid.UUID `json:"id"`
	PatientID      uuid.UUID `json:"patient_id"`
	Name           string    `json:"name"`
	Role           string    `json:"role"`
	Specialty      string    `json:"specialty"`
	StaffMessage   string    `json:"staff_message"`
	ProfilePicture *string   `json:"profile_picture,omitempty"`
}
