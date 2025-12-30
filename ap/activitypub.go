package ap

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"

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

func (a *ActivityPubAPI) getProtocol() string {
	if proto := os.Getenv("KNIFE_PROTOCOL"); proto != "" {
		return proto
	}
	return "https"
}

func (a *ActivityPubAPI) getHost(r *http.Request) string {
	if host := os.Getenv("KNIFE_HOST"); host != "" {
		return host
	}
	return r.Host
}

func (a *ActivityPubAPI) getBaseURL(r *http.Request) string {
	return fmt.Sprintf("%s://%s", a.getProtocol(), a.getHost(r))
}

// Webfinger handles /.well-known/webfinger requests
func (a *ActivityPubAPI) Webfinger(w http.ResponseWriter, r *http.Request) {
	resource := r.URL.Query().Get("resource")
	log.Printf("Webfinger request for resource: %s", resource)

	if resource == "" {
		http.Error(w, "missing resource", http.StatusBadRequest)
		return
	}

	profile, err := a.profileModel.Get()
	if err != nil {
		log.Printf("Webfinger: profile lookup failed: %v", err)
		http.Error(w, "profile not found", http.StatusNotFound)
		return
	}

	host := a.getHost(r)
	canonicalSubject := fmt.Sprintf("acct:%s@%s", profile.Finger, host)
	id := a.getBaseURL(r) + "/profile"

	jrd := map[string]interface{}{
		"subject": canonicalSubject,
		"aliases": []string{id},
		"links": []map[string]interface{}{
			{
				"rel":  "self",
				"type": "application/activity+json",
				"href": id,
			},
			{
				"rel":  "http://webfinger.net/rel/profile-page",
				"type": "text/html",
				"href": id,
			},
		},
	}

	w.Header().Set("Content-Type", "application/jrd+json")
	if err := json.NewEncoder(w).Encode(jrd); err != nil {
		log.Printf("Webfinger: failed to encode response: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// Actor serves the site's actor profile.
func (a *ActivityPubAPI) Actor(w http.ResponseWriter, r *http.Request) {
	profile, err := a.profileModel.Get()
	if err != nil {
		http.Error(w, "profile not found", http.StatusNotFound)
		return
	}

	id := a.getBaseURL(r) + "/profile"

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
		"@context": []interface{}{
			"https://www.w3.org/ns/activitystreams",
			"https://w3id.org/security/v1",
		},
		"id":                id,
		"type":              "Person",
		"preferredUsername": profile.Finger,
		"name":              profile.DisplayName,
		"summary":           profile.Bio,
		"inbox":             a.getBaseURL(r) + "/inbox",
		"outbox":            a.getBaseURL(r) + "/outbox",
		"endpoints": map[string]interface{}{
			"sharedInbox": a.getBaseURL(r) + "/inbox", // Single user, so shared inbox is same as inbox
		},
		"icon": map[string]interface{}{
			"type":      "Image",
			"mediaType": "image/png", // Assuming png, could be dynamic
			"url":       profile.AvatarURL,
		},
		"publicKey": map[string]interface{}{
			"id":           id + "#main-key",
			"type":         "Key",
			"owner":        id,
			"publicKeyPem": sig.PublicKey,
		},
	}

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	json.NewEncoder(w).Encode(actor)
}

// NodeInfoHandler handles /.well-known/nodeinfo requests
func (a *ActivityPubAPI) NodeInfoHandler(w http.ResponseWriter, r *http.Request) {
	profileData, _ := a.profileModel.Get()
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
			"nodeName":        "knife",                                  // Customizable
			"nodeDescription": profileData.Bio, // Customizable
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

	apNote := GenerateAPNote(note, a.getBaseURL(r))
	// Ensure the ID in the JSON matches the canonical URL
	apNote["id"] = a.getBaseURL(r) + "/notes/" + idStr

	w.Header().Set("Content-Type", "application/activity+json; charset=utf-8")
	json.NewEncoder(w).Encode(apNote)
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

	sig, err := a.httpsigModel.GetByActor(actorIRI)
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

	log.Printf("Sending activity to %s. Headers: Digest=%s, Signature=%s", inbox, req.Header.Get("Digest"), req.Header.Get("Signature"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send activity: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("inbox %s returned status %d: %s", inbox, resp.StatusCode, string(body))
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
		case activitypub.LikeType:
			return a.handleLikeActivity(act)
		case activitypub.AnnounceType:
			return a.handleAnnounceActivity(act)
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
	acceptID := fmt.Sprintf("https://%s/activities/accept-%d", host, time.Now().UnixNano())
	accept := activitypub.Accept{
		ID:     activitypub.IRI(acceptID),
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
	// The object of an Undo activity is the Activity being undone.
	return activitypub.OnActivity(act.Object, func(innerAct *activitypub.Activity) error {
		switch innerAct.GetType() {
		case activitypub.FollowType:
			// For Undo Follow, we need the actor of the *Undo* activity (the person unfollowing)
			// But wait, existing logic used act.Actor.
			actor, err := a.resolveActor(act.Actor)
			if err != nil {
				return fmt.Errorf("handleUndoActivity: %w", err)
			}

			log.Printf("Inbox: Removing follower %s", actor.GetID())
			return a.followerModel.RemoveFollower(actor.GetID().String())

		case activitypub.LikeType:
			return a.handleUndoLike(innerAct)

		case activitypub.AnnounceType:
			return a.handleUndoAnnounce(innerAct)
		}
		return nil
	})
}

func (a *ActivityPubAPI) handleUndoLike(innerAct *activitypub.Activity) error {
	var uri string
	if innerAct.Object.IsLink() {
		uri = innerAct.Object.GetLink().String()
	} else {
		// If it's an object, we need to extract ID
		// innerAct.Object is Item (interface)
		// We can't access First() directly on Item if it's not a collection?
		// Actually innerAct.Object is Item, IsLink works.
		// To get ID if it's not a link, we need to cast or use GetID().
		if innerAct.Object != nil {
			uri = innerAct.Object.GetID().String()
		}
	}

	if uri == "" {
		log.Printf("Inbox: Could not determine Note URI for Undo Like")
		return nil
	}

	note, err := a.noteModel.GetByURI(uri)
	if err != nil {
		log.Printf("Inbox: Note %s not found for Undo Like", uri)
		return nil
	}

	log.Printf("Inbox: Decrementing likes for note %s", uri)
	return a.noteModel.DecrementLikes(note.ID)
}

func (a *ActivityPubAPI) handleUndoAnnounce(innerAct *activitypub.Activity) error {
	var uri string
	if innerAct.Object.IsLink() {
		uri = innerAct.Object.GetLink().String()
	} else {
		if innerAct.Object != nil {
			uri = innerAct.Object.GetID().String()
		}
	}

	if uri == "" {
		log.Printf("Inbox: Could not determine Note URI for Undo Announce")
		return nil
	}

	note, err := a.noteModel.GetByURI(uri)
	if err != nil {
		log.Printf("Inbox: Note %s not found for Undo Announce", uri)
		return nil
	}

	log.Printf("Inbox: Decrementing shares for note %s", uri)
	return a.noteModel.DecrementShares(note.ID)
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
		var to []string
		for _, item := range obj.To {
			to = append(to, item.GetLink().String())
		}
		var cc []string
		for _, item := range obj.CC {
			cc = append(cc, item.GetLink().String())
		}
		publicRange := DeterminePublicRange(to, cc)

		log.Printf("Inbox: Creating federated note from %s", obj.GetID())
		note := &db.Note{
			URI:          obj.GetID().String(),
			Content:      StripHTML(obj.Content.First().String()),
			AuthorFinger: authorFinger,
			Host:         authorHost,
			AuthorName:   authorName,
			PublicRange:  publicRange,
		}
		return a.noteModel.CreateFederatedNote(note)
	})
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
			Content: StripHTML(obj.Content.First().String()),
		}
		return a.noteModel.UpdateFederatedNote(note)
	})
}

// handleDeleteActivity processes Delete activities.
func (a *ActivityPubAPI) handleDeleteActivity(act *activitypub.Activity) error {
	var uri string
	if act.Object.IsLink() {
		uri = act.Object.GetLink().String()
	} else {
		err := activitypub.OnObject(act.Object, func(obj *activitypub.Object) error {
			uri = obj.GetID().String()
			return nil
		})
		if err != nil {
			return err
		}
	}

	if uri != "" {
		log.Printf("Inbox: Deleting federated object %s", uri)
		return a.noteModel.DeleteByURI(uri)
	}
	return nil
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
	if err := validateIRI(iri); err != nil {
		return nil, fmt.Errorf("fetchActor: invalid IRI: %w", err)
	}

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

func validateIRI(iri string) error {
	u, err := url.Parse(iri)
	if err != nil {
		return err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("unsupported scheme: %s", u.Scheme)
	}

	if os.Getenv("KNIFE_DEV_MODE") == "true" {
		return nil
	}

	hostname := u.Hostname()
	ips, err := net.LookupIP(hostname)
	if err != nil {
		return err
	}

	for _, ip := range ips {
		if ip.IsLoopback() || ip.IsPrivate() || ip.IsUnspecified() {
			return fmt.Errorf("resolves to private/loopback IP: %s", ip.String())
		}
	}

	return nil
}

func (a *ActivityPubAPI) handleLikeActivity(act *activitypub.Activity) error {
	var uri string
	if act.Object.IsLink() {
		uri = act.Object.GetLink().String()
	} else {
		err := activitypub.OnObject(act.Object, func(object *activitypub.Object) error {
			uri = object.GetID().String()
			return nil
		})
		if err != nil {
			return err
		}
	}

	if uri == "" {
		return fmt.Errorf("could not determine object URI for Like")
	}

	note, err := a.noteModel.GetByURI(uri)
	if err != nil {
		log.Printf("Inbox: Note %s not found for Like", uri)
		return nil
	}
	log.Printf("Inbox: Incrementing likes for note %s", uri)
	return a.noteModel.IncrementLikes(note.ID)
}

func (a *ActivityPubAPI) handleAnnounceActivity(act *activitypub.Activity) error {
	var uri string
	if act.Object.IsLink() {
		uri = act.Object.GetLink().String()
	} else {
		err := activitypub.OnObject(act.Object, func(object *activitypub.Object) error {
			uri = object.GetID().String()
			return nil
		})
		if err != nil {
			return err
		}
	}

	if uri == "" {
		return fmt.Errorf("could not determine object URI for Announce")
	}

	note, err := a.noteModel.GetByURI(uri)
	if err != nil {
		log.Printf("Inbox: Note %s not found for Announce", uri)
		return nil
	}
	log.Printf("Inbox: Incrementing shares for note %s", uri)
	return a.noteModel.IncrementShares(note.ID)
}
