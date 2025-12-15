package api

import (
	"strconv"

	"knife/base"
	"knife/db"
)

type BookmarkAPI struct {
	BookmarkModel *db.BookmarkModel
}

func NewBookmarkAPI(bookmarkModel *db.BookmarkModel) *BookmarkAPI {
	return &BookmarkAPI{
		BookmarkModel: bookmarkModel,
	}
}

func (a *BookmarkAPI) RegisterHandlers(router *base.APIRouter) {
	router.POST("bookmarks", a.createBookmark, nil)
	router.GET("bookmarks", a.listBookmarks, nil)
	router.DELETE("bookmarks/{note_id}", a.deleteBookmark, nil)
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
	bookmarks, err := a.BookmarkModel.List()
	if err != nil {
		ctx.ReturnError("server_error", err.Error(), 500)
		return
	}

	ctx.ReturnJSON(bookmarks)
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
