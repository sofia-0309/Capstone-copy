package llm

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gorilla/mux"
)

// PostLLMResponseForPatient is the new route that forwards the entire GIGA JSON to app.py
func PostLLMResponseForPatient(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"] // optional if you want to log or pass it for debugging

	// Read the raw JSON from the request body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Cannot read POST body", http.StatusBadRequest)
		return
	}

	fmt.Println("Received GIGA JSON for patient:", id)
	fmt.Println("Full JSON body:", string(bodyBytes)) // optional debug print

	// Forward the entire JSON payload to your Python microservice
	flaskURL := os.Getenv("FLASK_EXPLAIN_URL")
	if flaskURL == "" {
		flaskURL = "http://127.0.0.1:5001/api/explain-request" // default for local dev
	}
	resp, err := http.Post(flaskURL, "application/json", bytes.NewBuffer(bodyBytes))
	if err != nil {
		fmt.Println("Error forwarding JSON to Flask microservice:", err)
		http.Error(w, "Failed to contact LLM microservice", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the LLM’s response from Python
	responseBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading LLM response:", err)
		http.Error(w, "Failed to read LLM response", http.StatusInternalServerError)
		return
	}

	// Return that response to the frontend
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(responseBytes)
}

// GetPatientImage proxies a GET request to the Flask microservice to retrieve a patient's image URL
func GetPatientImage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	flaskURL := os.Getenv("FLASK_IMAGE_URL")
	if flaskURL == "" {
		flaskURL = fmt.Sprintf("http://127.0.0.1:5001/patients/%s/dermnet_image", id)
	} else {
		flaskURL = fmt.Sprintf("%s/patients/%s/dermnet_image", flaskURL, id)
	}

	fmt.Println("Proxying dermnet image request to Flask:", flaskURL)
	resp, err := http.Get(flaskURL)
	if err != nil {
		fmt.Println("Error contacting Flask dermnet endpoint:", err)
		http.Error(w, "Failed to contact LLM image microservice", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}


// get flask url
func PostPatientConcerns(w http.ResponseWriter, r *http.Request) {
	flaskURL := os.Getenv("FLASK_URL")
	if flaskURL == "" {
		flaskURL = "http://127.0.0.1:5001/api/patient-concerns"
	} else {
		flaskURL = fmt.Sprintf("%s/api/patient-concerns", flaskURL)
	}

	fmt.Println("Proxying patient concerns request to Flask:", flaskURL)
	// log whats going on




	bod, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Println("Error reading patient concerns:", err)
		http.Error(w, "Cannot read body", http.StatusBadRequest)
		return
	}
	// read json from frontend
	resp, err := http.Post(flaskURL, "application/json", bytes.NewBuffer(bod))
	if err != nil {
		fmt.Println("Error contacting Flask patient concern endpoint:", err)
		http.Error(w, "Failed to contact LLM image microservice", http.StatusBadGateway)
		return
	}
	// send to flask
	defer resp.Body.Close()



	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
	// send flask back to the backend
}





// GetPatientProfilePicture proxies /patients/{id}/profile_picture to Flask
func GetPatientProfilePicture(w http.ResponseWriter, r *http.Request) {
	flaskURL := os.Getenv("FLASK_URL")
	if flaskURL == "" {
		flaskURL = "http://localhost:5001"
	}

	target := fmt.Sprintf("%s%s", flaskURL, r.URL.Path)
	fmt.Println("Proxying profile picture request to Flask:", target)

	resp, err := http.Get(target)
	if err != nil {
		http.Error(w, "Failed to reach Flask", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}
