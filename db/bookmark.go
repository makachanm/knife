package db

type Bookmark struct {
	ID     int64 `db:"id"`
	NoteID int64 `db:"note_id"`
}

type BookmarkModel struct {
	DB *DB
}

func NewBookmarkModel(db *DB) *BookmarkModel {
	return &BookmarkModel{DB: db}
}

func (m *BookmarkModel) Create(bookmark *Bookmark) error {
	query := `INSERT INTO bookmarks (note_id) VALUES (:note_id)`
	result, err := m.DB.NamedExec(query, bookmark)
	if err != nil {
		return err
	}
	id, err := result.LastInsertId()
	if err != nil {
		return err
	}
	bookmark.ID = id
	return nil
}

func (m *BookmarkModel) ListAll() ([]Bookmark, error) {
	var bookmarks []Bookmark
	query := `SELECT id, note_id FROM bookmarks`
	err := m.DB.Select(&bookmarks, query)
	return bookmarks, err
}

func (m *BookmarkModel) Delete(noteID int64) error {
	query := `DELETE FROM bookmarks WHERE note_id = ?`
	_, err := m.DB.Exec(query, noteID)
	return err
}
