package llm

type MessageRequest struct {
	Message string `json:"message"`
}

type PatientConcernsRequest struct {
	Age int `json:"age"`
	Gender string `json:"gender"`
}