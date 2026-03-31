package supabase

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"time"

	model "gitlab.msu.edu/team-corewell-2025/models"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
)

// GetUserNotifications retrieves all notifications for a user
func GetUserNotifications(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "Invalid user UUID", http.StatusBadRequest)
		return
	}

	var notifications []model.Notification
	err = Supabase.DB.From("notifications").
		Select("*").
		Eq("userId", userUUID.String()).
		Execute(&notifications)

	if err != nil {
		msg := fmt.Sprintf("GetUserNotifications: DB error (userId=%s): %v", userUUID.String(), err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	// Sort notifications by createdAt descending (newest first) in Go
	sort.Slice(notifications, func(i, j int) bool {
		return notifications[i].CreatedAt.Time.After(notifications[j].CreatedAt.Time)
	})

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notifications)
}

// GetUnreadNotificationCount gets the count of unread notifications for a user
func GetUnreadNotificationCount(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "Invalid user UUID", http.StatusBadRequest)
		return
	}

	var notifications []model.Notification
	err = Supabase.DB.From("notifications").
		Select("*").
		Eq("userId", userUUID.String()).
		Eq("isRead", "false").
		Execute(&notifications)

	if err != nil {
		msg := fmt.Sprintf("GetUnreadNotificationCount: DB error (userId=%s): %v", userUUID.String(), err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{
		"count": len(notifications),
	})
}

// MarkNotificationAsRead marks a specific notification as read
func MarkNotificationAsRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	notificationID := vars["notification_id"]

	notificationUUID, err := uuid.Parse(notificationID)
	if err != nil {
		http.Error(w, "Invalid notification UUID", http.StatusBadRequest)
		return
	}

	updateData := map[string]interface{}{
		"isRead": true,
	}

	err = Supabase.DB.From("notifications").
		Update(updateData).
		Eq("id", notificationUUID.String()).
		Execute(nil)

	if err != nil {
		msg := fmt.Sprintf("MarkNotificationAsRead: DB error (id=%s): %v", notificationUUID.String(), err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Notification marked as read",
	})
}

// MarkAllNotificationsAsRead marks all notifications for a user as read
func MarkAllNotificationsAsRead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userID := vars["user_id"]

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		http.Error(w, "Invalid user UUID", http.StatusBadRequest)
		return
	}

	updateData := map[string]interface{}{
		"isRead": true,
	}

	err = Supabase.DB.From("notifications").
		Update(updateData).
		Eq("userId", userUUID.String()).
		Eq("isRead", "false").
		Execute(nil)

	if err != nil {
		msg := fmt.Sprintf("MarkAllNotificationsAsRead: DB error (userId=%s): %v", userUUID.String(), err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"message": "All notifications marked as read",
	})
}

// CreateNotification creates a new notification for a user
func CreateNotification(w http.ResponseWriter, r *http.Request) {
	type NotificationRequest struct {
		UserId  string `json:"userId"`
		Type    string `json:"type"`
		Title   string `json:"title"`
		Message string `json:"message"`
	}

	var req NotificationRequest
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		msg := fmt.Sprintf("CreateNotification: failed to read request body: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		msg := fmt.Sprintf("CreateNotification: invalid JSON: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusBadRequest)
		return
	}

	userUUID, err := uuid.Parse(req.UserId)
	if err != nil {
		http.Error(w, "Invalid user UUID", http.StatusBadRequest)
		return
	}

	notification := model.Notification{
		Id:        uuid.New(),
		UserId:    userUUID,
		Type:      req.Type,
		Title:     req.Title,
		Message:   req.Message,
		IsRead:    false,
		CreatedAt: model.FlexibleTime{Time: time.Now()},
	}

	err = Supabase.DB.From("notifications").
		Insert(notification).
		Execute(nil)

	if err != nil {
		msg := fmt.Sprintf("CreateNotification: DB error: %v", err)
		fmt.Println(msg)
		http.Error(w, msg, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(notification)
}
