package ap

import (
	"knife/db"
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// GenerateAPNote constructs a map representing an ActivityPub Note object.
func GenerateAPNote(note *db.Note, baseURL string) map[string]interface{} {
	to, cc := GetVisibilityTargets(note, baseURL)

	apNote := map[string]interface{}{
		"@context":     "https://www.w3.org/ns/activitystreams",
		"id":           note.URI,
		"type":         "Note",
		"published":    note.CreateTime.Format("2006-01-02T15:04:05Z"),
		"attributedTo": baseURL + "/profile",
		"content":      note.Content,
		"to":           to,
		"cc":           cc,
	}

	if note.Cw != "" {
		apNote["sensitive"] = true
		apNote["summary"] = note.Cw
	}

	return apNote
}

// GetVisibilityTargets determines the "to" and "cc" fields based on the note's visibility.
func GetVisibilityTargets(note *db.Note, baseURL string) ([]string, []string) {
	var to []string
	var cc []string

	switch note.PublicRange {
	case db.NotePublicRangePublic:
		to = []string{"https://www.w3.org/ns/activitystreams#Public"}
	case db.NotePublicRangeFollowers:
		to = []string{}
		cc = []string{baseURL + "/followers"}
	case db.NotePublicRangeUnlisted:
		to = []string{}
		cc = []string{"https://www.w3.org/ns/activitystreams#Public"}
	case db.NotePublicRangePrivate:
		to = []string{baseURL + "/profile"}
	default:
		// Default to private if the range is unknown
		to = []string{baseURL + "/profile"}
	}

	return to, cc
}

// DeterminePublicRange determines the public range of a note based on its "to" and "cc" fields.
func DeterminePublicRange(to, cc []string) db.NotePublicRange {
	// Check if the note is public
	for _, item := range append(to, cc...) {
		if item == "https://www.w3.org/ns/activitystreams#Public" {
			return db.NotePublicRangePublic
		}
	}

	// Unlisted check (usually in CC)
	for _, item := range cc {
		if item == "https://www.w3.org/ns/activitystreams#Public" {
			return db.NotePublicRangeUnlisted
		}
	}

	// Followers check - this is a bit simplified
	// In a real implementation, you'd check if the followers URI is in 'to' or 'cc'

	// Default to private
	return db.NotePublicRangePrivate
}

// StripHTML takes a string that contains HTML and returns just the text content.
func StripHTML(s string) string {
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
