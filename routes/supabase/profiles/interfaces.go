package profiles

import "net/http"

type ProfilesService interface {
	GetProfile(w http.ResponseWriter, r *http.Request)
	UpdateProfile(w http.ResponseWriter, r *http.Request)
	UpdateLastActive(w http.ResponseWriter, r *http.Request)
	UpdateFeedback(w http.ResponseWriter, r *http.Request)
	GetFeedback(w http.ResponseWriter, r *http.Request)
	SaveRatings(w http.ResponseWriter, r *http.Request)
	GetRatings(w http.ResponseWriter, r *http.Request)
	GetRatingsByType(w http.ResponseWriter, r *http.Request)
	GetAllRatings(w http.ResponseWriter, r *http.Request)
	GetLeastTags(w http.ResponseWriter, r *http.Request)
	GetTagsStats(w http.ResponseWriter, r *http.Request)
}

