package ap

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"

	// For knife.Version
	"knife/db"
	"knife/etc"

	"crypto/x509"
	"encoding/pem"
	"time"

	"github.com/go-ap/activitypub"
	"github.com/go-fed/httpsig"
)

type ActivityPubAPI struct {
	noteModel     *db.NoteModel
	profileModel  *db.ProfileModel
	followerModel *db.FollowerModel
	httpsigModel  *db.HTTPSigModel
}

func NewActivityPubAPI(noteModel *db.NoteModel, profileModel *db.ProfileModel, followerModel *db.FollowerModel, httpsigModel *db.HTTPSigModel) *ActivityPubAPI {
	return &ActivityPubAPI{noteModel: noteModel, profileModel: profileModel, followerModel: followerModel, httpsigModel: httpsigModel}
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

	sig, err := a.httpsigModel.GetByActor(id)
	if err != nil {
		if err == sql.ErrNoRows {
			sig, err = a.httpsigModel.Create(id)
			if err != nil {
				http.Error(w, "failed to create httpsig", http.StatusInternalServerError)
				return
			}
		} else {
			http.Error(w, "failed to get httpsig", http.StatusInternalServerError)
			return
		}
	}

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
		"publicKey": map[string]interface{}{
			"id":           id + "#main-key",
			"owner":        id,
			"publicKeyPem": sig.PublicKey,
		},
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	json.NewEncoder(w).Encode(actor)
}

// NodeInfoHandler handles /.well-known/nodeinfo requests
func (a *ActivityPubAPI) NodeInfoHandler(w http.ResponseWriter, r *http.Request) {
	nodeInfo := NodeInfo{
		Version: "2.1", // Currently preferred NodeInfo version for ActivityPub
		Software: NodeInfoSoftware{
			Name:       "knife",
			Version:    etc.Version,                          // Using version from main package
			Homepage:   "https://github.com/makachanm/knife", // Placeholder, ideally from config
			Repository: "https://github.com/makachanm/knife", // Placeholder, ideally from config
		},
		Protocols: []string{"activitypub"},
		Services: NodeInfoServices{
			Outbound: []string{}, // Customize if your instance supports outbound federation for other services
			Inbound:  []string{}, // Customize if your instance supports inbound federation from other services
		},
		OpenRegistrations: false, // As per single-user application
		Usage: NodeInfoUsage{
			Users: NodeInfoUsageUsers{
				Total:          1, // Single-user application
				ActiveHalfyear: 1, // Single-user application
				ActiveMonth:    1, // Single-user application
			},
		},
		Metadata: map[string]interface{}{
			"nodeName":        "knife instance",                                  // Customizable
			"nodeDescription": "A personal ActivityPub server powered by knife.", // Customizable
		},
	}

	w.Header().Set("Content-Type", "application/json; profile=\"http://nodeinfo.diaspora.software/ns/schema/2.1#\"")
	json.NewEncoder(w).Encode(nodeInfo)
}

func (a *ActivityPubAPI) Note(w http.ResponseWriter, r *http.Request) {
	idStr := r.URL.Path[len("/notes/"):]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid note ID", http.StatusBadRequest)
		return
	}

	note, err := a.noteModel.Get(id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Note not found", http.StatusNotFound)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	// Determine visibility based on PublicRange
	to, cc := a.getVisibilityTargets(note)

	apNote := map[string]interface{}{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           note.URI,
		"type":         "Note",
		"attributedTo": "https://" + r.Host + "/profile",
		"content":      note.Content,
		"published":    note.CreateTime.Format("2006-01-02T15:04:05Z"), // RFC3339 format
		"to":           to,
		"cc":           cc,
	}

	if note.Cw != "" { 
		apNote["sensitive"] = true
		apNote["summary"] = note.Cw
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	json.NewEncoder(w).Encode(apNote)
}

// getVisibilityTargets determines the "to" and "cc" fields based on the note's visibility.
func (a *ActivityPubAPI) getVisibilityTargets(note *db.Note) ([]string, []string) {
	var to []string
	var cc []string

	switch note.PublicRange {
	case db.NotePublicRangePublic:
		to = []string{"https://www.w3.org/ns/activitystreams#Public"}
	case db.NotePublicRangeFollowers:
		to = []string{}
		cc = []string{"https://" + note.Host + "/followers"}
	case db.NotePublicRangeUnlisted:
		to = []string{}
		cc = []string{"https://www.w3.org/ns/activitystreams#Public"}
	case db.NotePublicRangePrivate:
		to = []string{"https://" + note.Host + "/profile"}
	default:
		// Default to private if the range is unknown
		to = []string{"https://" + note.Host + "/profile"}
	}

	return to, cc
}

func (a *ActivityPubAPI) sendActivity(inbox string, actorIRI string, activity interface{}) error {
	activityJSON, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("failed to marshal activity: %w", err)
	}

	req, err := http.NewRequest("POST", inbox, bytes.NewBuffer(activityJSON))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/activity+json")
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	inboxURL, err := url.Parse(inbox)
	if err != nil {
		return fmt.Errorf("failed to parse inbox url: %w", err)
	}
	req.Header.Set("Host", inboxURL.Host)

	profile, err := a.profileModel.Get()
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}
	sig, err := a.httpsigModel.GetByActor(profile.Finger)
	if err != nil {
		return fmt.Errorf("failed to get httpsig for %s: %w", actorIRI, err)
	}

	block, _ := pem.Decode([]byte(sig.PrivateKey))
	if block == nil {
		return fmt.Errorf("failed to decode private key")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	// Sign the request
	keyID := actorIRI + "#main-key"
	prefs := []httpsig.Algorithm{httpsig.RSA_SHA256}
	digestAlgo := httpsig.DigestSha256
	headersToSign := []string{httpsig.RequestTarget, "host", "date", "digest"}
	signer, _, err := httpsig.NewSigner(prefs, digestAlgo, headersToSign, httpsig.Signature, 65535)
	if err != nil {
		return fmt.Errorf("failed to create signer: %w", err)
	}
	if err := signer.SignRequest(privateKey, keyID, req, activityJSON); err != nil {
		return fmt.Errorf("failed to sign request: %w", err)
	}

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
			return a.handleFollowActivity(act, r.Host)
		case activitypub.UndoType:
			return a.handleUndoActivity(act)
		case activitypub.CreateType:
			return a.handleCreateActivity(act)
		case activitypub.UpdateType:
			return a.handleUpdateActivity(act)
		case activitypub.DeleteType:
			return a.handleDeleteActivity(act)
		default:
			log.Printf("Inbox: unsupported activity type %s", act.GetType())
			return nil
		}
	})

	if err != nil {
		log.Printf("Inbox: error processing activity: %v", err)
	}

	w.WriteHeader(http.StatusOK)
}

// handleFollowActivity processes Follow activities.
func (a *ActivityPubAPI) handleFollowActivity(act *activitypub.Activity, host string) error {
	actor, inboxURI, err := a.resolveActorAndInbox(act.Actor)
	if err != nil {
		return fmt.Errorf("handleFollowActivity: %w", err)
	}

	log.Printf("Inbox: Adding follower %s", actor.GetID())
	err = a.followerModel.AddFollower(actor.GetID().String(), inboxURI)
	if err != nil {
		return err
	}

	myActorIRI := "https://" + host + "/profile"
	accept := activitypub.Accept{
		Type:   activitypub.AcceptType,
		Actor:  activitypub.IRI(myActorIRI),
		Object: act,
		To:     []activitypub.Item{activitypub.IRI(inboxURI)},
	}

	go func() {
		log.Printf("Sending Accept for Follow to %s", inboxURI)
		if err := a.sendActivity(inboxURI, myActorIRI, accept); err != nil {
			log.Printf("Error sending Accept activity: %v", err)
		}
	}()

	return nil
}

// handleUndoActivity processes Undo activities.
func (a *ActivityPubAPI) handleUndoActivity(act *activitypub.Activity) error {
	return activitypub.OnObject(act.Object, func(object *activitypub.Object) error {
		if object.GetType() == activitypub.FollowType {
			actor, err := a.resolveActor(act.Actor)
			if err != nil {
				return fmt.Errorf("handleUndoActivity: %w", err)
			}

			log.Printf("Inbox: Removing follower %s", actor.GetID())
			return a.followerModel.RemoveFollower(actor.GetID().String())
		}
		return nil
	})
}

// handleCreateActivity processes Create activities.
func (a *ActivityPubAPI) handleCreateActivity(act *activitypub.Activity) error {
	actor, err := a.resolveActor(act.Actor)
	if err != nil {
		return fmt.Errorf("handleCreateActivity: %w", err)
	}

	authorName := actor.Name.String()
	authorFinger := actor.PreferredUsername.String() + "@" + a.extractHost(actor.URL.GetLink().String())
	authorHost := a.extractHost(actor.URL.GetLink().String())

	return activitypub.OnObject(act.Object, func(obj *activitypub.Object) error {
		if obj.GetType() != activitypub.NoteType {
			return nil
		}

		// Determine the visibility of the note
		publicRange := a.determinePublicRange(obj)

		log.Printf("Inbox: Creating federated note from %s", obj.GetID())
		note := &db.Note{
			URI:          obj.GetID().String(),
			Content:      stripHTML(obj.Content.First().String()),
			AuthorFinger: authorFinger,
			Host:         authorHost,
			AuthorName:   authorName,
			PublicRange:  publicRange,
		}
		return a.noteModel.CreateFederatedNote(note)
	})
}

// determinePublicRange determines the public range of a note based on its "to" and "cc" fields.
func (a *ActivityPubAPI) determinePublicRange(obj *activitypub.Object) db.NotePublicRange {
	to := obj.To
	cc := obj.CC

	// Check if the note is public
	for _, item := range append(to, cc...) {
		if item.GetLink() == "https://www.w3.org/ns/activitystreams#Public" {
			return db.NotePublicRangePublic
		}
	}

	// Check if the note is for followers
	for _, item := range to {
		if item.GetLink() == obj.GetLink() { // Replace with your followers URL
			return db.NotePublicRangeFollowers
		}
	}

	// Check if the note is unlisted
	for _, item := range cc {
		if item.GetLink() == "https://www.w3.org/ns/activitystreams#Public" {
			return db.NotePublicRangeUnlisted
		}
	}

	// Default to private
	return db.NotePublicRangePrivate
}

// handleUpdateActivity processes Update activities.
func (a *ActivityPubAPI) handleUpdateActivity(act *activitypub.Activity) error {
	return activitypub.OnObject(act.Object, func(obj *activitypub.Object) error {
		if obj.GetType() != activitypub.NoteType {
			return nil
		}
		log.Printf("Inbox: Updating federated note %s", obj.GetID())
		note := &db.Note{
			URI:     obj.GetID().String(),
			Content: stripHTML(obj.Content.First().String()),
		}
		return a.noteModel.UpdateFederatedNote(note)
	})
}

// handleDeleteActivity processes Delete activities.
func (a *ActivityPubAPI) handleDeleteActivity(act *activitypub.Activity) error {
	return activitypub.OnObject(act.Object, func(obj *activitypub.Object) error {
		log.Printf("Inbox: Deleting federated object %s", obj.GetID())
		return a.noteModel.DeleteByURI(obj.GetID().String())
	})
}

// resolveActorAndInbox resolves an actor and their inbox URI.
func (a *ActivityPubAPI) resolveActorAndInbox(actorRef activitypub.Item) (*activitypub.Actor, string, error) {
	actor, err := a.resolveActor(actorRef)
	if err != nil {
		return nil, "", err
	}

	inboxURI := actor.Inbox.GetLink().String()
	if inboxURI == "" {
		return nil, "", fmt.Errorf("actor %s has no inbox URI", actor.GetID())
	}

	return actor, inboxURI, nil
}

// resolveActor resolves an actor from an ActivityPub item.
func (a *ActivityPubAPI) resolveActor(actorRef activitypub.Item) (*activitypub.Actor, error) {
	actor, err := activitypub.ToActor(actorRef)
	if err == nil {
		return actor, nil
	}

	if iri := actorRef.GetLink(); iri != "" {
		log.Printf("Fetching actor from IRI %s", iri)
		return fetchActor(iri.String())
	}

	return nil, fmt.Errorf("could not resolve actor")
}

// extractHost extracts the host from a URL string.
func (a *ActivityPubAPI) extractHost(urlStr string) string {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return ""
	}
	return parsedURL.Host
}

// fetchActor fetches an ActivityPub Actor from the given IRI.
func fetchActor(iri string) (*activitypub.Actor, error) {
	// Create a GET request to fetch the actor
	req, err := http.NewRequest("GET", iri, nil)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/activity+json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: failed to fetch actor: %w", err)
	}
	defer resp.Body.Close()

	// Check for non-2xx status codes
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("fetchActor: unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Read and parse the response body
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: failed to read response body: %w", err)
	}

	item, err := activitypub.UnmarshalJSON(data)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: failed to unmarshal JSON: %w", err)
	}

	actor, err := activitypub.ToActor(item)
	if err != nil {
		return nil, fmt.Errorf("fetchActor: item is not an actor: %w", err)
	}

	return actor, nil
}

// stripHTML takes a string that contains HTML and returns just the text content.
func stripHTML(s string) string {
	if !strings.Contains(s, "<") {
		return s
	}
	// The context for ParseFragment is a body tag, which is a reasonable default.
	nodes, err := html.ParseFragment(strings.NewReader(s), &html.Node{
		Type:     html.ElementNode,
		Data:     "body",
		DataAtom: atom.Body,
	})
	if err != nil {
		// Fallback to original string on parse error.
		return s
	}

	var b strings.Builder
	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.TextNode {
			// When parsing fragments, newlines can be introduced as text nodes.
			// We can trim space to avoid extra whitespace.
			b.WriteString(strings.TrimSpace(n.Data))
		}
		// Traverse children
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			if n.Type == html.ElementNode {
				if n.DataAtom == atom.Script || n.DataAtom == atom.Style {
					continue
				}
			}
			f(c)
			// Add a space between block-level elements
			if c.Type == html.ElementNode && (c.DataAtom == atom.P || c.DataAtom == atom.Div || c.DataAtom == atom.Br) {
				b.WriteString(" ")
			}
		}
	}

	for _, n := range nodes {
		f(n)
	}

	// Clean up multiple spaces.
	return strings.Join(strings.Fields(b.String()), " ")
}
