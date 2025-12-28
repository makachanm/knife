package api

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"

	"knife/ap"
	"knife/base"
	"knife/db"

	"github.com/gomarkdown/markdown"
	"github.com/microcosm-cc/bluemonday"
)

type NoteAPI struct {
	noteModel     *db.NoteModel
	profileModel  *db.ProfileModel
	followerModel *db.FollowerModel
	dispatcher    *ap.ActivityDispatcher
}

func NewNoteAPI(noteModel *db.NoteModel, profileModel *db.ProfileModel, followerModel *db.FollowerModel, dispatcher *ap.ActivityDispatcher) *NoteAPI {
	return &NoteAPI{noteModel: noteModel, profileModel: profileModel, followerModel: followerModel, dispatcher: dispatcher}
}

type NoteResponse struct {
	ID           int64              `json:"id"`
	URI          string             `json:"uri"`
	Cw           string             `json:"cw,omitempty"`
	Content      string             `json:"content"`
	Host         string             `json:"host"`
	AuthorName   string             `json:"author_name"`
	AuthorFinger string             `json:"author_finger"`
	PublicRange  db.NotePublicRange `json:"public_range,string"`
	CreateTime   time.Time          `json:"create_time"`
	Category     string             `json:"category,omitempty"`
}

// RegisterHandlers registers the API handlers for notes.
func (a *NoteAPI) RegisterHandlers(router *base.APIRouter) {
	router.GET("notes", a.listNotes, []string{"AuthMiddleware"})
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

	noteResponses := make([]NoteResponse, 0, len(notes))
	for _, note := range notes {
		noteResponses = append(noteResponses, NoteResponse{
			ID:           note.ID,
			URI:          note.URI,
			Cw:           note.Cw,
			Content:      note.Content,
			Host:         note.Host,
			AuthorName:   note.AuthorName,
			AuthorFinger: note.AuthorFinger,
			PublicRange:  note.PublicRange,
			CreateTime:   note.CreateTime,
			Category:     note.Category,
		})
	}

	ctx.ReturnJSON(noteResponses)
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
	unsafeHTML := markdown.ToHTML([]byte(note.Content), nil, nil)
	note.Content = string(bluemonday.UGCPolicy().SanitizeBytes(unsafeHTML))
	if err := a.noteModel.CreateLocalNote(&note); err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}

	// Fan-out to followers
	if err := a.dispatcher.SendCreateNote(&note); err != nil {
		log.Printf("failed to dispatch create note activity: %v", err)
		// We don't fail the request if dispatching fails, but we log it.
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

	response := NoteResponse{
		ID:           note.ID,
		URI:          note.URI,
		Cw:           note.Cw,
		Content:      note.Content,
		Host:         note.Host,
		AuthorName:   note.AuthorName,
		AuthorFinger: note.AuthorFinger,
		PublicRange:  note.PublicRange,
		CreateTime:   note.CreateTime,
		Category:     note.Category,
	}

	ctx.ReturnJSON(response)
}

func (a *NoteAPI) deleteNote(ctx base.APIContext) {
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

	if err := a.dispatcher.SendDeleteNote(note); err != nil {
		log.Printf("failed to dispatch delete note activity: %v", err)
	}

	err = a.noteModel.Delete(id)
	if err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}
	ctx.RawRetrun([]byte(""), http.StatusNoContent)
}
