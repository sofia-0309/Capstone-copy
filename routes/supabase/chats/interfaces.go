package chats

import "net/http"

type ChatService interface {
	SendInvite(w http.ResponseWriter, r *http.Request)
	GetInvites(w http.ResponseWriter, r *http.Request)
	AcceptInvite(w http.ResponseWriter, r *http.Request)
	DeclineInvite(w http.ResponseWriter, r *http.Request)
	GetChats(w http.ResponseWriter, r *http.Request)
	GetChatMessages(w http.ResponseWriter, r *http.Request)
	SendMessage(w http.ResponseWriter, r *http.Request)
	ChatEventsWS(w http.ResponseWriter, r *http.Request)
}
