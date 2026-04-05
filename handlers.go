package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
)

// HealthHandler returns service health status
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"service": "character-service",
	})
}

// MoveRequest represents a character movement request
type MoveRequest struct {
	CharacterID string  `json:"character_id"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
}

// MoveHandler moves a character to a new position
func MoveHandler(w http.ResponseWriter, r *http.Request) {
	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CharacterID == "" {
		http.Error(w, "character_id is required", http.StatusBadRequest)
		return
	}

	// Get or create character
	char, err := store.GetCharacter(req.CharacterID)
	if err != nil {
		// Character doesn't exist, create new one
		char = &Character{
			ID:       req.CharacterID,
			Name:     fmt.Sprintf("Character-%s", req.CharacterID),
			X:        req.X,
			Y:        req.Y,
			State:    "waiting",
			CanCross: false,
		}
	} else {
		// Update position
		char.X = req.X
		char.Y = req.Y
	}

	// Save character
	if err := store.SaveCharacter(char); err != nil {
		log.Printf("Error saving character: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(char)
}

// GetPositionHandler returns a character's current position
func GetPositionHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	characterID := vars["characterId"]

	if characterID == "" {
		http.Error(w, "characterId is required", http.StatusBadRequest)
		return
	}

	char, err := store.GetCharacter(characterID)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"character_id": char.ID,
		"x":            char.X,
		"y":            char.Y,
	})
}

// CrossRequest represents a crossing attempt request
type CrossRequest struct {
	CharacterID string `json:"character_id"`
}

// SeaStatusResponse represents the sea state service response
type SeaStatusResponse struct {
	RedSea string `json:"red_sea"` // "closed", "splitting", "split"
	Status string `json:"status"`
}

// CrossHandler initiates a crossing attempt
func CrossHandler(w http.ResponseWriter, r *http.Request) {
	var req CrossRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.CharacterID == "" {
		http.Error(w, "character_id is required", http.StatusBadRequest)
		return
	}

	// Get character
	char, err := store.GetCharacter(req.CharacterID)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	// Check sea state
	seaStateURL := fmt.Sprintf("%s/status", seaStateServiceURL)
	resp, err := http.Get(seaStateURL)
	if err != nil {
		log.Printf("Error checking sea state: %v", err)
		http.Error(w, "Unable to check sea state", http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	var seaStatus SeaStatusResponse
	if err := json.NewDecoder(resp.Body).Decode(&seaStatus); err != nil {
		log.Printf("Error decoding sea state response: %v", err)
		http.Error(w, "Invalid sea state response", http.StatusInternalServerError)
		return
	}

	// Check if sea is split
	canCross := seaStatus.RedSea == "split"
	char.CanCross = canCross

	if canCross {
		char.State = "crossing"
	} else {
		char.State = "waiting"
	}

	// Save updated character state
	if err := store.SaveCharacter(char); err != nil {
		log.Printf("Error saving character: %v", err)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"character_id": char.ID,
		"can_cross":    canCross,
		"sea_state":    seaStatus.RedSea,
		"message":      getMessageForState(canCross),
	})
}

// GetStatusHandler returns a character's full status
func GetStatusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	characterID := vars["characterId"]

	if characterID == "" {
		http.Error(w, "characterId is required", http.StatusBadRequest)
		return
	}

	char, err := store.GetCharacter(characterID)
	if err != nil {
		http.Error(w, "Character not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(char)
}

func getMessageForState(canCross bool) string {
	if canCross {
		return "The sea is split! You may cross safely."
	}
	return "The sea is closed. You cannot cross yet."
}
