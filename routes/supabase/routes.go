package supabase

import (
	"context"
	"encoding/json"
	"io"
	"math"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"time"

	"fmt"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	supabase "github.com/nedpals/supabase-go"
	"gitlab.msu.edu/team-corewell-2025/auth"
	model "gitlab.msu.edu/team-corewell-2025/models"
	"golang.org/x/crypto/bcrypt"
)

var Supabase *supabase.Client

// Initializes the Database client
func InitClient(url, key string) *supabase.Client {
	Supabase = supabase.CreateClient(url, key)
	return Supabase
}

func generateUserID() string {
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		id := fmt.Sprintf("%06d", rand.Intn(1000000)) // random 6-digit string

		var existing []model.User
		err := Supabase.DB.From("users").Select("*").Eq("userId", id).Execute(&existing)

		if err != nil {
			fmt.Printf("generateUserID: error checking Supabase: %v\n", err)
			return id
		}

		if len(existing) == 0 {
			return id
		}

		fmt.Printf("collision trying again", id)
	}
	fmt.Println("using fallback of associating w/ time of creation")
	return fmt.Sprintf("%06d", time.Now().UnixNano()%1000000)
}

// generates random nickname.... ill make it better later
func generateNickname() string {
	adjectives := []string{
		"Swift", "Calm", "Brave", "Vital", "Gentle", "Sharp", "Steady", "Clever", "Heroic", "Nimble", "Bright", "Keen", "Lively", "Mighty", "Quick", "Vivid", "Wise", "Bold",
	}
	bodyParts := []string{
		"Neuron", "Scalpel", "Heartbeat", "Stethoscope", "Suture", "Plasma", "Capsule", "Tendon", "Pulse", "Cell", "Bandage", "Microscope", "Pulse", "Rhythm",
	}
	return adjectives[rand.Intn(len(adjectives))] + bodyParts[rand.Intn(len(bodyParts))]
}

func hashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Signs up the user
func SignUpUser(w http.ResponseWriter, r *http.Request) {
	var userRequest UserCreateRequest
	bodyBytes, err := io.ReadAll(r.Body)

	if err != nil {
		msg := fmt.Sprintf("SignUpUser: failed to read request body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest) // 400
		return
	}
	err = json.Unmarshal(bodyBytes, &userRequest)
	if err != nil {
		msg := fmt.Sprintf("SignUpUser: cannot unmarshal user from request: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest) // 400
		return
	}

	//now we create the uuid ourselves, not supabase auth
	internalID := uuid.New()
	uid := generateUserID()
	nickname := generateNickname()

	passwordHash, err := hashPassword(userRequest.Password)
	if err != nil {
		http.Error(w, "Failed to hash password", http.StatusInternalServerError)
		return
	}

	newUser := model.User{
		Id:               internalID, // uuid
		UserUniqueId:     uid,        //userRequest.UserId, // 6-digit login id
		Nickname:         nickname,
		PasswordHash:     passwordHash,
		IsAdmin:          userRequest.IsAdmin,
		StudentStanding:  &userRequest.StudentStanding,
		Name:             userRequest.Name,
		Icon:             &userRequest.Icon,
		Rotation:         &userRequest.Rotation,
		Mode:             userRequest.Mode,
		ImprovementAreas: userRequest.ImprovementAreas,
		FeedbackType:    userRequest.FeedbackType,
	}

	//insert user to supabase
	err = Supabase.DB.From("users").Insert(newUser).Execute(nil)
	if err != nil {
		msg := fmt.Sprintf("SignUpUser: insert to DB failed, possibly conflict: %v", err)
		fmt.Println(msg)
		http.Error(w, "User has already been created", http.StatusConflict) // 409
		return

	}
	fmt.Printf("DEBUG: Received userRequest: %+v\n", userRequest)
	fmt.Printf("DEBUG: Icon: '%s', Rotation: '%s',isDarkMode: '%s', ImprovementAreas: %v\n",
		userRequest.Icon, userRequest.Rotation, userRequest.Mode, userRequest.ImprovementAreas)

	nickname = newUser.Nickname
	if nickname == "" {
		nickname = newUser.Name
	}

	token, err := auth.GenerateJWT(newUser.UserUniqueId, nickname, newUser.IsAdmin)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}
	fmt.Printf("DEBUG: Token Created: %+v\n", token)

	//make a json response to tell the frontend the user was created
	response := map[string]interface{}{
		"success":      true,
		"userId":       newUser.Id,
		"token":        token,
		"userUniqueId": newUser.UserUniqueId,
		"nickname":     newUser.Nickname,
		"isAdmin":      newUser.IsAdmin,
	}

	fmt.Printf("DEBUG: Unique Id: %+v\n", newUser.UserUniqueId)
	fmt.Printf("DEBUG: UU Id: %+v\n", newUser.Id)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Signs in the user
func SignInUser(w http.ResponseWriter, r *http.Request) {
	var userRequest UserLoginRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("SignInUser: failed to read request body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(bodyBytes, &userRequest)
	if err != nil {
		msg := fmt.Sprintf("SignInUser: failed to Unmarshal request body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	// Query Supabase for user by userId
	var users []model.User
	err = Supabase.DB.From("users").Select("*").Eq("userId", userRequest.ID).Execute(&users)

	fmt.Printf("DEBUG: unique id: %+v\n", userRequest.ID)

	if err != nil {
		http.Error(w, fmt.Sprintf("SignInUser: Supabase query error: %v", err), http.StatusInternalServerError)
		return
	}

	if len(users) == 0 {
		http.Error(w, "Invalid user ID or password", http.StatusUnauthorized)
		return
	}

	user := users[0]

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(userRequest.Password)); err != nil {
		http.Error(w, "Invalid user ID or password", http.StatusUnauthorized)
		return
	}
	if err != nil {
		msg := fmt.Sprintf("Supabase Sign In Error: %v", err)
		fmt.Println(msg)
		http.Error(w, "Sign In User Error", http.StatusNotAcceptable)
		return
	}

	token, err := auth.GenerateJWT(user.UserUniqueId, user.Nickname, user.IsAdmin)
	if err != nil {
		http.Error(w, "Failed to create token", http.StatusInternalServerError)
		return
	}

	// Return token + basic user info
	response := map[string]interface{}{
		"success":      true,
		"token":        token,
		"userId":       user.Id,           // UUID
		"userUniqueId": user.UserUniqueId, // 6-digit login ID
		"nickname":     user.Nickname,
		"isAdmin":      user.IsAdmin,
	}

	fmt.Printf("DEBUG: UUID userId: %+v\n", user.Id)
	fmt.Printf("DEBUG: UNIQUE userId: %+v\n", user.UserUniqueId)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	/*
		Supabase.DB.AddHeader("Authorization", "Bearer "+user.AccessToken)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	*/
}

func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var ForgotPasswordRequest ForgotPasswordRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("Error Reading Forgot Password Request Body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	err = json.Unmarshal(bodyBytes, &ForgotPasswordRequest)
	if err != nil {
		msg := fmt.Sprintf("ForgotPassword: cannot unmarshal request body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest) // 400
		return
	}
	ctx := context.Background()
	// Get frontend URL from environment or default to localhost
	frontendURL := os.Getenv("FRONTEND_URL")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}
	err = Supabase.Auth.ResetPasswordForEmail(ctx, ForgotPasswordRequest.Email, frontendURL+"/reset-password")
	if err != nil {
		msg := fmt.Sprintf("ForgotPassword: ResetPasswordForEmail failed: %v", err)
		fmt.Println(msg)
		http.Error(w, "Failed to send reset password", http.StatusInternalServerError) // 500
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Reset password link sent (if email is valid)"))
}

func ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		msg := fmt.Sprintf("ResetPassword: cannot parse request: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest) // 400
		return
	}
	ctx := context.Background()
	err = ResetUserPassword(ctx, Supabase, req.AccessToken, req.NewPassword)
	if err != nil {
		msg := fmt.Sprintf("ResetPassword: error resetting user password: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError) // 500
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Password updated successfully"))
}
func ResetUserPassword(ctx context.Context, supabaseClient *supabase.Client, accessToken string, newPassword string) error {
	user, err := supabaseClient.Auth.User(ctx, accessToken)
	if err != nil {
		return err
	}

	if user == nil || user.ID == "" {
		fmt.Println("User Not Found")
	}
	_, err = supabaseClient.Auth.UpdateUser(ctx, accessToken, map[string]interface{}{
		"password": newPassword,
	})
	return err
}

func GetStudents(w http.ResponseWriter, r *http.Request) {
	var students []model.User
	err := Supabase.DB.From("users").Select("*").Eq("isAdmin", "FALSE").Execute(&students)
	if err != nil {
		http.Error(w, "No Students Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(students)
}

func GetStudentById(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]
	var student []model.User
	err := Supabase.DB.From("users").Select("*").Eq("id", id).Execute(&student)
	if err != nil {
		msg := fmt.Sprintf("GetStudents: DB error: %v", err)
		fmt.Println(msg)
		http.Error(w, "No Students Found", http.StatusNotFound)
		return
	}
	if len(student) == 0 {
		http.Error(w, "No Students Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(student[0])
}

func GetInstructors(w http.ResponseWriter, r *http.Request) {
	var instructors []model.User
	err := Supabase.DB.From("users").Select("*").Eq("isAdmin", "TRUE").Execute(&instructors)

	fmt.Printf("DEBUG: Instructor Info: %+v\n", instructors)
	if err != nil {
		msg := fmt.Sprintf("GetInstructors: DB error: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusNotFound)
		return
	}

	if len(instructors) == 0 {
		http.Error(w, "No Instructors Found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(instructors)
}

func AddStudentToInstructor(w http.ResponseWriter, r *http.Request) {
	var req AddStudentRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("AddStudentToInstructor: failed to read request body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		msg := fmt.Sprintf("AddStudentToInstructor: invalid JSON: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	instructorID, err := uuid.Parse(req.InstructorId)
	if err != nil {
		http.Error(w, "Invalid instructor UUID", http.StatusBadRequest)
		return
	}
	studentID, err := uuid.Parse(req.StudentId)
	if err != nil {
		msg := fmt.Sprintf("AddStudentToInstructor: invalid instructor UUID (%s): %v", req.InstructorId, err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	var instructorsFound []model.User
	err = Supabase.DB.From("users").
		Select("*").
		Eq("id", instructorID.String()).
		Execute(&instructorsFound)

	fmt.Printf("DEBUG: Instructor Info: %+v\n", instructorsFound)

	if err != nil {
		msg := fmt.Sprintf("AddStudentToInstructor: error fetching instructor (id=%s): %v", instructorID.String(), err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	if len(instructorsFound) == 0 {
		http.Error(w, "Instructor not found", http.StatusNotFound)
		return
	}

	instructor := instructorsFound[0]
	if !instructor.IsAdmin {
		http.Error(w, "User is not an instructor", http.StatusForbidden)
		return
	}

	for _, s := range instructor.Students {
		if s == studentID {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("Student already assigned"))
			return
		}
	}
	instructor.Students = append(instructor.Students, studentID)

	updateData := map[string]any{
		"students": instructor.Students,
	}
	err = Supabase.DB.From("users").
		Update(updateData).
		Eq("id", instructor.Id.String()).
		Execute(nil)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error updating instructor", http.StatusInternalServerError)
		return
	}
	updateData = map[string]any{
		"isAssigned": true,
	}
	err = Supabase.DB.From("users").
		Update(updateData).
		Eq("id", studentID.String()).
		Execute(nil)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Error updating student isAssigned", http.StatusInternalServerError)
		return
	}

	// Get student information for the notification
	var studentsFound []model.User
	err = Supabase.DB.From("users").
		Select("*").
		Eq("id", studentID.String()).
		Execute(&studentsFound)

	if err == nil && len(studentsFound) > 0 {
		student := studentsFound[0]

		// Create notification for the instructor
		notification := model.Notification{
			Id:        uuid.New(),
			UserId:    instructorID,
			Type:      "new_student",
			Title:     "New Student Assigned",
			Message:   fmt.Sprintf("%s (%s) - ID: %s has selected you as their instructor.", student.Name, student.Nickname, student.UserUniqueId),
			IsRead:    false,
			CreatedAt: model.FlexibleTime{Time: time.Now()},
		}

		err = Supabase.DB.From("notifications").
			Insert(notification).
			Execute(nil)
		if err != nil {
			// Log error but don't fail the request
			fmt.Printf("AddStudentToInstructor: Failed to create notification for instructor: %v\n", err)
		} else {
			fmt.Printf("DEBUG: Created notification for instructor %s about new student %s\n", instructorID.String(), studentID.String())
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("Student Added to Instructor"))
}

func GetInstructorStudents(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instructorID := vars["id"]

	fmt.Printf("DEBUG: TEACHER UUIID: %+v\n", instructorID)

	instructorUUID, err := uuid.Parse(instructorID)
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Invalid instructor UUID", http.StatusBadRequest)
		return
	}

	var instructorsFound []model.User
	fmt.Println("TESTING: ",instructorUUID.String())
	err = Supabase.DB.From("users").
		Select("*").
		Eq("id", instructorUUID.String()).
		Execute(&instructorsFound)
	if err != nil {
		msg := fmt.Sprintf("GetInstructorStudents: DB error (id=%s): %v", instructorUUID.String(), err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	if len(instructorsFound) == 0 {
		http.Error(w, "Instructor not found", http.StatusNotFound)
		return
	}

	instructor := instructorsFound[0]
	if !instructor.IsAdmin {
		http.Error(w, "User is not an instructor", http.StatusForbidden)
		return
	}

	if len(instructor.Students) == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("[]"))
		return
	}

	ids := make(map[string]interface{})

	var studentIDStrings []string
	for _, student_id := range instructor.Students {
		studentIDStrings = append(studentIDStrings, student_id.String())
	}
	ids["student_id"] = studentIDStrings
	//studentIDStrings = []string{studentIDStrings[1]}

	type User struct {
		Nickname               string    `json:"nickname"`
		UserId                 string    `json:"userId"`
		Id                     uuid.UUID `json:"id"`
		StudentStanding        *string   `json:"studentStanding"`
		TimeOnline             int       `json:"timeOnline"`
		LastHeartbeat          time.Time `json:"lastHeartbeat"`
		FormattedTimeOnline    string    `json:"formattedTimeOnline"`
		FormattedLastHeartBeat string    `json:"formattedLastHeartbeat"`
		FormattedActivity      string    `json:"formattedActivity"`
		Count                  int       `json:"count"`
		Score                  float64   `json:"score"`
	}
	var students []User

	type CountingUser struct {
		ID    uuid.UUID `json:"id"`
		Count int       `json:"count"`
	}
	var countingStudents []CountingUser

	type PercentUser struct {
		ID    uuid.UUID `json:"id"`
		Score float64   `json:"score"`
	}
	var percentStudents []PercentUser

	err = Supabase.DB.From("users").
		Select("*").
		In("id", studentIDStrings).
		Execute(&students)

	fmt.Println("STUDENTS: ",students)

	err = Supabase.DB.Rpc("hello_world", ids).Execute(&countingStudents)
	if err != nil {
		msg := fmt.Sprintf("hello_world RPC error:", studentIDStrings, err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	err = Supabase.DB.Rpc("hello_world2", ids).Execute(&percentStudents)
	if err != nil {
		msg := fmt.Sprintf("hello_world2 RPC error:", studentIDStrings, err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	for i := 0; i < len(students); i++ {

		for j := 0; j < len(countingStudents); j++ {
			if countingStudents[j].ID == students[i].Id {
				students[i].Count = countingStudents[j].Count
			}
		}

		for j := 0; j < len(percentStudents); j++ {
			if percentStudents[j].ID == students[i].Id {
				students[i].Score = float64(int(math.Round(percentStudents[j].Score / .09)))
			}
		}

		students[i].FormattedTimeOnline = time.Unix(int64(students[i].TimeOnline-68400), 0).Format("15:04:05")
		students[i].FormattedLastHeartBeat = students[i].LastHeartbeat.In(time.Local).Format("01/02/2006 3:04 pm")
		if students[i].LastHeartbeat.IsZero() {
			students[i].FormattedLastHeartBeat = "Never"
		}
		students[i].FormattedActivity = "Offline"
		if int(math.Round(time.Now().Sub(students[i].LastHeartbeat).Seconds())) <= 6 {
			students[i].FormattedActivity = "Online"
		}

	}

	if err != nil {
		msg := fmt.Sprintf("GetInstructorStudents: error fetching students for IDs=%v: %v", studentIDStrings, err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(students)
}

// GetStudentInstructor gets the instructor information for a specific student
func GetStudentInstructor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentID := vars["student_id"]

	studentUUID, err := uuid.Parse(studentID)
	if err != nil {
		fmt.Println("Invalid student UUID:", err)
		http.Error(w, "Invalid student UUID", http.StatusBadRequest)
		return
	}

	// Get all instructors
	var instructors []model.User
	err = Supabase.DB.From("users").
		Select("*").
		Eq("isAdmin", "true").
		Execute(&instructors)

	if err != nil {
		msg := fmt.Sprintf("GetStudentInstructor: DB error: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Find which instructor has this student
	for _, instructor := range instructors {
		for _, sid := range instructor.Students {
			if sid == studentUUID {
				// Found the instructor
				response := map[string]interface{}{
					"id":       instructor.Id,
					"name":     instructor.Name,
					"nickname": instructor.Nickname,
					"userId":   instructor.UserUniqueId,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
				return
			}
		}
	}

	// Student not assigned to any instructor
	http.Error(w, "Student not assigned to an instructor", http.StatusNotFound)
}

func RemoveStudentFromInstructor(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instructorID := vars["instructor_id"]
	studentID := vars["student_id"]

	fmt.Printf("DEBUG: Removing student %s from instructor %s\n", studentID, instructorID)

	// Parse UUIDs
	instructorUUID, err := uuid.Parse(instructorID)
	if err != nil {
		fmt.Println("Invalid instructor UUID:", err)
		http.Error(w, "Invalid instructor UUID", http.StatusBadRequest)
		return
	}

	studentUUID, err := uuid.Parse(studentID)
	if err != nil {
		fmt.Println("Invalid student UUID:", err)
		http.Error(w, "Invalid student UUID", http.StatusBadRequest)
		return
	}

	// Get the instructor
	var instructorsFound []model.User
	err = Supabase.DB.From("users").
		Select("*").
		Eq("id", instructorUUID.String()).
		Execute(&instructorsFound)
	if err != nil {
		msg := fmt.Sprintf("RemoveStudentFromInstructor: DB error (id=%s): %v", instructorUUID.String(), err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}
	if len(instructorsFound) == 0 {
		http.Error(w, "Instructor not found", http.StatusNotFound)
		return
	}

	instructor := instructorsFound[0]
	if !instructor.IsAdmin {
		http.Error(w, "User is not an instructor", http.StatusForbidden)
		return
	}

	// Remove the student from the instructor's student list
	updatedStudents := []uuid.UUID{}
	studentFound := false
	for _, id := range instructor.Students {
		if id == studentUUID {
			studentFound = true
			continue // Skip this student (remove them)
		}
		updatedStudents = append(updatedStudents, id)
	}

	if !studentFound {
		http.Error(w, "Student not found in instructor's list", http.StatusNotFound)
		return
	}

	// Update the instructor in the database
	updateData := map[string]interface{}{
		"students": updatedStudents,
	}

	err = Supabase.DB.From("users").
		Update(updateData).
		Eq("id", instructorUUID.String()).
		Execute(nil)
	if err != nil {
		msg := fmt.Sprintf("RemoveStudentFromInstructor: Failed to update instructor: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Update the student's isAssigned status to false
	studentUpdateData := map[string]interface{}{
		"isAssigned": false,
	}

	err = Supabase.DB.From("users").
		Update(studentUpdateData).
		Eq("id", studentUUID.String()).
		Execute(nil)
	if err != nil {
		msg := fmt.Sprintf("RemoveStudentFromInstructor: Failed to update student isAssigned status: %v", err)
		fmt.Println(msg)
		// Don't return error here as the main operation succeeded
	}

	// Create notification for the student
	actionUrl := "/SelectInstructor"
	notification := model.Notification{
		Id:        uuid.New(),
		UserId:    studentUUID,
		Type:      "instructor_removed",
		Title:     "Instructor Assignment Ended",
		Message:   fmt.Sprintf("Your instructor has removed you from their class. Please select a new instructor to continue your studies."),
		IsRead:    false,
		CreatedAt: model.FlexibleTime{Time: time.Now()},
		ActionUrl: &actionUrl,
	}

	err = Supabase.DB.From("notifications").
		Insert(notification).
		Execute(nil)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("RemoveStudentFromInstructor: Failed to create notification: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Created notification for student %s about instructor removal\n", studentUUID.String())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Student removed successfully",
	})
}

func ReassignStudent(w http.ResponseWriter, r *http.Request) {
	type ReassignRequest struct {
		StudentId       string `json:"studentId"`
		OldInstructorId string `json:"oldInstructorId"`
		NewInstructorId string `json:"newInstructorId"`
	}

	var req ReassignRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("ReassignStudent: failed to read request body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		msg := fmt.Sprintf("ReassignStudent: invalid JSON: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	fmt.Printf("DEBUG: Reassigning student %s from instructor %s to %s\n", req.StudentId, req.OldInstructorId, req.NewInstructorId)

	// Parse UUIDs
	studentUUID, err := uuid.Parse(req.StudentId)
	if err != nil {
		http.Error(w, "Invalid student UUID", http.StatusBadRequest)
		return
	}
	oldInstructorUUID, err := uuid.Parse(req.OldInstructorId)
	if err != nil {
		http.Error(w, "Invalid old instructor UUID", http.StatusBadRequest)
		return
	}
	newInstructorUUID, err := uuid.Parse(req.NewInstructorId)
	if err != nil {
		http.Error(w, "Invalid new instructor UUID", http.StatusBadRequest)
		return
	}

	// Get old instructor
	var oldInstructorsFound []model.User
	err = Supabase.DB.From("users").
		Select("*").
		Eq("id", oldInstructorUUID.String()).
		Execute(&oldInstructorsFound)
	if err != nil || len(oldInstructorsFound) == 0 {
		http.Error(w, "Old instructor not found", http.StatusNotFound)
		return
	}
	oldInstructor := oldInstructorsFound[0]

	// Get new instructor
	var newInstructorsFound []model.User
	err = Supabase.DB.From("users").
		Select("*").
		Eq("id", newInstructorUUID.String()).
		Execute(&newInstructorsFound)
	if err != nil || len(newInstructorsFound) == 0 {
		http.Error(w, "New instructor not found", http.StatusNotFound)
		return
	}
	newInstructor := newInstructorsFound[0]

	// Remove student from old instructor's list
	updatedOldStudents := []uuid.UUID{}
	for _, id := range oldInstructor.Students {
		if id != studentUUID {
			updatedOldStudents = append(updatedOldStudents, id)
		}
	}

	// Add student to new instructor's list
	updatedNewStudents := append(newInstructor.Students, studentUUID)

	// Update old instructor
	err = Supabase.DB.From("users").
		Update(map[string]interface{}{"students": updatedOldStudents}).
		Eq("id", oldInstructorUUID.String()).
		Execute(nil)
	if err != nil {
		msg := fmt.Sprintf("ReassignStudent: Failed to update old instructor: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Update new instructor
	err = Supabase.DB.From("users").
		Update(map[string]interface{}{"students": updatedNewStudents}).
		Eq("id", newInstructorUUID.String()).
		Execute(nil)
	if err != nil {
		msg := fmt.Sprintf("ReassignStudent: Failed to update new instructor: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Get student information for notifications
	var studentsFound []model.User
	err = Supabase.DB.From("users").
		Select("*").
		Eq("id", studentUUID.String()).
		Execute(&studentsFound)

	var studentName string
	var studentNickname string
	var studentUserId string
	if err == nil && len(studentsFound) > 0 {
		student := studentsFound[0]
		studentName = student.Name
		studentNickname = student.Nickname
		studentUserId = student.UserUniqueId
	} else {
		studentName = "A student"
		studentNickname = "Student"
		studentUserId = ""
	}

	// Create notification for the student
	studentNotification := model.Notification{
		Id:        uuid.New(),
		UserId:    studentUUID,
		Type:      "instructor_assignment",
		Title:     "New Instructor Assigned",
		Message:   fmt.Sprintf("You have been assigned to a new instructor: %s (%s)", newInstructor.Name, newInstructor.Nickname),
		IsRead:    false,
		CreatedAt: model.FlexibleTime{Time: time.Now()},
	}

	err = Supabase.DB.From("notifications").
		Insert(studentNotification).
		Execute(nil)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("ReassignStudent: Failed to create notification for student: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Created notification for student %s about instructor reassignment\n", studentUUID.String())
	}

	// Create notification for the new instructor
	instructorNotification := model.Notification{
		Id:        uuid.New(),
		UserId:    newInstructorUUID,
		Type:      "new_student",
		Title:     "New Student Assigned",
		Message:   fmt.Sprintf("%s (%s) - ID: %s has been reassigned to you by %s.", studentName, studentNickname, studentUserId, oldInstructor.Name),
		IsRead:    false,
		CreatedAt: model.FlexibleTime{Time: time.Now()},
	}

	err = Supabase.DB.From("notifications").
		Insert(instructorNotification).
		Execute(nil)
	if err != nil {
		// Log error but don't fail the request
		fmt.Printf("ReassignStudent: Failed to create notification for new instructor: %v\n", err)
	} else {
		fmt.Printf("DEBUG: Created notification for new instructor %s about new student %s\n", newInstructorUUID.String(), studentUUID.String())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Student reassigned successfully",
	})
}

func GetAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	instructorID := vars["id"]

	type Aggregations struct {
		Sum   int `json:"sum"`
		Count int `json:"count"`
	}

	type Analytics struct {
		Tasks           int          `json:"tasks"`
		Aggregation     Aggregations `json:"aggregation"`
		TotalTimeOnline string       `json:"totalTimeOnline"`
		AverageScore    int          `json:"averageScore"`
		Quizes          float64      `json:"quizes"`
		QuizesCompleted float64      `json:"quizesCompleted"`
	}

	var analytics Analytics

	var instructor model.User
	err := Supabase.DB.From("users").Select("*").Single().Eq("id", instructorID).Execute(&instructor)
	if err != nil {
		fmt.Println(err)
		return
	}

	ids := make(map[string]interface{})
	var studentIDs []string
	for _, student_id := range instructor.Students {
		studentIDs = append(studentIDs, student_id.String())
	}
	ids["student_id"] = studentIDs

	err = Supabase.DB.From("tasks").Select("*").Count().In("user_id", studentIDs).Eq("completed", "true").Execute(&analytics.Tasks)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = Supabase.DB.From("users").Select("timeOnline.sum()", "id.count()").Single().In("id", studentIDs).Eq("isAdmin", "false").Execute(&analytics.Aggregation)
	if err != nil {
		fmt.Println(err)
		return
	}

	type PercentUser struct {
		ID    uuid.UUID `json:"id"`
		Score float64   `json:"score"`
	}
	var percentStudents []PercentUser

	err = Supabase.DB.Rpc("hello_world2", ids).Execute(&percentStudents)
	for i := 0; i < len(percentStudents); i++ {
		analytics.AverageScore += int(math.Round(percentStudents[i].Score / .09))
	}

	if analytics.Aggregation.Count > 0 {
		analytics.AverageScore /= analytics.Aggregation.Count
	} else {
		analytics.AverageScore = 0
	}

	err = Supabase.DB.From("records").Select("*").Count().In("user_id", studentIDs).Execute(&analytics.Quizes)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = Supabase.DB.From("records").Select("*").Count().In("user_id", studentIDs).Neq("score", "0").Execute(&analytics.QuizesCompleted)
	if err != nil {
		fmt.Println(err)
		return
	}

	analytics.TotalTimeOnline = time.Unix(int64(analytics.Aggregation.Sum-68400), 0).Format("15:04:05")
	analytics.QuizesCompleted = (math.Ceil(analytics.QuizesCompleted / analytics.Quizes))

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

func CalculateAchievementTotal(a Achievements) int {
	total := 1

	// Streak Achievements
	streakMilestones := []int{3, 7, 14}
	for _, milestone := range streakMilestones {
		if a.MaxTaskStreak >= milestone {
			total++
		}
	}

	for _, milestone := range streakMilestones {
		if a.MaxTaskStreak >= milestone {
			total++
		}
	}

	// Tasks Completed Achievement
	taskMilestones := []int{1, 21, 42, 84}
	for _, milestone := range taskMilestones {
		if a.TaskCount >= milestone {
			total++
		}
	}

	examMilestones := []int{1, 5, 15}
	for _, milestone := range examMilestones {
		if a.QuizCount >= milestone {
			total++
		}
	}

	// Related Quiz Achievement
	if a.RelatedQuizCount >= 10 {
		total++
	}

	// Extended Message Achievement
	if a.ExtendedMessageCount >= 5 {
		total++
	}

	return total

}

func GetStudentAnalytics(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	studentID := vars["id"]

	type Analytics struct {
		QuestionTasks        int     `json:"questionTasks"`
		LabTasks             int     `json:"labTasks"`
		PrescriptionTasks    int     `json:"prescriptionTasks"`
		TotalTimeSpent       string  `json:"totalTimeSpent"`
		AverageQuizScore     float64 `json:"averageQuizScore"`
		QuizCompletionRating float64 `json:"quizCompletionRating"`
		MaxTaskStreak        int     `json:"maxTaskStreak"`
		MaxQuizStreak        int     `json:"maxQuizStreak"`
		AchievementCount     int     `json:"achievementCount"`
	}

	var analytics Analytics

	err := Supabase.DB.From("tasks").
		Select("*").
		Count().
		Eq("user_id", studentID).
		Eq("completed", "true").
		Eq("task_type", "patient_question").
		Execute(&analytics.QuestionTasks)
	if err != nil {
		fmt.Printf("Error fetching tasks: %v\n", err)
	}

	err = Supabase.DB.From("tasks").
		Select("*").
		Count().
		Eq("user_id", studentID).
		Eq("completed", "true").
		Eq("task_type", "lab_result").
		Execute(&analytics.LabTasks)
	if err != nil {
		fmt.Printf("Error fetching tasks: %v\n", err)
	}

	err = Supabase.DB.From("tasks").
		Select("*").
		Count().
		Eq("user_id", studentID).
		Eq("completed", "true").
		Eq("task_type", "prescription").
		Execute(&analytics.PrescriptionTasks)
	if err != nil {
		fmt.Printf("Error fetching tasks: %v\n", err)
	}

	var student model.User
	err = Supabase.DB.From("users").
		Select("timeOnline").
		Single().
		Eq("id", studentID).
		Execute(&student)
	if err != nil {
		fmt.Printf("Error fetching student: %v\n", err)
	} else {
		analytics.TotalTimeSpent = time.Unix(int64(student.TimeOnline-68400), 0).Format("15:04:05")
	}

	type QuizRecord struct {
		Score int `json:"score"`
	}
	var quizRecords []QuizRecord
	err = Supabase.DB.From("records").
		Select("score").
		Eq("user_id", studentID).
		Execute(&quizRecords)

	if err != nil {
		fmt.Printf("Error fetching quiz records: %v\n", err)
	}

	if len(quizRecords) > 0 {
		totalScore := 0
		completedQuizzes := 0
		for _, record := range quizRecords {
			if record.Score > 0 {
				totalScore += (record.Score * 11)
				completedQuizzes++
			}
		}
		if completedQuizzes > 0 {
			analytics.AverageQuizScore = math.Round(float64(totalScore)/float64(completedQuizzes)*100) / 100
			analytics.QuizCompletionRating = math.Round(float64(completedQuizzes)/float64(len(quizRecords))*100*100) / 100
		} else {
			analytics.AverageQuizScore = 0
			analytics.QuizCompletionRating = 0
		}
	} else {
		analytics.AverageQuizScore = 0
		analytics.QuizCompletionRating = 0
	}

	var achievement Achievements
	err = Supabase.DB.From("achievements").
		Select("*").
		Single().
		Eq("user_id", studentID).
		Execute(&achievement)

	if err == nil {
		analytics.MaxQuizStreak = achievement.MaxQuizStreak
		analytics.MaxTaskStreak = achievement.MaxTaskStreak
	} else {
		analytics.MaxQuizStreak = 0
		analytics.MaxTaskStreak = 0
	}

	analytics.AchievementCount = CalculateAchievementTotal(achievement)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(analytics)
}

func GetLeaderboard(w http.ResponseWriter, r *http.Request) {
	// Get timeFrame query parameter (all, week, month)
	timeFrame := r.URL.Query().Get("timeFrame")
	if timeFrame == "" {
		timeFrame = "all"
	}

	type LeaderboardEntry struct {
		Id             uuid.UUID `json:"id"`
		Name           string    `json:"name"`
		Nickname       string    `json:"nickname"`
		Icon           *string   `json:"icon"`
		TasksCompleted int       `json:"tasksCompleted"`
		AvgQuizScore   float64   `json:"avgQuizScore"`
		Border         string    `json:"border,omitempty"`
		TaskStreak     int       `json:"taskStreak"`
		MaxTaskStreak  int       `json:"maxTaskStreak"`
	}

	// Get students
	var students []model.User
	err := Supabase.DB.From("users").Select("*").Eq("isAdmin", "false").Execute(&students)
	if err != nil {
		msg := fmt.Sprintf("GetLeaderboard: error fetching students: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Calculate cutoff time based on timeFrame
	var cutoffTime time.Time
	now := time.Now()
	switch timeFrame {
	case "week":
		cutoffTime = now.AddDate(0, 0, -7)
	case "month":
		cutoffTime = now.AddDate(0, -1, 0)
	default:
		// "all" - no cutoff, use zero time
		cutoffTime = time.Time{}
	}

	// Fetch tasks
	type TaskRecord struct {
		UserId uuid.UUID `json:"user_id"`
	}
	var allTasks []TaskRecord
	if cutoffTime.IsZero() {
		err = Supabase.DB.From("tasks").
			Select("user_id").
			Eq("completed", "true").
			Execute(&allTasks)
	} else {
		err = Supabase.DB.From("tasks").
			Select("user_id").
			Eq("completed", "true").
			Gte("completed_at", cutoffTime.Format(time.RFC3339)).
			Execute(&allTasks)
	}
	if err != nil {
		fmt.Printf("GetLeaderboard: error fetching tasks: %v\n", err)
		http.Error(w, "Error fetching tasks", http.StatusInternalServerError)
		return
	}

	// Build a map of user_id -> task count
	taskCountMap := make(map[uuid.UUID]int)
	for _, task := range allTasks {
		taskCountMap[task.UserId]++
	}

	// Fetch quiz records
	type QuizRecord struct {
		UserId uuid.UUID `json:"user_id"`
		Score  int       `json:"score"`
	}
	var allQuizRecords []QuizRecord
	err = Supabase.DB.From("records").
		Select("user_id,score").
		Execute(&allQuizRecords)
	if err != nil {
		fmt.Printf("GetLeaderboard: error fetching quiz records: %v\n", err)
		http.Error(w, "Error fetching quiz records", http.StatusInternalServerError)
		return
	}

	// Build a map of user_id -> quiz scores
	quizScoresMap := make(map[uuid.UUID][]int)
	for _, record := range allQuizRecords {
		quizScoresMap[record.UserId] = append(quizScoresMap[record.UserId], record.Score)
	}

	// Fetch achievements data
	var allAchievements []Achievements
	err = Supabase.DB.From("achievements").
		Select("*").
		Execute(&allAchievements)
	if err != nil {
		fmt.Printf("GetLeaderboard: error fetching achievements: %v\n", err)
		http.Error(w, "Error fetching achievements", http.StatusInternalServerError)
		return
	}

	// Build a map of user_id -> achievements
	achievementsMap := make(map[uuid.UUID]Achievements)
	for _, achievement := range allAchievements {
		achievementsMap[achievement.UserId] = achievement
	}

	// Build leaderboard by processing students
	var leaderboard []LeaderboardEntry
	for _, student := range students {
		tasksCompleted := taskCountMap[student.Id]

		// Calculate average quiz score
		avgQuizScore := 0.0
		if scores, exists := quizScoresMap[student.Id]; exists && len(scores) > 0 {
			totalScore := 0
			for _, score := range scores {
				totalScore += score
			}
			avgQuizScore = float64(totalScore) / float64(len(scores))
		}

		// Get streak data from achievements
		taskStreak := 0
		maxTaskStreak := 0
		if achievement, exists := achievementsMap[student.Id]; exists {
			taskStreak = achievement.TaskStreak
			maxTaskStreak = achievement.MaxTaskStreak
		}

		leaderboard = append(leaderboard, LeaderboardEntry{
			Id:             student.Id,
			Name:           student.Name,
			Nickname:       student.Nickname,
			Icon:           student.Icon,
			TasksCompleted: tasksCompleted,
			AvgQuizScore:   math.Round(avgQuizScore*100) / 100, // Round to 2 decimal places
			Border:         student.Border,
			TaskStreak:     taskStreak,
			MaxTaskStreak:  maxTaskStreak,
		})
	}

	// Sort leaderboard by tasks completed (descending), then by avg quiz score
	sort.Slice(leaderboard, func(i, j int) bool {
		if leaderboard[i].TasksCompleted == leaderboard[j].TasksCompleted {
			return leaderboard[i].AvgQuizScore > leaderboard[j].AvgQuizScore
		}
		return leaderboard[i].TasksCompleted > leaderboard[j].TasksCompleted
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(leaderboard)
}

// achievements struct is only used by two functions below
type Achievements struct {
	UserId               uuid.UUID `json:"user_id"`
	TaskCount            int       `json:"task_count"`
	QuizCount            int       `json:"quiz_count"`
	LastTask             time.Time `json:"last_task"`
	LastQuiz             time.Time `json:"last_quiz"`
	TaskStreak           int       `json:"task_streak"`
	QuizStreak           int       `json:"quiz_streak"`
	RelatedQuizCount     int       `json:"related_quiz_count"`
	ExtendedMessageCount int       `json:"extended_message_count"`
	MaxTaskStreak        int       `json:"max_task_streak"`
	MaxQuizStreak        int       `json:"max_quiz_streak"`
}

// function to retrieve achievement data stored in achievement table
func GetAchievements(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId := vars["user_id"]

	var achievement Achievements
	err := Supabase.DB.From("achievements").
		Select("*").
		Single().
		Eq("user_id", userId).
		Execute(&achievement)
	if err != nil {
		id, _ := uuid.Parse(userId)
		var achievements []Achievements
		err2 := Supabase.DB.From("achievements").
			Insert(Achievements{UserId: id, TaskCount: 0, QuizCount: 0, TaskStreak: 0, QuizStreak: 0, RelatedQuizCount: 0, ExtendedMessageCount: 0, MaxTaskStreak: 0, MaxQuizStreak: 0}).
			Execute(&achievements)
		if len(achievements) > 0 {
			achievement = achievements[0]
		}
		if err2 != nil {
			msg := fmt.Sprintf("Get Achievements: invalid UUID (%s): %v", userId, err2)
			fmt.Println(err2)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(achievement)
}

// function to update that data (utilizing a upsert function in case of a non-existing row)
func UpdateAchievements(w http.ResponseWriter, r *http.Request) {
	var achievement Achievements
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = json.Unmarshal(bodyBytes, &achievement)
	if err != nil {
		fmt.Println(err)
		return
	}
	print(achievement.UserId.String())
	err = Supabase.DB.From("achievements").Upsert(achievement).Execute(nil)
	if err != nil {
		fmt.Println(err)
		return
	}
}

// function to generate medical journals
func GetPubMedArticles(w http.ResponseWriter, r *http.Request) {
	var req PubMedRequest
	err := json.NewDecoder(r.Body).Decode(&req)
	if err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	searchURL := fmt.Sprintf(
		"https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esearch.fcgi?db=pubmed&retmode=json&retmax=5&sort=relevance&term=%s",
		url.QueryEscape(req.Query),
	)

	resp, err := http.Get(searchURL)
	if err != nil {
		http.Error(w, "Failed to call PubMed", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	type SearchResponse struct {
		ESearchResult struct {
			IDList []string `json:"idlist"`
		} `json:"esearchresult"`
	}

	var searchData SearchResponse
	json.Unmarshal(body, &searchData)

	if len(searchData.ESearchResult.IDList) == 0 {
		w.Write([]byte(`{"articles":[]}`))
		return
	}

	ids := strings.Join(searchData.ESearchResult.IDList, ",")

	summaryURL := fmt.Sprintf(
		"https://eutils.ncbi.nlm.nih.gov/entrez/eutils/esummary.fcgi?db=pubmed&retmode=json&id=%s",
		ids,
	)

	summaryResp, err := http.Get(summaryURL)
	if err != nil {
		http.Error(w, "Failed to fetch article details", http.StatusInternalServerError)
		return
	}
	defer summaryResp.Body.Close()

	summaryBody, _ := io.ReadAll(summaryResp.Body)

	type SummaryResponse struct {
		Result map[string]json.RawMessage `json:"result"`
	}

	type SummaryItem struct {
		Title           string `json:"title"`
		FullJournalName string `json:"fulljournalname"`
		PubDate         string `json:"pubdate"`
	}

	var summaryData SummaryResponse
	err = json.Unmarshal(summaryBody, &summaryData)
	if err != nil {
		http.Error(w, "Failed to parse article details", http.StatusInternalServerError)
		return
	}

	articles := []PubMedArticle{}

	for _, id := range searchData.ESearchResult.IDList {
		rawItem, exists := summaryData.Result[id]
		if !exists {
			continue
		}

		var item SummaryItem
		err = json.Unmarshal(rawItem, &item)
		if err != nil {
			continue
		}

		year := 0
		fmt.Sscanf(item.PubDate, "%d", &year)

		currentYear := time.Now().Year()

		if currentYear-year > 10 {
			continue
		}

		article := PubMedArticle{
			PMID:    id,
			Title:   item.Title,
			Journal: item.FullJournalName,
			PubDate: item.PubDate,
			Link:    "https://pubmed.ncbi.nlm.nih.gov/" + id + "/",
		}

		articles = append(articles, article)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(articles)
}
