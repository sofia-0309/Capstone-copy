package model

import (
	"time"

	"github.com/google/uuid"
)

// 😒 Bruh this doesnt look professional at all but ...
type Questions struct {
	Question string `json:"question"`
	A        string `json:"A"`
	B        string `json:"B"`
	C        string `json:"C"`
	D        string `json:"D"`
	Answer   string `json:"Answer"`
}

type DQuestions struct {
	QuestionId   int    `json:"question_id"`
	QuidId       int    `json:"quiz_id"`
	QuestionText string `json:"question_text"`
}

type Options struct {
	OptionId   int    `json:"option_id"`
	QuestionId int    `json:"question_id"`
	OptionText string `json:"option_text"`
	IsCorrect  bool   `json:"is_correct"`
}

type Records struct {
	RecordId int       `json:"record_id"`
	QuizId   int       `json:"quiz_id"`
	UserId   uuid.UUID `json:"user_id"`
	Score    int       `json:"score"`
	CreatedAt *time.Time `json:"created_at"`
}

type Quizzes struct {
	QuizId    int        `json:"quiz_id"`
	CreatedAt *time.Time `json:"created_at"`
}

type UserProgress struct {
	ProgressId int `json:"progress_id`
	UserId   uuid.UUID  `json:"user_id"`
	QuizId   int `json:"quiz_id"`
	QuestionId   int `json:"question_id"`
	SelectedOptionId   int `json:"selected_option_id"`

}

type FlaggedQuestion struct {
	UserId   uuid.UUID  `json:"user_id"`
	QuestionId   int `json:"question_id"`
	Message string `json:"message"`

}

type Conversation struct {
	ConversationId int `json:"conversation_id"`
	UserId   uuid.UUID  `json:"user_id"`
	QuizId   int `json:"quiz_id"`
}


type Message struct {
	MessageId int `json:"message_id"`
	ConversationId int `json:"conversation_id"`
	Content string `json:"content"`	
	Role string `json:"role"`
	IsShown  bool   `json:"is_shown"`


}