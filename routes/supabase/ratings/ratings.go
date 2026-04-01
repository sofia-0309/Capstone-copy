package ratings

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gorilla/mux"
	supabase "github.com/nedpals/supabase-go"
	model "gitlab.msu.edu/team-corewell-2025/models"
)

type RatingHandler struct {
	Supabase *supabase.Client
}

type RatingService interface {
	SubmitRating(w http.ResponseWriter, r *http.Request)
	GetUserRatings(w http.ResponseWriter, r *http.Request)
}

// SubmitRating saves a new rating to the database
func (h *RatingHandler) SubmitRating(w http.ResponseWriter, r *http.Request) {
	var req struct {
		UserId    string `json:"user_id"`
		TaskId    string `json:"task_id"`
		PatientId string `json:"patient_id"`
		TaskType  string `json:"task_type"`
		Rating    int    `json:"rating"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate rating is 1-5
	if req.Rating < 1 || req.Rating > 5 {
		http.Error(w, "Rating must be between 1 and 5", http.StatusBadRequest)
		return
	}

	// Insert rating into database
	insertData := map[string]interface{}{
		"user_id":    req.UserId,
		"task_id":    req.TaskId,
		"patient_id": req.PatientId,
		"task_type":  req.TaskType,
		"rating":     req.Rating,
	}

	err := h.Supabase.DB.From("ratings").Insert(insertData).Execute(nil)
	if err != nil {
		fmt.Println("Error inserting rating:", err)
		http.Error(w, "Failed to save rating", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "Rating saved successfully"})
}

// GetUserRatings fetches all ratings for a user and calculates statistics
func (h *RatingHandler) GetUserRatings(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	userId := vars["user_id"]

	// Fetch all ratings for this user
	var ratings []model.Rating
	err := h.Supabase.DB.From("ratings").
		Select("*").
		Eq("user_id", userId).
		Execute(&ratings)

	if err != nil {
		fmt.Println("Error fetching ratings:", err)
		http.Error(w, "Failed to fetch ratings", http.StatusInternalServerError)
		return
	}

	// Sort ratings by created_at descending (newest first) in Go
	sort.Slice(ratings, func(i, j int) bool {
		return ratings[i].CreatedAt.After(ratings[j].CreatedAt)
	})

	// Calculate statistics
	stats := calculateRatingStats(ratings, h.Supabase)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func calculateRatingStats(ratings []model.Rating, supabase *supabase.Client) model.RatingStats {
	stats := model.RatingStats{
		RatingDistribution: [5]int{0, 0, 0, 0, 0},
		RatingsByTaskType:  make(map[string]model.TaskTypeRating),
		RecentRatings:      []model.RecentRating{},
	}

	if len(ratings) == 0 {
		return stats
	}

	stats.TotalRatings = len(ratings)

	// Calculate totals by task type
	taskTypeTotals := make(map[string]int)
	taskTypeCounts := make(map[string]int)
	totalRating := 0

	for _, rating := range ratings {
		// Distribution (array is 0-indexed, so rating 1 goes to index 0)
		stats.RatingDistribution[rating.Rating-1]++

		// Total for average
		totalRating += rating.Rating

		// By task type
		taskTypeTotals[rating.TaskType] += rating.Rating
		taskTypeCounts[rating.TaskType]++
	}

	// Calculate average
	stats.AverageRating = float64(totalRating) / float64(stats.TotalRatings)

	// Calculate averages by task type
	for taskType, total := range taskTypeTotals {
		count := taskTypeCounts[taskType]
		stats.RatingsByTaskType[taskType] = model.TaskTypeRating{
			Average: float64(total) / float64(count),
			Count:   count,
		}
	}

	// Get recent ratings (up to 10)
	recentCount := 10
	if len(ratings) < 10 {
		recentCount = len(ratings)
	}

	for i := 0; i < recentCount; i++ {
		rating := ratings[i]

		// Fetch patient name
		patientName := "Unknown Patient"
		if rating.PatientId != nil {
			var patients []struct {
				Name string `json:"name"`
			}
			err := supabase.DB.From("patients").
				Select("name").
				Eq("id", rating.PatientId.String()).
				Execute(&patients)

			if err == nil && len(patients) > 0 {
				patientName = patients[0].Name
			}
		}

		// Format timestamp
		timestamp := formatTimestamp(rating.CreatedAt)

		stats.RecentRatings = append(stats.RecentRatings, model.RecentRating{
			TaskType:    rating.TaskType,
			Rating:      rating.Rating,
			Timestamp:   timestamp,
			PatientName: patientName,
		})
	}

	return stats
}

func formatTimestamp(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < time.Minute {
		return "Just now"
	} else if diff < time.Hour {
		minutes := int(diff.Minutes())
		if minutes == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", minutes)
	} else if diff < 24*time.Hour {
		hours := int(diff.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	} else if diff < 48*time.Hour {
		return "1 day ago"
	} else if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d days ago", days)
	} else {
		return t.Format("Jan 2, 2006")
	}
}
