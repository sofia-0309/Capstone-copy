package questionboard

import "net/http"

type QuestionBoardService interface {
	PostQuestion(w http.ResponseWriter, r *http.Request)
	GetQuestions(w http.ResponseWriter, r *http.Request)
	GetQuestionTags(w http.ResponseWriter, r *http.Request)
	GetReplies(w http.ResponseWriter, r *http.Request)
	PostReply(w http.ResponseWriter, r *http.Request)
	DeleteReply(w http.ResponseWriter, r *http.Request)
	DeleteQuestion(w http.ResponseWriter, r *http.Request)
	EndorseReply(w http.ResponseWriter, r *http.Request)
	AddVote(w http.ResponseWriter, r *http.Request)
}
