package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"knife/base"
	"knife/db"
)

type NoteAPI struct {
	noteModel    *db.NoteModel
	profileModel *db.ProfileModel
}

func NewNoteAPI(noteModel *db.NoteModel, profileModel *db.ProfileModel) *NoteAPI {
	return &NoteAPI{noteModel: noteModel, profileModel: profileModel}
}

// RegisterHandlers registers the API handlers for notes.
func (a *NoteAPI) RegisterHandlers(router *base.APIRouter) {
	router.GET("notes", a.listNotes, []string{"all"})
	router.POST("notes", a.createNote, []string{"all"})
	router.GET("notes/{id}", a.getNote, []string{"all"})
	router.PUT("notes/{id}", a.updateNote, []string{"all"})
	router.DELETE("notes/{id}", a.deleteNote, []string{"all"})
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

func (a *NoteAPI) updateNote(ctx base.APIContext) {
	idStr := ctx.GetPathParamValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.ReturnError("badrequest", "Invalid note ID", http.StatusBadRequest)
		return
	}

	var note db.Note
	if err := json.Unmarshal(ctx.RawBody(), &note); err != nil {
		ctx.ReturnError("badrequest", "Invalid request body", http.StatusBadRequest)
		return
	}

	note.ID = id

	if err := a.noteModel.Update(&note); err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
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
