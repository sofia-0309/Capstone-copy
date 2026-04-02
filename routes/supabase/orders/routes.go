package orders

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"

	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
	request "gitlab.msu.edu/team-corewell-2025/routes/supabase"
)

type OrdersHandler struct {
	Supabase *supabase.Client
}

func (h *OrdersHandler) GetOrders(w http.ResponseWriter, r *http.Request) {
	var orders []model.Order
	err := h.Supabase.DB.From("orders").Select("*").Execute(&orders)
	if err != nil {
		msg := fmt.Sprintf("GetOrders: DB select error: %v", err)
		fmt.Println(msg)
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}
	ordersJSON, err := json.MarshalIndent(orders, "", "  ")
	w.Header().Set("Content-Type", "application/json")
	w.Write(ordersJSON)
}

func (h *OrdersHandler) LogOrder(w http.ResponseWriter, r *http.Request) {
	var req request.InsertOrdersRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		fmt.Println(err)
		http.Error(w, "Error unmarshaling request", http.StatusBadRequest)
		return
	}
	now := time.Now()
	newOrder := request.InsertOrdersRequest{
		ID:        uuid.New(),
		PatientID: req.PatientID,
		TaskID:    req.TaskID,
		Name:      req.Name,
		Date:      &now,
		Details:   req.Details,
	}

	fmt.Printf("ORDER INSERT: %+v\n", newOrder)
	err = h.Supabase.DB.From("orders").Insert(newOrder).Execute(nil)
	if err != nil {
		fmt.Println(err)
		http.Error(w, fmt.Sprintf("Error inserting new order row: %v", err), http.StatusInternalServerError)
		return
	}

	clearFeedback := map[string]interface{}{
		"order_feedback": "",
	}

	err = h.Supabase.DB.From("orders").
		Update(clearFeedback).
		Eq("patient_id", req.PatientID.String()).
		Eq("task_id", req.TaskID.String()).
		Execute(nil)

	if err != nil {
		fmt.Println("Failed to clear old order feedback:", err)
		http.Error(w, "Order inserted but failed to clear old feedback", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func (h *OrdersHandler) GetOrderedOrders(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Backend", r.Method, "URL:", r.URL.Path)
	patientID := r.URL.Query().Get("patient_id")
	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, "Missing task_id parameter", http.StatusBadRequest)
		return
	}
	if patientID == "" {
		http.Error(w, "Missing patient_id parameter", http.StatusBadRequest)
		return
	}

	var orders []model.OrderedOrders

	// Query the database for orders for this patient
	err := h.Supabase.DB.From("orders").Select("*").Eq("patient_id", patientID).Eq("task_id", taskID).Execute(&orders)

	if err != nil {
		fmt.Printf("GetOrderedOrders: DB select error: %v\n", err)
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}

	// Marshal and return as JSON
	ordersJSON, err := json.MarshalIndent(orders, "", "  ")
	if err != nil {
		fmt.Printf("GetOrderedOrders: JSON marshal error: %v\n", err)
		http.Error(w, "Failed to marshal orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(ordersJSON)
}

func (h *OrdersHandler) GetOrdersFeedback(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Backend", r.Method, "URL:", r.URL.Path)

	patientID := r.URL.Query().Get("patient_id")
	if patientID == "" {
		http.Error(w, "Missing patient_id parameter", http.StatusBadRequest)
		return
	}

	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, "Missing task_id parameter", http.StatusBadRequest)
		return
	}

	var orders []model.OrderedOrders

	err := h.Supabase.DB.From("orders").Select("*").Eq("patient_id", patientID).Eq("task_id", taskID).Execute(&orders)
	if err != nil {
		fmt.Printf("GetOrdersFeedback: DB select error: %v\n", err)
		http.Error(w, "Failed to fetch orders", http.StatusInternalServerError)
		return
	}

	fmt.Println("Orders for AI feedback:", orders)

	var savedRows []struct {
		OrderFeedback string `json:"order_feedback"`
	}

	err = h.Supabase.DB.From("orders").
		Select("order_feedback").
		Eq("patient_id", patientID).
		Eq("task_id", taskID).
		Execute(&savedRows)

	if err == nil {
		for _, row := range savedRows {
			if row.OrderFeedback != "" {
				response := map[string]interface{}{
					"orders":   orders,
					"feedback": row.OrderFeedback,
				}

				responseJSON, _ := json.MarshalIndent(response, "", "  ")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write(responseJSON)
				return
			}
		}
	}

	var patients []model.Patient

	err = h.Supabase.DB.From("patients").Select("*").Eq("id", patientID).Execute(&patients)
	if err != nil {
		fmt.Printf("GetOrdersFeedback: patient DB select error: %v\n", err)
		http.Error(w, "Failed to fetch patient", http.StatusInternalServerError)
		return
	}

	if len(patients) == 0 {
		http.Error(w, "Patient not found", http.StatusNotFound)
		return
	}

	patient := patients[0]

	fmt.Println("Patient for AI feedback:", patient.Name)

	prompt := `
- You are a medical practicioner leaving feedback on a student's orders.
- Provide feedback as a numbered list, with each item on a new line.
- Use exactly this format:
	1. Order Name - appropriate/not appropriate: short reason (8 word limit)
- Do not use any other format.
- Use the format for all order numbers.
- You have a 100 word limit for the overall order_feedback.

`

	prompt += fmt.Sprintf(
		"Patient name: %s\nAge: %d\nGender: %s\nMedical history: %s\n\nOrders placed:\n",
		patient.Name,
		patient.Age,
		patient.Gender,
		patient.MedicalHistory,
	)

	for _, order := range orders {
		prompt += "- Order: " + order.Name + "\n"
	}

	fmt.Println("AI PROMPT:\n", prompt)

	llmRequest := map[string]interface{}{
		"task_type":    "orders_feedback",
		"patient":      patient,
		"order":        orders,
		"user_message": prompt,
	}

	requestBody, err := json.Marshal(llmRequest)
	if err != nil {
		fmt.Println("Error marshaling LLM request:", err)
		http.Error(w, "Failed to create LLM request", http.StatusInternalServerError)
		return
	}

	flaskURL := os.Getenv("FLASK_EXPLAIN_URL")
	if flaskURL == "" {
		//flaskURL = "http://127.0.0.1:5001/api/explain-request"
		flaskURL ="https://llm-flask-production.up.railway.app/explain-request "
	}

	resp, err := http.Post(flaskURL, "application/json", bytes.NewBuffer(requestBody))
	if err != nil {
		fmt.Println("Error contacting Flask microservice:", err)
		http.Error(w, "Failed to contact LLM microservice", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading LLM response:", err)
		http.Error(w, "Failed to read LLM response", http.StatusInternalServerError)
		return
	}

	fmt.Println("LLM raw response:", string(responseBytes))

	var feedbackText string
	err = json.Unmarshal(responseBytes, &feedbackText)
	if err != nil {
		feedbackText = string(responseBytes)
	}
	feedbackText = strings.ReplaceAll(feedbackText, "**", "")

	updateData := map[string]interface{}{
		"order_feedback": feedbackText,
	}

	err = h.Supabase.DB.From("orders").
		Update(updateData).
		Eq("patient_id", patientID).
		Eq("task_id", r.URL.Query().Get("task_id")).
		Execute(nil)

	if err != nil {
		fmt.Println("Failed to save order feedback:", err)
		http.Error(w, "Failed to save order feedback", http.StatusInternalServerError)
		return
	}

	feedbackMessage := "No orders placed yet."

	if len(orders) > 0 {

		feedbackMessage = feedbackText

	}

	response := map[string]interface{}{
		"orders":   orders,
		"feedback": feedbackMessage,
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		fmt.Println("JSON marshal error:", err)
		http.Error(w, "Failed to create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}

func (h *OrdersHandler) GetSavedOrdersFeedback(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Backend", r.Method, "URL:", r.URL.Path)

	patientID := r.URL.Query().Get("patient_id")
	if patientID == "" {
		http.Error(w, "Missing patient_id parameter", http.StatusBadRequest)
		return
	}

	taskID := r.URL.Query().Get("task_id")
	if taskID == "" {
		http.Error(w, "Missing task_id parameter", http.StatusBadRequest)
		return
	}

	var rows []struct {
		OrderFeedback string `json:"order_feedback"`
	}

	err := h.Supabase.DB.From("orders").
		Select("order_feedback").
		Eq("patient_id", patientID).
		Eq("task_id", taskID).
		Execute(&rows)

	if err != nil {
		fmt.Println("GetSavedOrdersFeedback DB error:", err)
		http.Error(w, "Failed to fetch saved order feedback", http.StatusInternalServerError)
		return
	}

	feedbackMessage := ""

	for _, row := range rows {
		if row.OrderFeedback != "" {
			feedbackMessage = row.OrderFeedback
			break
		}
	}

	response := map[string]interface{}{
		"feedback": feedbackMessage,
	}

	responseJSON, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		fmt.Println("JSON marshal error:", err)
		http.Error(w, "Failed to create response", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseJSON)
}
