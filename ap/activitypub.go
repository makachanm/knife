package ap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
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

func (a *ActivityPubAPI) sendActivity(inbox string, activity interface{}) error {
	activityJSON, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal activity: %w", err)
	}

	req, err := http.NewRequest("POST", inbox, bytes.NewBuffer(activityJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/activity+json")

	// TODO: Sign the request

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send activity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("activity send failed with status %d: %s", resp.StatusCode, string(body))
	}

	log.Printf("Successfully sent activity to %s", inbox)
	return nil
}

// Inbox handles incoming ActivityPub POST requests.
func (a *ActivityPubAPI) Inbox(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("Inbox: failed to read request body: %v", err)
		http.Error(w, "failed to read request body", http.StatusInternalServerError)
		return
	}
	log.Printf("Inbox: received body: %s", data)

	item, err := activitypub.UnmarshalJSON(data)
	if err != nil {
		log.Printf("Inbox: Invalid ActivityPub JSON: %v", err)
		http.Error(w, "Invalid ActivityPub JSON", http.StatusBadRequest)
		return
	}

	err = activitypub.OnActivity(item, func(act *activitypub.Activity) error {
		log.Printf("Inbox: processing activity of type %s", act.GetType())
		switch act.GetType() {
		case activitypub.FollowType:
			var actorURI, inboxURI string
			var actor *activitypub.Actor
			var err error

			actor, err = activitypub.ToActor(act.Actor)
			if err != nil {
				// It's probably an IRI, let's fetch it
				if iri := act.Actor.GetLink(); iri != "" {
					log.Printf("Inbox: fetching actor from IRI %s", iri)
					actor, err = fetchActor(iri.String())
					if err != nil {
						return fmt.Errorf("failed to fetch actor from IRI: %w", err)
					}
				} else {
					return fmt.Errorf("could not resolve actor from Follow activity")
				}
			}

			actorURI = actor.GetID().String()
			inboxURI = string(actor.Inbox.GetLink())

			if actorURI != "" && inboxURI != "" {
				log.Printf("Inbox: Adding follower %s", actorURI)
				err := a.followerModel.AddFollower(actorURI, inboxURI)
				if err != nil {
					return err
				}

				// Send Accept activity
				_, err = a.profileModel.Get()
				if err != nil {
					log.Printf("Error getting profile for sending Accept: %v", err)
					return nil // Don't block the rest of the processing
				}

				host := r.Host // Assuming we can get host from the request
				myActorIRI := "https://" + host + "/profile"

				accept := activitypub.Accept{
					Type:   activitypub.AcceptType,
					Actor:  activitypub.IRI(myActorIRI),
					Object: act,
				}

				go func() {
					log.Printf("Sending Accept for Follow to %s", inboxURI)
					err := a.sendActivity(inboxURI, accept)
					if err != nil {
						log.Printf("Error sending Accept activity: %v", err)
					}
				}()

				return nil
			}
			return fmt.Errorf("could not determine actor URI or inbox URI from Follow activity")
		case activitypub.UndoType:
			activitypub.OnObject(act.Object, func(object *activitypub.Object) error {
				if object.GetType() == activitypub.FollowType {
					var actorURI string
					var actor *activitypub.Actor
					var err error

					actor, err = activitypub.ToActor(act.Actor)
					if err != nil {
						// It's probably an IRI, let's fetch it
						if iri := act.Actor.GetLink(); iri != "" {
							log.Printf("Inbox: fetching actor from IRI %s", iri)
							actor, err = fetchActor(iri.String())
							if err != nil {
								return fmt.Errorf("failed to fetch actor from IRI: %w", err)
							}
						} else {
							return fmt.Errorf("could not resolve actor from Follow activity")
						}
					}

					actorURI = actor.GetID().String()
					if actorURI != "" {
						log.Printf("Inbox: Removing follower %s", actorURI)
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
				log.Printf("Inbox: Creating federated note from %s", obj.GetID())
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
				log.Printf("Inbox: Updating federated note %s", obj.GetID())
				note := &db.Note{
					URI:     obj.GetID().String(),
					Content: obj.Content.First().String(),
				}
				return a.noteModel.UpdateFederatedNote(note)
			})
		case activitypub.DeleteType:
			activitypub.OnObject(act.Object, func(obj *activitypub.Object) error {
				log.Printf("Inbox: Deleting federated object %s", obj.GetID())
				return a.noteModel.DeleteByURI(obj.GetID().String())
			})
		}
		return nil
	})
	if err != nil {
		log.Printf("Inbox: error processing activity: %v", err)
		// Not returning http error because we don't want to expose internal errors
	}

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

func fetchActor(actorIRI string) (*activitypub.Actor, error) {
	req, err := http.NewRequest("GET", actorIRI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/activity+json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("failed to fetch actor: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	item, err := activitypub.UnmarshalJSON(body)
	if err != nil {
		return nil, err
	}

	actor, err := activitypub.ToActor(item)
	if err != nil {
		return nil, err
	}
	return actor, nil
}
