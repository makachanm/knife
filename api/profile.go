package api

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"knife/base"
	"knife/db"
)

type ProfileAPI struct {
	profileModel *db.ProfileModel
	noteModel    *db.NoteModel
}

func NewProfileAPI(profileModel *db.ProfileModel, noteModel *db.NoteModel) *ProfileAPI {
	return &ProfileAPI{profileModel: profileModel, noteModel: noteModel}
}

// RegisterHandlers registers the API handlers for profiles.
func (a *ProfileAPI) RegisterHandlers(router *base.APIRouter) {
	router.GET("profile", a.getProfile, nil)
	router.GET("profile/recent", a.getRecentNotes, nil)
	router.PUT("profile", a.updateProfile, []string{"AuthMiddleware"})
}

func (a *ProfileAPI) getProfile(ctx base.APIContext) {
	profile, err := a.profileModel.Get()
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.ReturnError("notfound", "Profile not found", http.StatusNotFound)
		} else {
			ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		}
		return
	}

	profile.Finger = "@" + profile.Finger + "@" + ctx.GetHost()
	profile.PasswordHash = "" // Hide sensitive information
	ctx.ReturnJSON(profile)
}

func (a *ProfileAPI) updateProfile(ctx base.APIContext) {
	var profile db.Profile
	if err := json.Unmarshal(ctx.RawBody(), &profile); err != nil {
		ctx.ReturnError("badrequest", "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := a.profileModel.Update(&profile); err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}

	ctx.ReturnJSON(profile)
}

func (a *ProfileAPI) getRecentNotes(ctx base.APIContext) {
	notes, err := a.noteModel.ListByMyRecent()
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
