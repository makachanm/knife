package ap

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"knife/db"

	"github.com/go-ap/activitypub"
)

type ActivityPubAPI struct {
	noteModel     *db.NoteModel
	profileModel  *db.ProfileModel
	followerModel *db.FollowerModel
}

func NewActivityPubAPI(noteModel *db.NoteModel, profileModel *db.ProfileModel, followerModel *db.FollowerModel) *ActivityPubAPI {
	return &ActivityPubAPI{noteModel: noteModel, profileModel: profileModel, followerModel: followerModel}
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
	id := "https://" + host + "/profile"

	actor := map[string]interface{}{
		"@context":          "https://www.w3.org/ns/activitystreams",
		"id":                id,
		"type":              "Person",
		"preferredUsername": profile.Finger,
		"name":              profile.DisplayName,
		"summary":           profile.Bio,
		"inbox":             "https://" + host + "/inbox",
		"outbox":            "https://" + host + "/outbox", // Placeholder
		"icon": map[string]interface{}{
			"type":      "Image",
			"mediaType": "image/png", // Assuming png, could be dynamic
			"url":       profile.AvatarURL,
		},
	}

	w.Header().Set("Content-Type", "application/jrd+json")
	json.NewEncoder(w).Encode(actor)
}

// Actor serves the site's actor profile.
func (a *ActivityPubAPI) Actor(w http.ResponseWriter, r *http.Request) {
	profile, err := a.profileModel.Get()
	if err != nil {
		http.Error(w, "profile not found", http.StatusNotFound)
		return
	}

	host := r.Host
	id := "https://" + host + "/profile"

	actor := map[string]interface{}{
		"@context":          "https://www.w3.org/ns/activitystreams",
		"id":                id,
		"type":              "Person",
		"preferredUsername": profile.Finger,
		"name":              profile.DisplayName,
		"summary":           profile.Bio,
		"inbox":             "https://" + host + "/inbox",
		"outbox":            "https://" + host + "/outbox",
		"icon": map[string]interface{}{
			"type":      "Image",
			"mediaType": "image/png", // Assuming png, could be dynamic
			"url":       profile.AvatarURL,
		},
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	json.NewEncoder(w).Encode(actor)
}

// Inbox handles incoming ActivityPub POST requests.
func (a *ActivityPubAPI) Inbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}

	item, err := activitypub.UnmarshalJSON(data)
	if err != nil {
		http.Error(w, "Invalid ActivityPub JSON", http.StatusBadRequest)
		return
	}

	activitypub.OnActivity(item, func(act *activitypub.Activity) error {
		switch act.GetType() {
		case activitypub.FollowType:
			var actorURI, inboxURI string
			actor, err := activitypub.ToActor(act.Actor)
			if err != nil {
				return err
			}
			actorURI = actor.GetID().String()
			inboxURI = string(actor.Inbox.GetLink())
			fmt.Println("Actor URI:", actorURI)
			fmt.Println("Inbox URI:", inboxURI)
			if actorURI != "" && inboxURI != "" {
				return a.followerModel.AddFollower(actorURI, inboxURI)
			}
		case activitypub.UndoType:
			activitypub.OnObject(act.Object, func(object *activitypub.Object) error {
				if object.GetType() == activitypub.FollowType {
					var actorURI string
					activitypub.OnObject(act.Actor, func(actor *activitypub.Object) error {
						actorURI = actor.GetID().String()
						return nil
					})
					if actorURI != "" {
						return a.followerModel.RemoveFollower(actorURI)
					}
				}
				return nil
			})
		case activitypub.CreateType:
			var authorName, authorFinger, authorHost string

			activitypub.OnObject(act.Actor, func(author *activitypub.Object) error {
				authorName = author.Name.String()
				authorFinger = author.GetID().String()
				authorHost = author.GetID().GetLink().String()
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

		case activitypub.UpdateType:
			activitypub.OnObject(act.Object, func(obj *activitypub.Object) error {
				if obj.GetType() != activitypub.NoteType {
					return nil
				}
				note := &db.Note{
					URI:     obj.GetID().String(),
					Content: obj.Content.First().String(),
				}
				return a.noteModel.UpdateFederatedNote(note)
			})
		case activitypub.DeleteType:
			activitypub.OnObject(act.Object, func(obj *activitypub.Object) error {
				return a.noteModel.DeleteByURI(obj.GetID().String())
			})
		}
		return nil
	})

	w.WriteHeader(http.StatusOK)
}

// Note serves a single note as an ActivityPub object.
func (a *ActivityPubAPI) Note(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/notes/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	note, err := a.noteModel.Get(id)
	if err != nil {
		http.Error(w, "Note not found", http.StatusNotFound)
		return
	}

	actorURI := "https://" + r.Host + "/profile" // Assuming single user profile URI
	apNote := map[string]interface{}{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           note.URI,
		"type":         "Note",
		"published":    note.CreateTime.Format("2006-01-02T15:04:05Z"),
		"attributedTo": actorURI,
		"content":      note.Content,
		"to":           []string{"https://www.w3.org/ns/activitystreams#Public"},
	}
	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	json.NewEncoder(w).Encode(apNote)
}
