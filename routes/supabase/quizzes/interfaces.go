package quizzes
import (
	"net/http"

)


type QuizService interface {
	GenerateQuestions(topics []string) error
	GetQuizForIndividuals(w http.ResponseWriter, r *http.Request)
	CompleteQuiz(w http.ResponseWriter, r *http.Request)
	GiveInitialQuiz(w http.ResponseWriter, r *http.Request)

}



