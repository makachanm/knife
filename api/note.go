package api

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"knife/base"
	"knife/db"
)

type NoteAPI struct {
	noteModel     *db.NoteModel
	profileModel  *db.ProfileModel
	followerModel *db.FollowerModel
	jobQueue      *base.JobQueue
}

func NewNoteAPI(noteModel *db.NoteModel, profileModel *db.ProfileModel, followerModel *db.FollowerModel, jobQueue *base.JobQueue) *NoteAPI {
	return &NoteAPI{noteModel: noteModel, profileModel: profileModel, followerModel: followerModel, jobQueue: jobQueue}
}

// RegisterHandlers registers the API handlers for notes.
func (a *NoteAPI) RegisterHandlers(router *base.APIRouter) {
	router.GET("notes", a.listNotes, nil)
	router.POST("notes", a.createNote, []string{"AuthMiddleware"})
	router.GET("notes/{id}", a.getNote, nil)
	router.DELETE("notes/{id}", a.deleteNote, []string{"AuthMiddleware"})
}

func (a *NoteAPI) listNotes(ctx base.APIContext) {
	notes, err := a.noteModel.ListRecent()
	if err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}
	ctx.ReturnJSON(notes)
}

func (a *NoteAPI) createNote(ctx base.APIContext) {
	var note db.Note
	if err := ctx.GetContext(&note); err != nil {
		ctx.ReturnError("badrequest", "Invalid request body", http.StatusBadRequest)
		return
	}

	profile, err := a.profileModel.Get()
	if err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}

	note.Host = ctx.GetHost()
	note.AuthorName = profile.DisplayName
	note.AuthorFinger = profile.Finger
	if err := a.noteModel.CreateLocalNote(&note); err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}

	// Fan-out to followers
	followers, err := a.followerModel.ListFollowers()
	if err != nil {
		log.Printf("failed to list followers: %v", err)
		ctx.RawRetrun([]byte(""), http.StatusCreated)
		return
	}

	actorURI := "https://" + note.Host + "/profile"
	apNote := map[string]interface{}{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           note.URI,
		"type":         "Note",
		"published":    note.CreateTime.Format("2006-01-02T15:04:05Z"),
		"attributedTo": actorURI,
		"content":      note.Content,
		"to":           []string{"https://www.w3.org/ns/activitystreams#Public"},
	}

	activity := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type":     "Create",
		"actor":    actorURI,
		"object":   apNote,
	}
	activityBytes, err := json.Marshal(activity)
	if err != nil {
		log.Printf("failed to marshal activity: %v", err)
		ctx.RawRetrun([]byte(""), http.StatusCreated)
		return
	}

	for _, follower := range followers {
		follower := follower // Create a new variable for the closure
		job := func() {
			req, err := http.NewRequest("POST", follower.InboxURI, bytes.NewBuffer(activityBytes))
			if err != nil {
				log.Printf("failed to create request for follower %s: %v", follower.ActorURI, err)
				return
			}
			req.Header.Set("Content-Type", "application/activity+json")

			// TODO: Add HTTP Signature

			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				log.Printf("failed to send note to follower %s: %v", follower.ActorURI, err)
				return
			}
			defer resp.Body.Close()

			if resp.StatusCode < 200 || resp.StatusCode >= 300 {
				log.Printf("follower %s returned status %d", follower.ActorURI, resp.StatusCode)
			} else {
				log.Printf("Successfully sent note to follower %s", follower.ActorURI)
			}
		}
		a.jobQueue.Enqueue(job)
	}

	ctx.RawRetrun([]byte(""), http.StatusCreated)
}

func (a *NoteAPI) getNote(ctx base.APIContext) {
	idStr := ctx.GetPathParamValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.ReturnError("badrequest", "Invalid note ID", http.StatusBadRequest)
		return
	}

	note, err := a.noteModel.Get(id)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.ReturnError("notfound", "Note not found", http.StatusNotFound)
		} else {
			ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		}
		return
	}

	ctx.ReturnJSON(note)
}

func (a *NoteAPI) deleteNote(ctx base.APIContext) {
	idStr := ctx.GetPathParamValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.ReturnError("badrequest", "Invalid note ID", http.StatusBadRequest)
		return
	}

	err = a.noteModel.Delete(id)
	if err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}
	ctx.RawRetrun([]byte(""), http.StatusNoContent)
}
