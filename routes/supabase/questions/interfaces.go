package questions
import (
	"net/http"

)
type QuestionService interface {
	GetQuestionsForQuiz(w http.ResponseWriter, r *http.Request)
	CompleteQuestion(w http.ResponseWriter, r *http.Request)
	GetQuestionProgress(w http.ResponseWriter, r *http.Request)
	GetQuestionStats(w http.ResponseWriter, r *http.Request)
	FlaggedQuestionRequest(w http.ResponseWriter, r *http.Request)
	GetFlaggedQuestions(w http.ResponseWriter, r *http.Request)
	FlaggedQuestionResolve(w http.ResponseWriter, r *http.Request)



}

