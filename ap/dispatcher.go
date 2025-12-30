package ap

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
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

func (d *ActivityDispatcher) getProtocol() string {
	if proto := os.Getenv("KNIFE_PROTOCOL"); proto != "" {
		return proto
	}
	return "https"
}

func (d *ActivityDispatcher) getHost(fallback string) string {
	if host := os.Getenv("KNIFE_HOST"); host != "" {
		return host
	}
	return fallback
}

func (d *ActivityDispatcher) getBaseURL(fallbackHost string) string {
	return fmt.Sprintf("%s://%s", d.getProtocol(), d.getHost(fallbackHost))
}

// SendCreateNote dispatches a Create activity for a Note to all followers.
func (d *ActivityDispatcher) SendCreateNote(note *db.Note) error {
	followers, err := d.followerModel.ListFollowers()
	if err != nil {
		log.Printf("failed to list followers: %v", err)
		return err
	}

	baseURL := d.getBaseURL(note.Host)
	actorURI := baseURL + "/profile"
	apNote := GenerateAPNote(note, baseURL)
	// Ensure the ID in the activity matches the canonical URL
	apNote["id"] = baseURL + "/notes/" + fmt.Sprintf("%d", note.ID)

	activity := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       note.URI,//fmt.Sprintf("%s/activities/create-%d", baseURL, note.ID),
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

	baseURL := d.getBaseURL(note.Host)
	actorURI := baseURL + "/profile"
	activity := map[string]interface{}{
		"@context": "https://www.w3.org/ns/activitystreams",
		"id":       note.URI,//fmt.Sprintf("%s/activities/delete-%d-%d", baseURL, note.ID, time.Now().Unix()),
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
	req.Host = inboxURL.Host

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

	log.Printf("Sending activity to %s. Headers: Digest=%s, Signature=%s", follower.ActorURI, req.Header.Get("Digest"), req.Header.Get("Signature"))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("failed to send activity to follower %s: %v", follower.ActorURI, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		// Read body for error details
		b, _ := io.ReadAll(resp.Body)
		log.Printf("follower %s returned status %d: %s", follower.ActorURI, resp.StatusCode, string(b))
	} else {
		log.Printf("Successfully sent activity to follower %s", follower.ActorURI)
	}
}
