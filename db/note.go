package db

import (
	"fmt"
	"time"
)

type NotePublicRange int

const (
	NotePublicRangePrivate NotePublicRange = iota
	NotePublicRangeFollowers
	NotePublicRangeUnlisted
	NotePublicRangePublic
)

type Note struct {
	ID           int64           `db:"id" json:"id"`
	URI          string          `db:"uri" json:"uri"`
	Cw           string          `db:"cw" json:"cw,omitempty"`
	Content      string          `db:"content" json:"content"`
	Host         string          `db:"host" json:"host"`
	AuthorName   string          `db:"author_name" json:"author_name"`
	AuthorFinger string          `db:"author_finger" json:"author_finger"`
	PublicRange  NotePublicRange `db:"public_range" json:"public_range,string"`
	CreateTime   time.Time       `db:"create_time" json:"create_time"`
	Category     string          `db:"category" json:"category,omitempty"`
}

type NoteModel struct {
	DB *DB
}

func NewNoteModel(db *DB) *NoteModel {
	return &NoteModel{DB: db}
}

// CreateFederatedNote creates a note that already has a URI (e.g., from ActivityPub).
func (m *NoteModel) CreateFederatedNote(note *Note) error {
	query := `
		INSERT INTO notes (uri, cw, content, host, author_name, public_range, author_finger, category)
		VALUES (:uri, :cw, :content, :host, :author_name, :public_range, :author_finger, :category)
	`

	_, err := m.DB.NamedExec(query, note)
	if err != nil {
		return err
	}

	return nil
}

// CreateLocalNote creates a note originating from the local instance.
// It inserts the note, gets the ID, and then constructs the URI.
func (m *NoteModel) CreateLocalNote(note *Note) error {
	tx, err := m.DB.Beginx()
	if err != nil {
		return err
	}
	defer tx.Rollback() // Rollback on error

	// Insert the note without the URI
	query := `
		INSERT INTO notes (cw, content, host, author_name, public_range, author_finger, category)
		VALUES (:cw, :content, :host, :author_name, :public_range, :author_finger, :category)
	`
	result, err := tx.NamedExec(query, note)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	note.ID = id

	// Generate the URI and update the note
	// Note: The note.Host must be set by the caller (API layer)
	note.URI = fmt.Sprintf("https://%s/notes/%d", note.Host, note.ID)
	updateQuery := "UPDATE notes SET uri = ? WHERE id = ?"
	if _, err := tx.Exec(updateQuery, note.URI, note.ID); err != nil {
		return err
	}

	return tx.Commit()
}

func (m *NoteModel) Get(id int64) (*Note, error) {
	var note Note
	query := "SELECT id, uri, cw, content, host, author_name, author_finger, public_range, create_time, category FROM notes WHERE id = ?"
	err := m.DB.Get(&note, query, id)
	return &note, err
}

// Update allows modifying a note's content. It also allows setting the URI.
func (m *NoteModel) Update(note *Note) error {
	query := `
		UPDATE notes
		SET uri = :uri, cw = :cw, content = :content, public_range = :public_range, category = :category
		WHERE id = :id
	`
	_, err := m.DB.NamedExec(query, note)
	return err
}

func (m *NoteModel) UpdateFederatedNote(note *Note) error {
	query := "UPDATE notes SET content = ? WHERE uri = ?"
	_, err := m.DB.Exec(query, note.Content, note.URI)
	return err
}

func (m *NoteModel) Delete(id int64) error {
	query := "DELETE FROM notes WHERE id = ?"
	_, err := m.DB.Exec(query, id)
	return err
}

func (m *NoteModel) DeleteByURI(uri string) error {
	query := "DELETE FROM notes WHERE uri = ?"
	_, err := m.DB.Exec(query, uri)
	return err
}

func (m *NoteModel) ListRecent() ([]Note, error) {
	var notes []Note
	query := "SELECT id, uri, cw, content, host, author_name, author_finger, public_range, create_time, category FROM notes ORDER BY create_time DESC LIMIT 100"
	err := m.DB.Select(&notes, query)
	return notes, err
}

func (m *NoteModel) ListByMyRecent() ([]Note, error) {
	var notes []Note

	fquery := `SELECT finger FROM profile LIMIT 1`
	var myFinger string
	err := m.DB.Get(&myFinger, fquery)
	if err != nil {
		return nil, err
	}

	query := "SELECT id, uri, cw, content, host, author_name, author_finger, public_range, create_time, category FROM notes WHERE author_finger = ? ORDER BY create_time DESC LIMIT 100"
	err = m.DB.Select(&notes, query, myFinger)
	return notes, err
}

func (m *NoteModel) ListCategories() ([]string, error) {
	var categories []string
	query := "SELECT DISTINCT category FROM notes WHERE category != '' AND category IS NOT NULL ORDER BY category ASC"
	err := m.DB.Select(&categories, query)
	return categories, err
}

func (m *NoteModel) ListByCategory(category string) ([]Note, error) {
	var notes []Note
	query := "SELECT id, uri, cw, content, host, author_name, author_finger, public_range, create_time, category FROM notes WHERE category = ? ORDER BY create_time DESC LIMIT 100"
	err := m.DB.Select(&notes, query, category)
	return notes, err
}
