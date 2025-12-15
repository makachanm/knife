package db

import (
	"time"
)

type Bookmark struct {
	ID        int64     `db:"id" json:"id"`
	NoteID    int64     `db:"note_id" json:"note_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

type BookmarkModel struct {
	DB *DB
}

func NewBookmarkModel(db *DB) *BookmarkModel {
	return &BookmarkModel{DB: db}
}

func (m *BookmarkModel) Create(bookmark *Bookmark) error {
	query := `
		INSERT INTO bookmarks (note_id)
		VALUES (:note_id)
	`
	_, err := m.DB.NamedExec(query, bookmark)
	return err
}

func (m *BookmarkModel) List() ([]*Bookmark, error) {
	var bookmarks []*Bookmark
	query := `
		SELECT id, note_id, created_at
		FROM bookmarks
		ORDER BY created_at DESC
	`
	err := m.DB.Select(&bookmarks, query)
	return bookmarks, err
}

func (m *BookmarkModel) Delete(noteID int64) error {
	query := `
		DELETE FROM bookmarks
		WHERE note_id = ?
	`
	_, err := m.DB.Exec(query, noteID)
	return err
}
