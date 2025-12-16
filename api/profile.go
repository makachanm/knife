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
}

func NewProfileAPI(profileModel *db.ProfileModel) *ProfileAPI {
	return &ProfileAPI{profileModel: profileModel}
}

// RegisterHandlers registers the API handlers for profiles.
func (a *ProfileAPI) RegisterHandlers(router *base.APIRouter) {
	router.GET("profile", a.getProfile, nil)
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

	//var profile_n *db.Profile
	profile_n, err := a.profileModel.Get()
	profile.Finger = profile_n.Finger

	if err != nil {
		if err == sql.ErrNoRows {
			ctx.ReturnError("notfound", "Profile not found", http.StatusNotFound)
		} else {
			ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		}
		return
	}

	if err := a.profileModel.Update(&profile); err != nil {
		ctx.ReturnError("dberror", err.Error(), http.StatusInternalServerError)
		return
	}

	ctx.ReturnJSON(profile)
}
