package api

import (
	"net/http"
	"strconv"

	"knife/base"
	"knife/db"
)

type DraftAPI struct {
	draftModel *db.DraftModel
}

func NewDraftAPI(draftModel *db.DraftModel) *DraftAPI {
	return &DraftAPI{draftModel: draftModel}
}

func (a *DraftAPI) RegisterHandlers(router *base.APIRouter) {
	router.POST("drafts", a.saveDraft, []string{"AuthMiddleware"})
	router.GET("drafts", a.listDrafts, []string{"AuthMiddleware"})
	router.GET("drafts/{id}", a.getDraft, []string{"AuthMiddleware"})
	router.DELETE("drafts/{id}", a.deleteDraft, []string{"AuthMiddleware"})
}

func (a *DraftAPI) saveDraft(ctx base.APIContext) {
	var draft db.Draft
	if err := ctx.GetContext(&draft); err != nil {
		ctx.ReturnError("badrequest", "Invalid request body", http.StatusBadRequest)
		return
	}
	if err := a.draftModel.Save(&draft); err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}
	ctx.ReturnJSON(draft)
}

func (a *DraftAPI) listDrafts(ctx base.APIContext) {
	drafts, err := a.draftModel.List()
	if err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}
	ctx.ReturnJSON(drafts)
}

func (a *DraftAPI) getDraft(ctx base.APIContext) {
	idStr := ctx.GetPathParamValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.ReturnError("badrequest", "Invalid draft ID", http.StatusBadRequest)
		return
	}
	draft, err := a.draftModel.Get(id)
	if err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}
	ctx.ReturnJSON(draft)
}

func (a *DraftAPI) deleteDraft(ctx base.APIContext) {
	idStr := ctx.GetPathParamValue("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		ctx.ReturnError("badrequest", "Invalid draft ID", http.StatusBadRequest)
		return
	}
	if err := a.draftModel.Delete(id); err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}
	ctx.RawRetrun([]byte(""), http.StatusNoContent)
}
