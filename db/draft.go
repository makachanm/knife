package db

import (
	"time"
)

type Draft struct {
	ID         int64     `db:"id" json:"id"`
	Content    string    `db:"content" json:"content"`
	CreateTime time.Time `db:"create_time" json:"create_time"`
	UpdateTime time.Time `db:"update_time" json:"update_time"`
}

type DraftModel struct {
	DB *DB
}

func NewDraftModel(db *DB) *DraftModel {
	return &DraftModel{DB: db}
}

// Save creates or updates a draft.
func (m *DraftModel) Save(draft *Draft) error {
	// If ID is 0, it's a new draft
	if draft.ID == 0 {
		query := `INSERT INTO drafts (content, create_time, update_time) VALUES (?, ?, ?)`
		now := time.Now()
		res, err := m.DB.Exec(query, draft.Content, now, now)
		if err != nil {
			return err
		}
		id, err := res.LastInsertId()
		if err != nil {
			return err
		}
		draft.ID = id
		draft.CreateTime = now
		draft.UpdateTime = now
		return nil
	}

	// Otherwise, update existing draft
	query := `UPDATE drafts SET content = ?, update_time = ? WHERE id = ?`
	now := time.Now()
	_, err := m.DB.Exec(query, draft.Content, now, draft.ID)
	if err == nil {
		draft.UpdateTime = now
	}
	return err
}

func (m *DraftModel) Get(id int64) (*Draft, error) {
	var draft Draft
	query := "SELECT * FROM drafts WHERE id = ?"
	err := m.DB.Get(&draft, query, id)
	return &draft, err
}

func (m *DraftModel) List() ([]Draft, error) {
	var drafts []Draft
	query := "SELECT * FROM drafts ORDER BY update_time DESC"
	err := m.DB.Select(&drafts, query)
	return drafts, err
}

func (m *DraftModel) Delete(id int64) error {
	query := "DELETE FROM drafts WHERE id = ?"
	_, err := m.DB.Exec(query, id)
	return err
}
