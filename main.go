package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/robfig/cron/v3"
	"github.com/rs/cors"

	"gitlab.msu.edu/team-corewell-2025/routes/llm"
	reports "gitlab.msu.edu/team-corewell-2025/routes/reports"
	"gitlab.msu.edu/team-corewell-2025/routes/supabase"
	chats "gitlab.msu.edu/team-corewell-2025/routes/supabase/chats"
	conversations "gitlab.msu.edu/team-corewell-2025/routes/supabase/conversations"
	orders "gitlab.msu.edu/team-corewell-2025/routes/supabase/orders"
	ordersList "gitlab.msu.edu/team-corewell-2025/routes/supabase/ordersList"
	patients "gitlab.msu.edu/team-corewell-2025/routes/supabase/patients"
	prescriptions "gitlab.msu.edu/team-corewell-2025/routes/supabase/prescriptions"
	"gitlab.msu.edu/team-corewell-2025/routes/supabase/profiles"
	questionboard "gitlab.msu.edu/team-corewell-2025/routes/supabase/questionboard"
	questions "gitlab.msu.edu/team-corewell-2025/routes/supabase/questions"
	quizzes "gitlab.msu.edu/team-corewell-2025/routes/supabase/quizzes"
	results "gitlab.msu.edu/team-corewell-2025/routes/supabase/results"
	staff "gitlab.msu.edu/team-corewell-2025/routes/supabase/staff"
	staffTasks "gitlab.msu.edu/team-corewell-2025/routes/supabase/staff_tasks"
	tasks "gitlab.msu.edu/team-corewell-2025/routes/supabase/tasks"
	tickets "gitlab.msu.edu/team-corewell-2025/routes/supabase/tickets"
)

func main() {

	// Load env vars
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println("Error loading in .env")
	}

	// Create client
	url := os.Getenv("SUPABASE_URL")
	key := os.Getenv("SUPABASE_SERVICE_KEY")
	if key == "" {
		// Backward-compatible fallback while environments are migrated.
		key = os.Getenv("SUPABASE_KEY")
	}
	supa := supabase.InitClient(url, key)

	// Dependency Injection
	var ph patients.PatientService = &patients.PatientHandler{Supabase: supa}
	var prh prescriptions.PrescriptionService = &prescriptions.PrescriptionHandler{Supabase: supa}
	var ord orders.OrdersService = &orders.OrdersHandler{Supabase: supa}
	var ordl ordersList.OrdersListService = &ordersList.OrdersListHandler{Supabase: supa}
	var rh results.ResultService = &results.ResultHandler{Supabase: supa}
	var sh staff.StaffService = &staff.StaffHandler{Supabase: supa}
	var sth staffTasks.StaffTaskService = &staffTasks.StaffTaskHandler{Supabase: supa}
	var th tasks.TaskService = &tasks.TaskHandler{Supabase: supa}
	var proh profiles.ProfilesService = &profiles.ProfilesHandler{Supabase: supa}
	var qh quizzes.QuizService = &quizzes.QuizHandler{Supabase: supa}
	var qu questions.QuestionService = &questions.QuestionHandler{Supabase: supa}
	var ti tickets.TicketService = &tickets.TicketHandler{Supabase: supa}
	var qbo questionboard.QuestionBoardService = &questionboard.QuestionBoardHandler{Supabase: supa}
	var conv conversations.ConversationService = &conversations.ConversationHandler{Supabase: supa}
	var chat chats.ChatService = &chats.ChatHandler{Supabase: supa}
	var rep reports.ReportService = &reports.ReportHandler{}

	// // API Gateway
	m := mux.NewRouter()

	topics := []string{
		"General Principles, Including Normal Age-Related Findings and Care of the Well Patient",
		"Immune System",
		"Blood & Lymphoreticular System",
		"Behavioral Health",
		"Nervous System & Special Senses",
		"Skin & Subcutaneous Tissue",
		"Musculoskeletal System",
		"Cardiovascular System",
		"Respiratory System",
		"Gastrointestinal System",
		"Renal & Urinary System",
		"Pregnancy, Childbirth, & the Puerperium",
		"Female Reproductive System & Breast",
		"Male Reproductive System",
		"Endocrine System",
		"Multisystem Processes & Disorders",
		"Biostatistics, Epidemiology/Population Health, & Interpretation of the Medical Literature",
		"Social Sciences - Communication and interpersonal skills",
		"Social Sciences - Medical ethics and jurisprudence",
		"Social Sciences - Systems-based practice and patient safety",
	}
	// err = qh.GenerateQuestions(topics)
	// 	if err != nil {
	// 		fmt.Println("Error from Flask:", err)
	// 	} else {
	// 		fmt.Println("Successfully called Flask and received response.")
	// 	}

	c := cron.New(cron.WithSeconds())
	_, err = c.AddFunc("@daily", func() {
		fmt.Println("Job Scheduler Triggered. ")
		fmt.Println("Give questions out")
		err = qh.GenerateQuestions(topics)
		if err != nil {
			fmt.Println("Error from Flask:", err)
		} else {
			fmt.Println("Successfully called Flask and received response.")
		}
	})

	if err != nil {
		fmt.Println("Error scheduling cron job:", err)
		return
	}
	c.Start()

	// Auth  aaaa
	m.HandleFunc("/addUser", supabase.SignUpUser).Methods("POST")
	m.HandleFunc("/login", supabase.SignInUser).Methods("POST")
	m.HandleFunc("/forgotPassword", supabase.ForgotPassword).Methods("POST")
	m.HandleFunc("/resetPassword", supabase.ResetPassword).Methods("POST")

	//Profile
	m.HandleFunc("/profiles/GetTagsStats/{user_id}",proh.GetTagsStats).Methods("GET")
	m.HandleFunc("/profiles/GetTags/{user_id}",proh.GetLeastTags).Methods("GET")
	m.HandleFunc("/profiles/GetAllRatings",proh.GetAllRatings).Methods("GET")
	m.HandleFunc("/profiles/{id}", proh.GetProfile).Methods("GET")
	m.HandleFunc("/profiles/update", proh.UpdateProfile).Methods("POST")
	m.HandleFunc("/profiles/heartbeat", proh.UpdateLastActive).Methods("POST")

	m.HandleFunc("/profiles/updateFeedback", proh.UpdateFeedback).Methods("PUT")
	m.HandleFunc("/profiles/GetFeedback/{id}", proh.GetFeedback).Methods("GET")

	m.HandleFunc("/profiles/updateFeedback",proh.UpdateFeedback).Methods("PUT")
	m.HandleFunc("/profiles/GetFeedback/{id}",proh.GetFeedback).Methods("GET")
	m.HandleFunc("/profiles/AddRating",proh.SaveRatings).Methods("POST")
	
	
	m.HandleFunc("/profiles/GetRatings/{user_id}",proh.GetRatings).Methods("GET")
	m.HandleFunc("/profiles/GetRatings/{user_id}/{type}",proh.GetRatingsByType).Methods("GET")
	

	
	


	// Patients form
	//m.HandleFunc("/patients/getPatients",ph.GetPatients).Methods("GET")
	m.HandleFunc("/patients/Update/{id}", ph.UpdateData).Methods("PUT")
	m.HandleFunc("/patients/getDemo", ph.GetDemo).Methods("GET")
	m.HandleFunc("/patients/getVitals", ph.GetVitals).Methods("GET")
	m.HandleFunc("/patients/getMedH", ph.GetMedH).Methods("GET")
	m.HandleFunc("/patients/getFMH", ph.GetFMH).Methods("GET")
	m.HandleFunc("/patients/GetPatientsData", ph.GetPatientsData).Methods("GET")

	//Patients
	patientsRouter := m.PathPrefix("/patients").Subrouter()
	//patientsRouter.HandleFunc("/GetPatientsData", ph.GetPatientsData).Methods("GET")
	patientsRouter.HandleFunc("", ph.GetPatients).Methods("GET")
	patientsRouter.HandleFunc("/{id}", ph.GetPatientByID).Methods("GET")
	patientsRouter.HandleFunc("/{id}/prescriptions", prh.GetPrescriptionsByPatientID).Methods("GET")
	patientsRouter.HandleFunc("/{id}/results", rh.GetResultsByPatientID).Methods("GET")
	patientsRouter.HandleFunc("/{id}/llm-response", llm.PostLLMResponseForPatient).Methods("POST")
	patientsRouter.HandleFunc("/getBulkMessages", ph.GetMultiplePatientsByID).Methods("POST")
	patientsRouter.HandleFunc("/{id}/dermnet_image", llm.GetPatientImage).Methods("GET")
	patientsRouter.HandleFunc("/{id}/profile_picture", llm.GetPatientProfilePicture).Methods("GET")

	// Adding featured patients
	m.HandleFunc("/patients/addPatient", ph.AddNewPatient).Methods("POST")

	// Staff
	m.HandleFunc("/staff", sh.GetAllStaff).Methods("GET", "OPTIONS")

	// Flagging FeaturE
	// LLM
	m.HandleFunc("/api/patient-concerns", llm.PostPatientConcerns).Methods("POST")
	m.HandleFunc("/pubmed", supabase.GetPubMedArticles).Methods("POST")

	// Flagging Feature
	flaggedPatientsRouter := m.PathPrefix("/flagged").Subrouter()
	flaggedPatientsRouter.HandleFunc("/flaggedPatients", ph.GetFlaggedPatients).Methods("GET")
	flaggedPatientsRouter.HandleFunc("/addFlag", ph.AddFlaggedPatient).Methods("POST")
	flaggedPatientsRouter.HandleFunc("/{id}/updateFlaggedPatient", ph.UpdateFlaggedPatientByID).Methods("POST")

	// Prescriptions
	prescriptionsRouter := m.PathPrefix("/prescriptions").Subrouter()
	prescriptionsRouter.HandleFunc("", prh.GetPrescriptions).Methods("GET")
	prescriptionsRouter.HandleFunc("/{id}", prh.GetPrescriptionByID).Methods("GET")
	prescriptionsRouter.HandleFunc("/getBulkPrescriptions", prh.GetMultiplePrescriptionsByID).Methods("POST")

	// Orders
	ordersRouter := m.PathPrefix("/orders").Subrouter()
	ordersRouter.HandleFunc("/get", ord.GetOrders).Methods("GET")
	ordersRouter.HandleFunc("/getOrdered", ord.GetOrderedOrders).Methods("GET")
	ordersRouter.HandleFunc("/{task_id}", ord.LogOrder).Methods("POST")

	// Orders_List
	ordersListRouter := m.PathPrefix("/orders_list").Subrouter()
	ordersListRouter.HandleFunc("/get", ordl.GetOrdersList).Methods("GET")
	ordersListRouter.HandleFunc("/{id}/update", ordl.UpdateOrder).Methods("POST")
	ordersListRouter.HandleFunc("/add", ordl.AddOrder).Methods("POST")
	ordersListRouter.HandleFunc("/{id}/delete", ordl.DeleteOrder).Methods("DELETE")

	// Progress Report
	m.HandleFunc("/reports/progress", rep.GenerateProgressReport).Methods("POST")

	// Leaderboard
	m.HandleFunc("/students/leaderboard", supabase.GetLeaderboard).Methods("GET")

	// Question Board
	questionBoardRouter := m.PathPrefix("/questionboard").Subrouter()
	questionBoardRouter.HandleFunc("/post", qbo.PostQuestion).Methods("POST")
	questionBoardRouter.HandleFunc("/get", qbo.GetQuestions).Methods("GET")
	questionBoardRouter.HandleFunc("/tags", qbo.GetQuestionTags).Methods("GET")
	questionBoardRouter.HandleFunc("/replies/post", qbo.PostReply).Methods("POST")
	questionBoardRouter.HandleFunc("/replies/{question_id}", qbo.GetReplies).Methods("GET")
	questionBoardRouter.HandleFunc("/replies/{reply_id}/delete", qbo.DeleteReply).Methods("DELETE")
	questionBoardRouter.HandleFunc("/{question_id}/delete", qbo.DeleteQuestion).Methods("DELETE")
	questionBoardRouter.HandleFunc("/replies/{reply_id}/endorse", qbo.EndorseReply).Methods("POST")
	questionBoardRouter.HandleFunc("/vote", qbo.AddVote).Methods("POST")

	// Achievements
	m.HandleFunc("/getAchievements/{user_id}", supabase.GetAchievements).Methods("GET")
	m.HandleFunc("/updateAchievements", supabase.UpdateAchievements).Methods("POST")

	studentsRouter := m.PathPrefix("/students").Subrouter()
	studentsRouter.HandleFunc("", supabase.GetStudents).Methods("GET")
	studentsRouter.HandleFunc("/{id}", supabase.GetStudentById).Methods("GET")
	studentsRouter.HandleFunc("/{id}/analytics", supabase.GetStudentAnalytics).Methods("GET")

	// Results
	resultsRouter := m.PathPrefix("/results").Subrouter()
	resultsRouter.HandleFunc("", rh.GetResults).Methods("GET")
	resultsRouter.HandleFunc("/{id}", rh.GetResultByID).Methods("GET")
	resultsRouter.HandleFunc("/getBulkResults", rh.GetMultipleResultsByID).Methods("POST")

	// Endpoints for tasks (generating, getting, completing, etc.)
	m.HandleFunc("/generateTasks", th.GenerateTasksHTMLWrapper).Methods("POST")
	m.HandleFunc("/{student_id}/tasks", th.GetTasksByStudentID).Methods("POST", "OPTIONS") //had to make this post bc the function expects a body
	m.HandleFunc("/students/{student_id}/generate-initial-tasks", th.GenerateInitialTasksHandler).Methods("POST")
	m.HandleFunc("/generateNewTasks", th.GenerateNewTasks).Methods("POST")
	//hardcoded body to show incomplete tasks ^^^
	m.HandleFunc("/{student_id}/tasks/week", th.GetTasksByWeekAndDay).Methods("GET")
	m.HandleFunc("/{student_id}/tasks/{task_id}", th.GetTaskByID).Methods("GET")
	m.HandleFunc("/{student_id}/tasks/{task_id}/completeTask", th.CompleteTask).Methods("POST")
	m.HandleFunc("/{student_id}/calendar", th.GetTaskCalendar).Methods("GET")

	// Endpoints for staff tasks
	m.HandleFunc("/{student_id}/staff-tasks", sth.GetStaffTasksByStudentID).Methods("POST", "OPTIONS")
	m.HandleFunc("/{student_id}/staff-tasks/{task_id}", sth.GetStaffTaskByID).Methods("GET", "OPTIONS")
	m.HandleFunc("/{student_id}/staff-tasks/{task_id}/completeTask", sth.CompleteStaffTask).Methods("POST", "OPTIONS")
	m.HandleFunc("/students/{student_id}/generate-initial-staff-tasks", sth.GenerateInitialStaffTasksHandler).Methods("POST")

	//Student Instructor Assignment Feature
	m.HandleFunc("/instructors", supabase.GetInstructors).Methods("GET")
	m.HandleFunc("/instructors/{id}/students", supabase.GetInstructorStudents).Methods("GET")
	m.HandleFunc("/instructors/{id}/analytics", supabase.GetAnalytics).Methods("GET")
	m.HandleFunc("/addStudent", supabase.AddStudentToInstructor).Methods("POST")
	m.HandleFunc("/instructors/reassignStudent", supabase.ReassignStudent).Methods("POST")
	m.HandleFunc("/instructors/{instructor_id}/students/{student_id}", supabase.RemoveStudentFromInstructor).Methods("DELETE")
	m.HandleFunc("/students/{student_id}/instructor", supabase.GetStudentInstructor).Methods("GET")

	// Notifications
	m.HandleFunc("/notifications/{user_id}", supabase.GetUserNotifications).Methods("GET")
	m.HandleFunc("/notifications/{user_id}/unread-count", supabase.GetUnreadNotificationCount).Methods("GET")
	m.HandleFunc("/notifications/{notification_id}/read", supabase.MarkNotificationAsRead).Methods("POST")
	m.HandleFunc("/notifications/{user_id}/read-all", supabase.MarkAllNotificationsAsRead).Methods("POST")
	m.HandleFunc("/notifications", supabase.CreateNotification).Methods("POST")

	// quiz
	m.HandleFunc("/distribution/{question_id}", qu.GetQuestionStats).Methods("GET", "OPTIONS")
	m.HandleFunc("/quiz/{quiz_id}/{student_id}", qu.GetQuestionsForQuiz).Methods("GET", "OPTIONS")
	m.HandleFunc("/flagged/{question_id}", qu.FlaggedQuestionRequest).Methods("POST", "OPTIONS")
	m.HandleFunc("/flagged/flaggedQuestions", qu.GetFlaggedQuestions).Methods("GET", "OPTIONS")
	m.HandleFunc("/updateflaggedquestion/{question_id}", qu.FlaggedQuestionResolve).Methods("POST", "OPTIONS")
	m.HandleFunc("/quizzes/{quiz_id}/{student_id}/{question_id}/{selected_option_id}", qu.CompleteQuestion).Methods("POST", "OPTIONS")
	m.HandleFunc("/quiz/{student_id}", qh.GetQuizForIndividuals).Methods("GET", "OPTIONS")
	m.HandleFunc("/quizzes/{quiz_id}/{student_id}", qh.CompleteQuiz).Methods("POST", "OPTIONS")
	m.HandleFunc("/generate-initial-quiz/{student_id}", qh.GiveInitialQuiz).Methods("POST")

	// ticket
	m.HandleFunc("/get-tickets", ti.GetTickets).Methods("GET", "OPTIONS")
	m.HandleFunc("/report", ti.SaveTicket).Methods("POST", "OPTIONS")
	m.HandleFunc("/close-ticket/{ticket_id}", ti.CloseTicket).Methods("POST", "OPTIONS")

	//chat bot
	m.HandleFunc("/conversation/{quiz_id}/{student_id}", conv.GetConversationForQuiz).Methods("GET", "OPTIONS")

	// Chats
	chatsRouter := m.PathPrefix("/chats").Subrouter()
	chatsRouter.HandleFunc("/ws", chat.ChatEventsWS).Methods("GET")
	chatsRouter.HandleFunc("", chat.GetChats).Methods("GET")
	chatsRouter.HandleFunc("/invites", chat.SendInvite).Methods("POST")
	chatsRouter.HandleFunc("/invites", chat.GetInvites).Methods("GET")
	chatsRouter.HandleFunc("/invites/{id}/accept", chat.AcceptInvite).Methods("POST")
	chatsRouter.HandleFunc("/invites/{id}/decline", chat.DeclineInvite).Methods("POST")
	chatsRouter.HandleFunc("/{chat_id}/messages", chat.GetChatMessages).Methods("GET")
	chatsRouter.HandleFunc("/{chat_id}/messages", chat.SendMessage).Methods("POST")

	// Allow API requests from frontend
	allowedOrigins := []string{"http://localhost:3000"}

	// Add production frontend URL if set
	if frontendURL := os.Getenv("FRONTEND_URL"); frontendURL != "" {
		allowedOrigins = append(allowedOrigins, frontendURL)
	}

	handler := cors.New(cors.Options{
		AllowedOrigins:   allowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type", "Authorization", "User-ID", "Is-Admin"},
		AllowCredentials: true,
	}).Handler(m)

	// Use PORT from environment (Railway) or default to 8060
	port := os.Getenv("PORT")
	if port == "" {
		port = "8060"
	}

	fmt.Printf("Server starting on port %s...\n", port)
	err = http.ListenAndServe(":"+port, handler)
	if err != nil {
		fmt.Println(err)
	}

}
