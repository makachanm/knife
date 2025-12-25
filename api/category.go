package api

import (
	"net/http"

	"knife/base"
	"knife/db"
)

type CategoryAPI struct {
	NoteModel *db.NoteModel
}

func NewCategoryAPI(noteModel *db.NoteModel) *CategoryAPI {
	return &CategoryAPI{
		NoteModel: noteModel,
	}
}

func (a *CategoryAPI) RegisterHandlers(router *base.APIRouter) {
	router.GET("category", a.listCategory, nil)
	router.GET("category/{name}", a.getCategoryContentByName, nil)
}

func (a *CategoryAPI) listCategory(ctx base.APIContext) {
	categories, err := a.NoteModel.ListCategories()
	if err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}

	ctx.ReturnJSON(categories)
}

func (a *CategoryAPI) getCategoryContentByName(ctx base.APIContext) {
	categoryName := ctx.GetPathParamValue("name")
	if categoryName == "" {
		ctx.ReturnError("badrequest", "Category name is required", http.StatusBadRequest)
		return
	}

	notes, err := a.NoteModel.ListByCategory(categoryName)
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