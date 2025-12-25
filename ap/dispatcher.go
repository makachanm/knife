package ap

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"log"
	"net/http"
	"net/url"
	"time"

	"knife/base"
	"knife/db"

	"github.com/go-fed/httpsig"
)

type ActivityDispatcher struct {
	followerModel *db.FollowerModel
	httpsigModel  *db.HTTPSigModel
	jobQueue      *base.JobQueue
}

func NewActivityDispatcher(followerModel *db.FollowerModel, httpsigModel *db.HTTPSigModel, jobQueue *base.JobQueue) *ActivityDispatcher {
	return &ActivityDispatcher{
		followerModel: followerModel,
		httpsigModel:  httpsigModel,
		jobQueue:      jobQueue,
	}
}

// SendCreateNote dispatches a Create activity for a Note to all followers.
func (d *ActivityDispatcher) SendCreateNote(note *db.Note) error {
	followers, err := d.followerModel.ListFollowers()
	if err != nil {
		log.Printf("failed to list followers: %v", err)
		return err
	}

	actorURI := "https://" + note.Host + "/profile"
	apNote := map[string]interface{}{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           note.URI,
		"type":         "Note",
		"published":    note.CreateTime.Format("2006-01-02T15:04:05Z"),
		"attributedTo": actorURI,
		"content":      note.Content,
		"to":           []string{"https://www.w3.org/ns/activitystreams#Public"},
	}

	if note.Cw != "" { 
		apNote["sensitive"] = true
		apNote["summary"] = note.Cw
	}

	activity := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type":     "Create",
		"actor":    actorURI,
		"object":   apNote,
	}
	activityBytes, err := json.Marshal(activity)
	if err != nil {
		log.Printf("failed to marshal activity: %v", err)
		return err
	}

	for _, follower := range followers {
		follower := follower // Create a new variable for the closure
		job := func() {
			d.sendActivityToFollower(follower, activityBytes, actorURI)
		}
		d.jobQueue.Enqueue(job)
	}

	return nil
}

// SendDeleteNote dispatches a Delete activity for a Note to all followers.
func (d *ActivityDispatcher) SendDeleteNote(note *db.Note) error {
	followers, err := d.followerModel.ListFollowers()
	if err != nil {
		log.Printf("failed to list followers: %v", err)
		return err
	}

	actorURI := "https://" + note.Host + "/profile"
	activity := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"type":     "Delete",
		"actor":    actorURI,
		"object":   note.URI,
	}
	activityBytes, err := json.Marshal(activity)
	if err != nil {
		log.Printf("failed to marshal activity: %v", err)
		return err
	}

	for _, follower := range followers {
		follower := follower // Create a new variable for the closure
		job := func() {
			d.sendActivityToFollower(follower, activityBytes, actorURI)
		}
		d.jobQueue.Enqueue(job)
	}

	return nil
}

func (d *ActivityDispatcher) sendActivityToFollower(follower db.Follower, activityBytes []byte, actorURI string) {
	req, err := http.NewRequest("POST", follower.InboxURI, bytes.NewBuffer(activityBytes))
	if err != nil {
		log.Printf("failed to create request for follower %s: %v", follower.ActorURI, err)
		return
	}
	req.Header.Set("Content-Type", "application/activity+json")
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	inboxURL, err := url.Parse(follower.InboxURI)
	if err != nil {
		log.Printf("failed to parse inbox url for follower %s: %v", follower.ActorURI, err)
		return
	}
	req.Header.Set("Host", inboxURL.Host)

	sig, err := d.httpsigModel.GetByActor(actorURI)
	if err != nil {
		log.Printf("failed to get httpsig for %s: %v", actorURI, err)
		return
	}

	block, _ := pem.Decode([]byte(sig.PrivateKey))
	if block == nil {
		log.Printf("failed to decode private key for %s", actorURI)
		return
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		log.Printf("failed to parse private key for %s: %v", actorURI, err)
		return
	}

	// Sign the request
	keyID := actorURI + "#main-key"
	prefs := []httpsig.Algorithm{httpsig.RSA_SHA256}
	digestAlgo := httpsig.DigestSha256
	headersToSign := []string{httpsig.RequestTarget, "host", "date", "digest"}
	signer, _, err := httpsig.NewSigner(prefs, digestAlgo, headersToSign, httpsig.Signature, 65535)
	if err != nil {
		log.Printf("failed to create signer for %s: %v", actorURI, err)
		return
	}
	if err := signer.SignRequest(privateKey, keyID, req, activityBytes); err != nil {
		log.Printf("failed to sign request for %s: %v", follower.ActorURI, err)
		return
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send activity to follower %s: %v", follower.ActorURI, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		log.Printf("follower %s returned status %d", follower.ActorURI, resp.StatusCode)
	} else {
		log.Printf("Successfully sent activity to follower %s", follower.ActorURI)
	}
}
