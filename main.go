package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Character represents a character in the game
type Character struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	X        float64 `json:"x"`
	Y        float64 `json:"y"`
	State    string  `json:"state"` // "waiting", "crossing", "crossed"
	CanCross bool    `json:"can_cross"`
}

// CharacterStore manages character state
type CharacterStore struct {
	mu         sync.RWMutex
	characters map[string]*Character
	db         *sql.DB
	useDB      bool
}

// SeaStateService endpoint
const seaStateServiceURL = "http://sea-state-service:8080"

var store *CharacterStore

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	// Initialize store
	store = NewCharacterStore()
	defer func() {
		if store.db != nil {
			store.db.Close()
		}
	}()

	// Setup router
	r := mux.NewRouter()
	r.HandleFunc("/health", HealthHandler).Methods("GET")
	r.HandleFunc("/move", MoveHandler).Methods("POST")
	r.HandleFunc("/position/{characterId}", GetPositionHandler).Methods("GET")
	r.HandleFunc("/cross", CrossHandler).Methods("POST")
	r.HandleFunc("/status/{characterId}", GetStatusHandler).Methods("GET")

	// Start server
	log.Printf("Character service starting on port %s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatal(err)
	}
}

// NewCharacterStore creates a new character store
func NewCharacterStore() *CharacterStore {
	s := &CharacterStore{
		characters: make(map[string]*Character),
		useDB:      false,
	}

	// Try to connect to PostgreSQL if configured
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")

	if dbHost != "" && dbUser != "" {
		connStr := fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
			dbHost, dbUser, dbPassword, dbName)

		db, err := sql.Open("postgres", connStr)
		if err != nil {
			log.Printf("Warning: Could not connect to database: %v. Using in-memory storage.", err)
		} else if err := db.Ping(); err != nil {
			log.Printf("Warning: Could not ping database: %v. Using in-memory storage.", err)
		} else {
			s.db = db
			s.useDB = true
			log.Println("Connected to PostgreSQL database")
			s.initDB()
		}
	} else {
		log.Println("Database not configured. Using in-memory storage.")
	}

	return s
}

// initDB initializes the database schema
func (s *CharacterStore) initDB() {
	schema := `
	CREATE TABLE IF NOT EXISTS characters (
		id VARCHAR(255) PRIMARY KEY,
		name VARCHAR(255) NOT NULL,
		x FLOAT NOT NULL,
		y FLOAT NOT NULL,
		state VARCHAR(50) NOT NULL,
		can_cross BOOLEAN NOT NULL
	)`

	_, err := s.db.Exec(schema)
	if err != nil {
		log.Printf("Warning: Could not create schema: %v", err)
	}
}

// GetCharacter retrieves a character by ID
func (s *CharacterStore) GetCharacter(id string) (*Character, error) {
	if s.useDB {
		return s.getCharacterFromDB(id)
	}
	return s.getCharacterFromMemory(id)
}

// SaveCharacter saves a character
func (s *CharacterStore) SaveCharacter(char *Character) error {
	if s.useDB {
		return s.saveCharacterToDB(char)
	}
	return s.saveCharacterToMemory(char)
}

// Memory operations
func (s *CharacterStore) getCharacterFromMemory(id string) (*Character, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if char, exists := s.characters[id]; exists {
		return char, nil
	}
	return nil, fmt.Errorf("character not found: %s", id)
}

func (s *CharacterStore) saveCharacterToMemory(char *Character) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.characters[char.ID] = char
	return nil
}

// Database operations
func (s *CharacterStore) getCharacterFromDB(id string) (*Character, error) {
	var char Character
	err := s.db.QueryRow(
		"SELECT id, name, x, y, state, can_cross FROM characters WHERE id = $1",
		id,
	).Scan(&char.ID, &char.Name, &char.X, &char.Y, &char.State, &char.CanCross)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("character not found: %s", id)
	}
	if err != nil {
		return nil, err
	}

	return &char, nil
}

func (s *CharacterStore) saveCharacterToDB(char *Character) error {
	_, err := s.db.Exec(
		`INSERT INTO characters (id, name, x, y, state, can_cross)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (id) DO UPDATE SET
		 name = $2, x = $3, y = $4, state = $5, can_cross = $6`,
		char.ID, char.Name, char.X, char.Y, char.State, char.CanCross,
	)
	return err
}
