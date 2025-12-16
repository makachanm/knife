package api

import (
	"strconv"

	"knife/base"
	"knife/db"
)

type BookmarkAPI struct {
	BookmarkModel *db.BookmarkModel
	NoteModel     *db.NoteModel // Assuming NoteModel is needed for fetching note details
}

func NewBookmarkAPI(bookmarkModel *db.BookmarkModel, noteModel *db.NoteModel) *BookmarkAPI {
	return &BookmarkAPI{
		BookmarkModel: bookmarkModel,
		NoteModel:     noteModel,
	}
}

func (a *BookmarkAPI) RegisterHandlers(router *base.APIRouter) {
	router.POST("bookmarks", a.createBookmark, []string{"AuthMiddleware"})
	router.GET("bookmarks", a.listBookmarks, nil)
	router.DELETE("bookmarks/{note_id}", a.deleteBookmark, []string{"AuthMiddleware"})
}

func (a *BookmarkAPI) createBookmark(ctx base.APIContext) {
	var req struct {
		NoteID interface{} `json:"note_id"`
	}
	if err := ctx.GetContext(&req); err != nil {
		ctx.ReturnError("invalid_request", err.Error(), 400)
		return
	}

	var noteID int64
	switch v := req.NoteID.(type) {
	case string:
		var err error
		noteID, err = strconv.ParseInt(v, 10, 64)
		if err != nil {
			ctx.ReturnError("invalid_request", "note_id must be a valid integer", 400)
			return
		}
	case float64:
		noteID = int64(v)
	default:
		ctx.ReturnError("invalid_request", "note_id must be a valid integer", 400)
		return
	}

	bookmark := &db.Bookmark{NoteID: noteID}
	if err := a.BookmarkModel.Create(bookmark); err != nil {
		ctx.ReturnError("server_error", err.Error(), 500)
		return
	}

	ctx.ReturnJSON(map[string]interface{}{
		"bookmark_id": bookmark.ID,
	})
}

func (a *BookmarkAPI) listBookmarks(ctx base.APIContext) {
	// Fetch all bookmarks for the single user
	bookmarks, err := a.BookmarkModel.ListAll()
	if err != nil {
		ctx.ReturnError("server_error", err.Error(), 500)
		return
	}

	// Fetch note details for each bookmark
	notes := make([]map[string]interface{}, 0)
	for _, bookmark := range bookmarks {
		note, err := a.NoteModel.Get(bookmark.NoteID)
		if err != nil {
			ctx.ReturnError("server_error", "Failed to fetch note details", 500)
			return
		}
		notes = append(notes, map[string]interface{}{
			"note_id": note.ID,
			"content": note.Content,
			"author":  note.AuthorName,
			"created": note.CreateTime,
		})
	}

	ctx.ReturnJSON(notes)
}

func (a *BookmarkAPI) deleteBookmark(ctx base.APIContext) {
	noteIDStr := ctx.GetPathParamValue("note_id")
	noteID, err := strconv.ParseInt(noteIDStr, 10, 64)
	if err != nil {
		ctx.ReturnError("invalid_request", "Invalid note ID", 400)
		return
	}

	if err := a.BookmarkModel.Delete(noteID); err != nil {
		ctx.ReturnError("server_error", err.Error(), 500)
		return
	}

	ctx.RawRetrun(nil, 204)
}
