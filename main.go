package main

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"

	"knife/api"
	"knife/base"
	"knife/db"
)

func main() {
	if args := len(os.Args); args > 1 && os.Args[1] == "init" {
		initializes()
		return
	}

	// --- Generate or Load Secret Key ---
	secretKey := getOrCreateSecretKey("secret.key")

	// --- Database Initialization ---
	dbconn, err := db.InitDB("knife.db")
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer dbconn.Close()

	profileModel := db.NewProfileModel(dbconn)
	profileApi := api.NewProfileAPI(profileModel)
	noteModel := db.NewNoteModel(dbconn)
	noteAPI := api.NewNoteAPI(noteModel, profileModel)
	bookmarkModel := db.NewBookmarkModel(dbconn)
	bookmarkApi := api.NewBookmarkAPI(bookmarkModel, noteModel)
	activityPubApi := api.NewActivityPubAPI(noteModel, profileModel)

	// --- Authentication API ---
	authAPI := api.NewAuthAPI(profileModel, secretKey)
	authMiddleware := api.NewAuthMiddleware(authAPI)

	// --- API Routing ---
	apiRouter := base.NewAPIRouter()
	authAPI.RegisterHandlers(&apiRouter)
	profileApi.RegisterHandlers(&apiRouter)
	noteAPI.RegisterHandlers(&apiRouter)
	bookmarkApi.RegisterHandlers(&apiRouter)
	activityPubApi.RegisterHandlers(&apiRouter)

	// Apply authentication middleware to protected routes
	apiRouter.RegisterMidddleware(authMiddleware)

	// --- Main Server Routing ---
	mainMux := http.NewServeMux()
	mainMux.Handle("/api/", http.StripPrefix("/api", apiRouter.GetMUX()))

	// --- WebFinger ---
	mainMux.HandleFunc("/.well-known/webfinger", activityPubApi.Webfinger)

	// Static file handling
	staticFS, err := fs.Sub(Content, "frontend/static")
	if err != nil {
		panic(err)
	}
	mainMux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Serve HTML files
	mainMux.HandleFunc("/", serveFile("frontend/index.html"))
	mainMux.HandleFunc("/new-note", serveFile("frontend/new-note.html"))
	mainMux.HandleFunc("/profile", serveFile("frontend/profile.html"))
	mainMux.HandleFunc("/profile-settings", serveFile("frontend/profile-settings.html"))
	mainMux.HandleFunc("/bookmarks", serveFile("frontend/bookmarks.html"))
	mainMux.HandleFunc("/login", serveFile("frontend/login.html"))

	// --- Start Server ---
	log.Println("Server starting on :8080")
	log.Println("Access the frontend at http://localhost:8080")
	if err := http.ListenAndServe(":8080", mainMux); err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}

func serveFile(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, err := Content.Open(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		io.Copy(w, file)
	}
}

// getOrCreateSecretKey generates or loads a secret key from a file.
func getOrCreateSecretKey(filename string) string {
	// Check if the key file exists
	if _, err := os.Stat(filename); err == nil {
		// Load the key from the file
		key, err := os.ReadFile(filename)
		if err != nil {
			log.Fatalf("could not read secret key file: %v", err)
		}
		return string(key)
	}

	// Generate a new random key
	key := make([]byte, 32) // 256-bit key
	if _, err := rand.Read(key); err != nil {
		log.Fatalf("could not generate secret key: %v", err)
	}

	// Save the key to the file
	if err := os.WriteFile(filename, []byte(hex.EncodeToString(key)), 0600); err != nil {
		log.Fatalf("could not write secret key to file: %v", err)
	}

	log.Println("Generated new secret key and saved to", filename)
	return hex.EncodeToString(key)
}
