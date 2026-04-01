package chats

import (
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	supabase "github.com/nedpals/supabase-go"
)

var sixDigitUserIDRegex = regexp.MustCompile(`^\d{6}$`)

type ChatHandler struct {
	Supabase *supabase.Client
}

type chatInvite struct {
	ID         int64      `json:"id"`
	FromUserID string     `json:"from_user_id"`
	ToUserID   string     `json:"to_user_id"`
	Status     string     `json:"status"`
	CreatedAt  time.Time  `json:"created_at"`
	Responded  *time.Time `json:"responded_at"`
}

type chatRoom struct {
	ID            int64      `json:"id"`
	UserAID       string     `json:"user_a_id"`
	UserBID       string     `json:"user_b_id"`
	CreatedAt     time.Time  `json:"created_at"`
	LastMessageAt *time.Time `json:"last_message_at"`
}

type chatMessage struct {
	ID           int64      `json:"id"`
	ChatID       int64      `json:"chat_id"`
	SenderUserID string     `json:"sender_user_id"`
	Content      string     `json:"content"`
	TaskID       *uuid.UUID `json:"task_id,omitempty"`
	PatientID    *uuid.UUID `json:"patient_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	ReadAt       *time.Time `json:"read_at"`
}

type sendInviteRequest struct {
	ToUserID string `json:"to_user_id"`
}

type sendMessageRequest struct {
	Content string `json:"content"`
	TaskID  string `json:"task_id,omitempty"`
}

type chatListItem struct {
	ID            int64      `json:"id"`
	OtherUserID   string     `json:"other_user_id"`
	LastMessage   string     `json:"last_message"`
	LastMessageAt *time.Time `json:"last_message_at"`
}

func getRequestUserID(r *http.Request) (string, error) {
	headerUserID := strings.TrimSpace(r.Header.Get("User-ID"))
	if headerUserID != "" {
		if !isValidSixDigitUserID(headerUserID) {
			return "", errors.New("invalid User-ID header")
		}
		// In handler-authorized mode, prioritize explicit User-ID context.
		return headerUserID, nil
	}

	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if authHeader == "" {
		return "", errors.New("missing auth context")
	}

	var tokenUserID string
	if authHeader != "" {
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return "", errors.New("invalid authorization header")
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			return "", errors.New("missing bearer token")
		}

		jwtSecret := []byte(os.Getenv("JWT_SECRET"))

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return jwtSecret, nil
		})
		if err != nil || !token.Valid {
			return "", errors.New("invalid token")
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return "", errors.New("invalid token claims")
		}

		userID, ok := claims["userId"].(string)
		if !ok || !isValidSixDigitUserID(userID) {
			return "", errors.New("missing userId claim")
		}

		tokenUserID = strings.TrimSpace(userID)
	}

	return tokenUserID, nil
}

func readJSON(r *http.Request, out interface{}) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(out)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func isValidSixDigitUserID(value string) bool {
	return sixDigitUserIDRegex.MatchString(strings.TrimSpace(value))
}

func (h *ChatHandler) userExistsByUserID(userID string) (bool, error) {
	var users []struct {
		UserID string `json:"userId"`
	}
	err := h.Supabase.DB.From("users").Select("userId").Eq("userId", userID).Execute(&users)
	if err != nil {
		return false, err
	}
	return len(users) > 0, nil
}

func (h *ChatHandler) getChatByPair(userA, userB string) (*chatRoom, error) {
	var direct []chatRoom
	if err := h.Supabase.DB.From("chats").Select("*").Eq("user_a_id", userA).Eq("user_b_id", userB).Execute(&direct); err != nil {
		return nil, err
	}
	if len(direct) > 0 {
		return &direct[0], nil
	}

	var reverse []chatRoom
	if err := h.Supabase.DB.From("chats").Select("*").Eq("user_a_id", userB).Eq("user_b_id", userA).Execute(&reverse); err != nil {
		return nil, err
	}
	if len(reverse) > 0 {
		return &reverse[0], nil
	}

	return nil, nil
}

func (h *ChatHandler) getUserUUIDBySixDigitUserID(userID string) (*uuid.UUID, error) {
	var users []struct {
		ID uuid.UUID `json:"id"`
	}
	err := h.Supabase.DB.From("users").
		Select("id").
		Eq("userId", userID).
		Execute(&users)
	if err != nil {
		return nil, err
	}
	if len(users) == 0 {
		return nil, nil
	}
	return &users[0].ID, nil
}

func (h *ChatHandler) getValidatedPatientQuestionAttachment(taskID string, senderSixDigitID string) (*uuid.UUID, *uuid.UUID, error) {
	parsedTaskID, err := uuid.Parse(strings.TrimSpace(taskID))
	if err != nil {
		return nil, nil, errors.New("invalid task_id")
	}

	senderUUID, err := h.getUserUUIDBySixDigitUserID(senderSixDigitID)
	if err != nil {
		return nil, nil, err
	}
	if senderUUID == nil {
		return nil, nil, errors.New("sender not found")
	}

	var tasks []struct {
		ID        uuid.UUID  `json:"id"`
		UserID    uuid.UUID  `json:"user_id"`
		TaskType  string     `json:"task_type"`
		PatientID *uuid.UUID `json:"patient_id"`
	}
	err = h.Supabase.DB.From("tasks").
		Select("id,user_id,task_type,patient_id").
		Eq("id", parsedTaskID.String()).
		Eq("user_id", senderUUID.String()).
		Eq("task_type", "patient_question").
		Execute(&tasks)
	if err != nil {
		return nil, nil, err
	}
	if len(tasks) == 0 {
		return nil, nil, errors.New("task_id is not an attachable patient_question task")
	}
	if tasks[0].PatientID == nil {
		return nil, nil, errors.New("task has no patient_id")
	}

	return &tasks[0].ID, tasks[0].PatientID, nil
}

func (h *ChatHandler) SendInvite(w http.ResponseWriter, r *http.Request) {
	fromUserID, err := getRequestUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var req sendInviteRequest
	if err := readJSON(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.ToUserID = strings.TrimSpace(req.ToUserID)
	if !isValidSixDigitUserID(req.ToUserID) {
		http.Error(w, "to_user_id must be a 6-digit userId", http.StatusBadRequest)
		return
	}
	if fromUserID == req.ToUserID {
		http.Error(w, "cannot invite yourself", http.StatusBadRequest)
		return
	}

	exists, err := h.userExistsByUserID(req.ToUserID)
	if err != nil {
		http.Error(w, "failed to validate target user", http.StatusInternalServerError)
		return
	}
	if !exists {
		http.Error(w, "target user does not exist", http.StatusNotFound)
		return
	}

	existingChat, err := h.getChatByPair(fromUserID, req.ToUserID)
	if err != nil {
		http.Error(w, "failed to check existing chats", http.StatusInternalServerError)
		return
	}
	if existingChat != nil {
		http.Error(w, "chat already exists for this pair", http.StatusConflict)
		return
	}

	var pendingOutgoing []chatInvite
	if err := h.Supabase.DB.From("chat_invites").
		Select("*").
		Eq("from_user_id", fromUserID).
		Eq("to_user_id", req.ToUserID).
		Eq("status", "pending").
		Execute(&pendingOutgoing); err != nil {
		http.Error(w, "failed to validate existing invites", http.StatusInternalServerError)
		return
	}
	if len(pendingOutgoing) > 0 {
		http.Error(w, "pending invite already exists", http.StatusConflict)
		return
	}

	insertData := map[string]interface{}{
		"from_user_id": fromUserID,
		"to_user_id":   req.ToUserID,
		"status":       "pending",
	}
	if err := h.Supabase.DB.From("chat_invites").Insert(insertData).Execute(nil); err != nil {
		http.Error(w, "failed to create invite", http.StatusInternalServerError)
		return
	}

	chatRealtimeHub.emitToUsers([]string{fromUserID, req.ToUserID}, wsEvent{
		Type: "invite_updated",
		Data: map[string]interface{}{
			"from_user_id": fromUserID,
			"to_user_id":   req.ToUserID,
			"status":       "pending",
		},
	})

	writeJSON(w, http.StatusCreated, map[string]string{"message": "invite created"})
}

func (h *ChatHandler) GetInvites(w http.ResponseWriter, r *http.Request) {
	userID, err := getRequestUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	status := strings.TrimSpace(r.URL.Query().Get("status"))
	if status == "" {
		status = "pending"
	}

	var incoming []chatInvite
	queryIncoming := h.Supabase.DB.From("chat_invites").Select("*").Eq("to_user_id", userID)
	if status != "all" {
		queryIncoming = queryIncoming.Eq("status", status)
	}
	if err := queryIncoming.Execute(&incoming); err != nil {
		http.Error(w, "failed to fetch incoming invites", http.StatusInternalServerError)
		return
	}

	var outgoing []chatInvite
	queryOutgoing := h.Supabase.DB.From("chat_invites").Select("*").Eq("from_user_id", userID)
	if status != "all" {
		queryOutgoing = queryOutgoing.Eq("status", status)
	}
	if err := queryOutgoing.Execute(&outgoing); err != nil {
		http.Error(w, "failed to fetch outgoing invites", http.StatusInternalServerError)
		return
	}

	sort.Slice(incoming, func(i, j int) bool {
		return incoming[i].CreatedAt.After(incoming[j].CreatedAt)
	})
	sort.Slice(outgoing, func(i, j int) bool {
		return outgoing[i].CreatedAt.After(outgoing[j].CreatedAt)
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"incoming": incoming,
		"outgoing": outgoing,
	})
}

func (h *ChatHandler) AcceptInvite(w http.ResponseWriter, r *http.Request) {
	userID, err := getRequestUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	inviteID := mux.Vars(r)["id"]
	var invites []chatInvite
	if err := h.Supabase.DB.From("chat_invites").Select("*").Eq("id", inviteID).Execute(&invites); err != nil {
		http.Error(w, "failed to fetch invite", http.StatusInternalServerError)
		return
	}
	if len(invites) == 0 {
		http.Error(w, "invite not found", http.StatusNotFound)
		return
	}

	invite := invites[0]
	if invite.ToUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if invite.Status != "pending" {
		http.Error(w, "invite is not pending", http.StatusConflict)
		return
	}

	now := time.Now()
	updateData := map[string]interface{}{
		"status":       "accepted",
		"responded_at": now,
	}
	if err := h.Supabase.DB.From("chat_invites").Update(updateData).Eq("id", inviteID).Execute(nil); err != nil {
		http.Error(w, "failed to update invite", http.StatusInternalServerError)
		return
	}

	chat, err := h.getChatByPair(invite.FromUserID, invite.ToUserID)
	if err != nil {
		http.Error(w, "failed to validate chat pair", http.StatusInternalServerError)
		return
	}
	if chat == nil {
		userA := invite.FromUserID
		userB := invite.ToUserID
		if strings.Compare(userA, userB) > 0 {
			userA, userB = userB, userA
		}
		insertData := map[string]interface{}{
			"user_a_id": userA,
			"user_b_id": userB,
		}
		if err := h.Supabase.DB.From("chats").Insert(insertData).Execute(nil); err != nil {
			http.Error(w, "failed to create chat", http.StatusInternalServerError)
			return
		}

		chat, err = h.getChatByPair(invite.FromUserID, invite.ToUserID)
		if err != nil || chat == nil {
			http.Error(w, "failed to read created chat", http.StatusInternalServerError)
			return
		}
	}

	chatRealtimeHub.emitToUsers([]string{invite.FromUserID, invite.ToUserID}, wsEvent{
		Type:   "invite_updated",
		ChatID: chat.ID,
		Data: map[string]interface{}{
			"status": "accepted",
		},
	})
	chatRealtimeHub.emitToUsers([]string{invite.FromUserID, invite.ToUserID}, wsEvent{
		Type:   "chat_updated",
		ChatID: chat.ID,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"chat_id": chat.ID,
		"status":  "accepted",
	})
}

func (h *ChatHandler) DeclineInvite(w http.ResponseWriter, r *http.Request) {
	userID, err := getRequestUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	inviteID := mux.Vars(r)["id"]
	var invites []chatInvite
	if err := h.Supabase.DB.From("chat_invites").Select("*").Eq("id", inviteID).Execute(&invites); err != nil {
		http.Error(w, "failed to fetch invite", http.StatusInternalServerError)
		return
	}
	if len(invites) == 0 {
		http.Error(w, "invite not found", http.StatusNotFound)
		return
	}

	invite := invites[0]
	if invite.ToUserID != userID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if invite.Status != "pending" {
		http.Error(w, "invite is not pending", http.StatusConflict)
		return
	}

	updateData := map[string]interface{}{
		"status":       "declined",
		"responded_at": time.Now(),
	}
	if err := h.Supabase.DB.From("chat_invites").Update(updateData).Eq("id", inviteID).Execute(nil); err != nil {
		http.Error(w, "failed to update invite", http.StatusInternalServerError)
		return
	}

	chatRealtimeHub.emitToUsers([]string{invite.FromUserID, invite.ToUserID}, wsEvent{
		Type: "invite_updated",
		Data: map[string]interface{}{
			"status": "declined",
		},
	})

	writeJSON(w, http.StatusOK, map[string]string{"status": "declined"})
}

func (h *ChatHandler) GetChats(w http.ResponseWriter, r *http.Request) {
	userID, err := getRequestUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	var chatsA []chatRoom
	if err := h.Supabase.DB.From("chats").Select("*").Eq("user_a_id", userID).Execute(&chatsA); err != nil {
		http.Error(w, "failed to fetch chats", http.StatusInternalServerError)
		return
	}
	var chatsB []chatRoom
	if err := h.Supabase.DB.From("chats").Select("*").Eq("user_b_id", userID).Execute(&chatsB); err != nil {
		http.Error(w, "failed to fetch chats", http.StatusInternalServerError)
		return
	}

	combined := make([]chatRoom, 0, len(chatsA)+len(chatsB))
	combined = append(combined, chatsA...)
	combined = append(combined, chatsB...)

	items := make([]chatListItem, 0, len(combined))
	for _, c := range combined {
		otherUserID := c.UserAID
		if c.UserAID == userID {
			otherUserID = c.UserBID
		}

		lastMessageText := ""
		lastMessageAt := c.LastMessageAt
		var messages []chatMessage
		err := h.Supabase.DB.From("chat_messages").Select("*").Eq("chat_id", strconv.FormatInt(c.ID, 10)).Execute(&messages)
		if err == nil && len(messages) > 0 {
			sort.Slice(messages, func(i, j int) bool {
				return messages[i].CreatedAt.After(messages[j].CreatedAt)
			})
			lastMessageText = messages[0].Content
			if lastMessageAt == nil {
				lastMessageAt = &messages[0].CreatedAt
			}
		}

		items = append(items, chatListItem{
			ID:            c.ID,
			OtherUserID:   otherUserID,
			LastMessage:   lastMessageText,
			LastMessageAt: lastMessageAt,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].LastMessageAt == nil {
			return false
		}
		if items[j].LastMessageAt == nil {
			return true
		}
		return items[i].LastMessageAt.After(*items[j].LastMessageAt)
	})

	writeJSON(w, http.StatusOK, items)
}

func (h *ChatHandler) getChatForUser(chatID string, userID string) (*chatRoom, error) {
	var chats []chatRoom
	err := h.Supabase.DB.From("chats").Select("*").Eq("id", chatID).Execute(&chats)
	if err != nil {
		return nil, err
	}
	if len(chats) == 0 {
		return nil, nil
	}
	chat := chats[0]
	if chat.UserAID != userID && chat.UserBID != userID {
		return nil, errors.New("forbidden")
	}
	return &chat, nil
}

func (h *ChatHandler) GetChatMessages(w http.ResponseWriter, r *http.Request) {
	userID, err := getRequestUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := mux.Vars(r)["chat_id"]
	chat, err := h.getChatForUser(chatID, userID)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if chat == nil {
		http.Error(w, "chat not found", http.StatusNotFound)
		return
	}

	limit := 50
	if limitQuery := r.URL.Query().Get("limit"); strings.TrimSpace(limitQuery) != "" {
		parsed, parseErr := strconv.Atoi(limitQuery)
		if parseErr == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	var messages []chatMessage
	if err := h.Supabase.DB.From("chat_messages").
		Select("*").
		Eq("chat_id", chatID).
		Execute(&messages); err != nil {
		http.Error(w, "failed to fetch messages", http.StatusInternalServerError)
		return
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].CreatedAt.Before(messages[j].CreatedAt)
	})
	if len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}

	writeJSON(w, http.StatusOK, messages)
}

func (h *ChatHandler) SendMessage(w http.ResponseWriter, r *http.Request) {
	userID, err := getRequestUserID(r)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	chatID := mux.Vars(r)["chat_id"]
	chat, err := h.getChatForUser(chatID, userID)
	if err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if chat == nil {
		http.Error(w, "chat not found", http.StatusNotFound)
		return
	}

	var req sendMessageRequest
	if err := readJSON(r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	content := strings.TrimSpace(req.Content)
	if content == "" {
		http.Error(w, "content cannot be empty", http.StatusBadRequest)
		return
	}
	if len(content) > 4000 {
		http.Error(w, "content exceeds 4000 characters", http.StatusBadRequest)
		return
	}

	attachmentTaskID := strings.TrimSpace(req.TaskID)
	var attachedTaskID *uuid.UUID
	var attachedPatientID *uuid.UUID
	if attachmentTaskID != "" {
		validTaskID, validPatientID, validateErr := h.getValidatedPatientQuestionAttachment(attachmentTaskID, userID)
		if validateErr != nil {
			http.Error(w, validateErr.Error(), http.StatusForbidden)
			return
		}
		attachedTaskID = validTaskID
		attachedPatientID = validPatientID
	}

	now := time.Now()
	insertData := map[string]interface{}{
		"chat_id":        chat.ID,
		"sender_user_id": userID,
		"content":        content,
		"created_at":     now,
	}
	if attachedTaskID != nil {
		insertData["task_id"] = attachedTaskID.String()
	}
	if attachedPatientID != nil {
		insertData["patient_id"] = attachedPatientID.String()
	}
	if err := h.Supabase.DB.From("chat_messages").Insert(insertData).Execute(nil); err != nil {
		http.Error(w, "failed to send message", http.StatusInternalServerError)
		return
	}

	if err := h.Supabase.DB.From("chats").Update(map[string]interface{}{"last_message_at": now}).Eq("id", chatID).Execute(nil); err != nil {
		http.Error(w, "failed to update chat metadata", http.StatusInternalServerError)
		return
	}

	chatRealtimeHub.emitToUsers([]string{chat.UserAID, chat.UserBID}, wsEvent{
		Type:   "message_created",
		ChatID: chat.ID,
		Data: map[string]interface{}{
			"sender_user_id": userID,
			"content":        content,
			"created_at":     now,
			"task_id":        attachedTaskID,
			"patient_id":     attachedPatientID,
		},
	})
	chatRealtimeHub.emitToUsers([]string{chat.UserAID, chat.UserBID}, wsEvent{
		Type:   "chat_updated",
		ChatID: chat.ID,
	})

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"chat_id":        chat.ID,
		"sender_user_id": userID,
		"content":        content,
		"created_at":     now,
		"task_id":        attachedTaskID,
		"patient_id":     attachedPatientID,
	})
}
