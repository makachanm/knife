package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"

	"knife/ap"
	"knife/api"
	"knife/base"
	"knife/db"
	"knife/etc"
)

func main() {
	fmt.Println("Knife version ", etc.Version)
	if len(os.Args) > 1 && os.Args[1] == "setup" {
		initializes()
		return
	}

	// --- 초기화 ---
	dbconn := initializeDatabase("knife.db")
	defer dbconn.Close()
	log.Println("Database connected.")

	if len(os.Args) > 1 && os.Args[1] == "initkey" {
		getOrCreateSecretKey("secret.key")

		httpsigModel := db.NewHTTPSigModel(dbconn)
		profileModel := db.NewProfileModel(dbconn)

		profile, err := profileModel.Get()
		if err != nil {
			log.Fatalf("could not get profile: %v", err)
		}

		username := profile.Finger
		httpsigModel.Create(username)

		return
	}

	secretKey := initializeSecretKey("secret.key")
	jobQueue := initializeJobQueue()
	log.Println("Job queue started.")

	// --- 모델 및 API 초기화 ---
	profileModel := db.NewProfileModel(dbconn)
	noteModel := db.NewNoteModel(dbconn)
	followerModel := db.NewFollowerModel(dbconn)
	bookmarkModel := db.NewBookmarkModel(dbconn)
	httpsigModel := db.NewHTTPSigModel(dbconn)
	draftModel := db.NewDraftModel(dbconn)
	log.Println("Models initialized.")

	activityDispatcher := ap.NewActivityDispatcher(followerModel, httpsigModel, jobQueue)

	authAPI := api.NewAuthAPI(profileModel, secretKey)
	profileAPI := api.NewProfileAPI(profileModel, noteModel)
	noteAPI := api.NewNoteAPI(noteModel, profileModel, followerModel, activityDispatcher)
	bookmarkAPI := api.NewBookmarkAPI(bookmarkModel, noteModel)
	draftAPI := api.NewDraftAPI(draftModel)
	categoryAPI := api.NewCategoryAPI(noteModel)
	activityPubAPI := ap.NewActivityPubAPI(noteModel, profileModel, followerModel, httpsigModel)
	log.Println("APIs initialized.")

	// --- 라우터 설정 ---
	apiRouter := setupAPIRouter(authAPI, profileAPI, noteAPI, bookmarkAPI, draftAPI, categoryAPI)
	mainMux := setupMainRouter(apiRouter, activityPubAPI)
	log.Println("Router setup complete.")

	log.Println("Boot complete.")
	// --- 서버 시작 ---
	startServer(mainMux)
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

// --- 초기화 함수들 ---
func initializeDatabase(dbPath string) *db.DB {
	dbconn, err := db.InitDB(dbPath)
	if err != nil {
		log.Fatalf("could not connect to database: %v", err)
	}
	return dbconn
}

func initializeSecretKey(filename string) string {
	return getOrCreateSecretKey(filename)
}

func initializeJobQueue() *base.JobQueue {
	jobQueue := base.NewJobQueue(100)
	jobQueue.Start()
	return jobQueue
}

// --- 라우터 설정 함수 ---
func setupAPIRouter(authAPI *api.AuthAPI, profileAPI *api.ProfileAPI, noteAPI *api.NoteAPI, bookmarkAPI *api.BookmarkAPI, draftAPI *api.DraftAPI, categoryAPI *api.CategoryAPI) base.APIRouter {
	apiRouter := base.NewAPIRouter()
	authAPI.RegisterHandlers(&apiRouter)
	profileAPI.RegisterHandlers(&apiRouter)
	noteAPI.RegisterHandlers(&apiRouter)
	bookmarkAPI.RegisterHandlers(&apiRouter)
	draftAPI.RegisterHandlers(&apiRouter)
	categoryAPI.RegisterHandlers(&apiRouter)

	// Apply authentication middleware to protected routes
	apiRouter.RegisterMidddleware(api.NewAuthMiddleware(authAPI))

	return apiRouter
}

func setupMainRouter(apiRouter base.APIRouter, activityPubAPI *ap.ActivityPubAPI) *http.ServeMux {
	mainMux := http.NewServeMux()
	mainMux.Handle("/api/", http.StripPrefix("/api", apiRouter.GetMUX()))

	// --- WebFinger ---
	mainMux.HandleFunc("/.well-known/webfinger", activityPubAPI.Webfinger)

	// --- ActivityPub ---
	mainMux.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")
		if strings.Contains(acceptHeader, "application/activity+json") {
			activityPubAPI.Actor(w, r)
		} else {
			serveFile("frontend/profile.html")(w, r)
		}
	})
	mainMux.HandleFunc("/inbox", activityPubAPI.Inbox)
	mainMux.HandleFunc("/notes/", func(w http.ResponseWriter, r *http.Request) {
		acceptHeader := r.Header.Get("Accept")
		if strings.Contains(acceptHeader, "application/activity+json") {
			activityPubAPI.Note(w, r)
		} else {
			serveFile("frontend/note.html")(w, r)
		}
	})

	// Static file handling
	staticFS, err := fs.Sub(Content, "frontend/static")
	if err != nil {
		panic(err)
	}
	mainMux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Serve HTML files
	mainMux.HandleFunc("/", serveFile("frontend/index.html"))
	mainMux.HandleFunc("/categories", serveFile("frontend/categories.html"))
	mainMux.HandleFunc("/category/", serveFile("frontend/category.html"))
	mainMux.HandleFunc("/new-note", serveFile("frontend/new-note.html"))
	mainMux.HandleFunc("/profile-settings", serveFile("frontend/profile-settings.html"))
	mainMux.HandleFunc("/bookmarks", serveFile("frontend/bookmarks.html"))
	mainMux.HandleFunc("/login", serveFile("frontend/login.html"))

	return mainMux
}

// --- 서버 시작 함수 ---
func startServer(mux *http.ServeMux) {
	log.Println("Server starting on :8080")
	log.Println("Access the frontend at http://localhost:8080")
	if err := http.ListenAndServe(":8080", mux); err != nil {
		log.Fatalf("could not start server: %v", err)
	}
}
