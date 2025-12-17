package api

import (
	"encoding/json"
	"net/http"

	"knife/base"
	"knife/db"

	"github.com/go-ap/activitypub"
)

type ActivityPubAPI struct {
	noteModel    *db.NoteModel
	profileModel *db.ProfileModel
}

func NewActivityPubAPI(noteModel *db.NoteModel, profileModel *db.ProfileModel) *ActivityPubAPI {
	return &ActivityPubAPI{noteModel: noteModel, profileModel: profileModel}
}

// Webfinger handles /.well-known/webfinger requests
func (a *ActivityPubAPI) Webfinger(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	if resource == "" {
		http.Error(w, "missing resource", http.StatusBadRequest)
		return
	}
	profile, err := a.profileModel.Get()
	if err != nil {
		http.Error(w, "profile not found", http.StatusNotFound)
		return
	}

	host := r.Host
	id := "https://" + host + "/api/profile"

	actor := map[string]interface{}{
		"@context":          "https://www.w3.org/ns/activitystreams",
		"id":                id,
		"type":              "Person",
		"preferredUsername": profile.Finger,
		"name":              profile.DisplayName,
		"summary":           profile.Bio,
		"inbox":             "https://" + host + "/api/inbox",
		"outbox":            "https://" + host + "/api/outbox", // Placeholder
		"icon": map[string]interface{}{
			"type":      "Image",
			"mediaType": "image/png", // Assuming png, could be dynamic
			"url":       profile.AvatarURL,
		},
	}

	w.Header().Set("Content-Type", "application/jrd+json")
	json.NewEncoder(w).Encode(actor)
}

// RegisterHandlers registers the API handlers for notes.
func (a *ActivityPubAPI) RegisterHandlers(router *base.APIRouter) {
	router.POST("inbox", a.inbox, nil)
}

func (a *ActivityPubAPI) inbox(ctx base.APIContext) {
	data := ctx.RawBody()

	item, err := activitypub.UnmarshalJSON(data)
	if err != nil {
		ctx.ReturnError("badrequest", "Invalid activitypub json", http.StatusBadRequest)
		return
	}

	activitypub.OnActivity(item, func(act *activitypub.Activity) error {
		switch act.GetType() {
		case activitypub.CreateType:
			var authorName, authorFinger, authorHost string

			activitypub.OnObject(act.Actor, func(author *activitypub.Object) error {
				authorName = author.Name.String()
				authorFinger = author.GetID().String()
				authorHost = author.GetID().GetLink().String()
				if err != nil {
					return err
				}
				return nil
			})

			activitypub.OnObject(act.Object, func(obj *activitypub.Object) error {
				if obj.GetType() != activitypub.NoteType {
					return nil
				}
				note := &db.Note{
					URI:          obj.GetID().String(),
					Content:      obj.Content.First().String(),
					AuthorFinger: authorFinger,
					Host:         authorHost,
					AuthorName:   authorName,
				}
				return a.noteModel.CreateFederatedNote(note)
			})

		case activitypub.DeleteType:
			activitypub.OnObject(act.Object, func(obj *activitypub.Object) error {
				return a.noteModel.DeleteByURI(obj.GetID().String())
			})
		}
		return nil
	})

	ctx.RawRetrun([]byte(""), http.StatusOK)
}
