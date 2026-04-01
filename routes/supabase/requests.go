package supabase

import (
	"time"

	"github.com/google/uuid"
)

type QuizCompleteReq struct {
	Score int `json:"score"`
}

// User Create Request Model
type UserCreateRequest struct {
	Name            string `json:"name"`
	Password        string `json:"password"`
	UserId          string `json:"userId"`
	Nickname        string `json:"nickname"`
	IsAdmin         bool   `json:"isAdmin"`
	StudentStanding string `json:"studentStanding"`
	PasswordHash    string `json:"passwordHash"`
	//profile fields
	Icon             string   `json:"icon,omitempty"`
	Rotation         string   `json:"rotation,omitempty"`
	Mode             bool     `json:"isDarkMode,omitempty"`
	ImprovementAreas []string `json:"improvementAreas,omitempty"`
	FeedbackType     string    `json:"feedback_type"`
}

// User Update Profile Request Model (they can only update these fields... tbh not sure if we need this)
type UpdateProfileRequest struct {
	Icon             string   `json:"icon,omitempty"`
	Rotation         string   `json:"rotation,omitempty"`
	Mode             string   `json:"mode,omitempty"`
	ImprovementAreas []string `json:"improvementAreas,omitempty"`
}

// User Login Request Model
type UserLoginRequest struct {
	ID       string `json:"id"` // matches frontend key
	Password string `json:"password"`
}

// Task Create Request Model
type TaskCreateRequest struct {
	PatientTaskCount      int  `json:"patient_task_count"`
	LabResultTaskCount    int  `json:"lab_result_task_count"`
	PrescriptionTaskCount int  `json:"prescription_task_count"`
	GenerateQuestion      bool `json:"generate_question"`
}

// Task Get Request Model
type TaskGetRequest struct {
	GetIncompleteTasks *bool  `json:"get_incomplete_tasks,omitempty"`
	GetCompleteTasks   *bool  `json:"get_complete_tasks,omitempty"`
	AgeFilter          string `json:"age_filter,omitempty"`
}

type TaskCompleteRequest struct {
	StudentResponse string `json:"student_response"`
	LLMResponse     string `json:"llm_response"`
	LLMFeedback     string `json:"llm_feedback"`
}

type FlaggedPatientRequest struct {
	PatientID   uuid.UUID `json:"patient_id"`
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"user_name"`
	Explanation string    `json:"explanation"`
}

type InsertFlaggedPatient struct {
	ID        uuid.UUID         `json:"id"`
	PatientID uuid.UUID         `json:"patient_id"`
	Flaggers  []uuid.UUID       `json:"flaggers"`
	Messages  map[string]string `json:"messages"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email"`
}

type UpdateUserInput struct {
	Email        string                 `json:"email,omitempty"`
	Password     string                 `json:"password,omitempty"`
	AppMetadata  map[string]interface{} `json:"app_metadata,omitempty"`
	UserMetadata map[string]interface{} `json:"user_metadata,omitempty"`
}

type ResetPasswordRequest struct {
	AccessToken string `json:"accessToken"`
	NewPassword string `json:"newPassword"`
}

type AddStudentRequest struct {
	InstructorId string `json:"instructor_id"`
	StudentId    string `json:"student_id"`
}

type UpdateFlaggedPatientByIDRequest struct {
	PatientID string `json:"patient_id"`
	Field     string `json:"selectedField"`
	OldKey    string `json:"oldKey,omitempty"`
	NewKey    string `json:"newKey,omitempty"`
	OldValue  string `json:"oldValue"`
	NewValue  string `json:"newValue"`
}

type InsertOrdersRequest struct {
	ID        uuid.UUID  `json:"id"`
	PatientID uuid.UUID  `json:"patient_id"`
	TaskID    uuid.UUID  `json:"task_id"`
	Name      string     `json:"name"`
	Date      *time.Time `json:"date"`
	Details   any        `json:"details,omitempty"`
}

// User update Request Model
type UserUpdateRequest struct {
	ID               uuid.UUID `json:"id"`
	Nickname         string    `json:"nickname,omitempty"`
	Icon             string    `json:"icon,omitempty"`
	Rotation         string    `json:"rotation,omitempty"`
	Mode             string    `json:"isDarkMode,omitempty"`
	ImprovementAreas []string  `json:"improvementAreas"`
	Border           string    `json:"border,omitempty"`
}

type InsertQuestionRequest struct {
	QuestionID string     `json:"question_id"`
	UserID     uuid.UUID  `json:"user_id"`
	Nickname   string     `json:"nickname"`
	Content    string     `json:"content"`
	CreatedAt  *time.Time `json:"created_at"`
	Tags       []string   `json:"tags,omitempty"`
	PatientID  *uuid.UUID `json:"patient_id,omitempty"`
}

type InsertReplyRequest struct {
	ReplyID    string     `json:"reply_id"`
	QuestionID string     `json:"question_id"`
	UserID     uuid.UUID  `json:"user_id"`
	Nickname   string     `json:"nickname"`
	Content    string     `json:"content"`
	CreatedAt  *time.Time `json:"created_at"`
}
type InsertVoteRequest struct {
	VoteID     uuid.UUID `json:"vote_id"`
	UserID     string    `json:"user_id"`
	TargetID   string    `json:"target_id"`
	TargetType string    `json:"target_type"`
	IsUpvote   bool      `json:"is_upvote"`
}

// type GetPatientFieldByIDRequest struct {
// 	PatientID string `json:"patient_id"`
// 	Field     string `json:"field"`
// }
//CHECK DATS TYPES

type PatientCreateRequest struct {

	//ID                   uuid.UUID `json:"id"`
	Name                 string `json:"name"`
	DateOfBirth          string `json:"date_of_birth"`
	Genre                string `json:"gender"`
	Height               string `json:"height"`
	Weight               string `json:"weight"`
	BloodPressure        string `json:"last_bp"`
	MedicalHistory       string `json:"medical_history"`
	Symptoms             string `json:"medical_condition"`
	FamilyMedicalHistory string `json:"family_medical_history"`
}

// Get named
type PatientNames struct {
	Name string `json:"name"`
}

//Get Demograohics

type Demographics struct {
	DateOfBirth string `json:"date_of_birth"`
	Gender      string `json:"gender"`
}

//Get vitals

type Vitals struct {
	Height  string `json:"height"`
	Weight  string `json:"weight"`
	Last_bp string `json:"last_bp"`
}
type MedH struct {
	MedicalHistory string `json:"medical_history"`
}
type FMH struct {
	FamilyMedicalHistory string `json:"family_medical_history"`
}

type Modify struct {
	PatientID        string `json:"id"`
	MedicalHistory   string `json:"medical_history"`
	MedicalCondition string `json:"medical_condition"`
	Message          string `json:"patient_message"`
}

type PubMedRequest struct {
	Query string `json:"query"`
}

type PubMedArticle struct {
	PMID    string `json:"pmid"`
	Title   string `json:"title"`
	Journal string `json:"journal"`
	PubDate string `json:"pubDate"`
	Link    string `json:"link"`
}
