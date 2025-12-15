package main

import (
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

	// --- Database Initialization ---
	dbconn, err := db.InitDB("knife.db")
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	defer dbconn.Close()

	profileModel := db.NewProfileModel(dbconn)
	profileAPI := api.NewProfileAPI(profileModel)

	// --- API Model and Handler Setup ---
	noteModel := db.NewNoteModel(dbconn)
	noteAPI := api.NewNoteAPI(noteModel, profileModel)

	bookmarkModel := db.NewBookmarkModel(dbconn)
	bookmarkAPI := api.NewBookmarkAPI(bookmarkModel)

	// --- API Routing ---
	// The API router will handle all routes under /api/
	apiRouter := base.NewAPIRouter()
	// No prefix is set here because the main mux will route /api/ to this router.
	// The APIRouter's pathMaker will create paths like "GET /notes", not "GET /api/notes"
	noteAPI.RegisterHandlers(&apiRouter)
	profileAPI.RegisterHandlers(&apiRouter)
	bookmarkAPI.RegisterHandlers(&apiRouter)

	// --- Main Server Routing ---
	mainMux := http.NewServeMux()

	// Route API calls to the API router
	mainMux.Handle("/api/", http.StripPrefix("/api", apiRouter.GetMUX()))

	// Create a http.FileServer to serve embedded files
	// The http.Dir("frontend") part in the original code pointed to the filesystem.
	// We need to strip "frontend" from the path when serving from embed.FS
	// because web.Content embeds the "frontend" directory directly.
	staticFS, err := fs.Sub(Content, "frontend/static")
	if err != nil {
		panic(err)
	}

	mainMux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Serve the main index.html for the root path
	mainMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		file, err := Content.Open("frontend/index.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		io.Copy(w, file)
	})

	mainMux.HandleFunc("/new-note", func(w http.ResponseWriter, r *http.Request) {
		file, err := Content.Open("frontend/new-note.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		io.Copy(w, file)
	})

	mainMux.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		file, err := Content.Open("frontend/profile.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		io.Copy(w, file)
	})

	mainMux.HandleFunc("/profile-settings", func(w http.ResponseWriter, r *http.Request) {
		file, err := Content.Open("frontend/profile-settings.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		io.Copy(w, file)
	})

	mainMux.HandleFunc("/bookmarks", func(w http.ResponseWriter, r *http.Request) {
		file, err := Content.Open("frontend/bookmarks.html")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		io.Copy(w, file)
	})

	// --- Start Server ---
	log.Println("Server starting on :8080")
	log.Println("Access the frontend at http://localhost:8080")
	if err := http.ListenAndServe(":8080", mainMux); err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
