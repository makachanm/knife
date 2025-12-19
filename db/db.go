package db

import (
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
)

// DB is a wrapper around sqlx.DB
type DB struct {
	*sqlx.DB
}

// InitDB initializes the database connection and creates tables
func InitDB(dataSourceName string) (*DB, error) {
	db, err := sqlx.Connect("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	if _, err := db.Exec(schemaNotes); err != nil {
		return nil, err
	}

	if _, err := db.Exec(schemaProfiles); err != nil {
		return nil, err
	}

	if _, err := db.Exec(schemaBookmarks); err != nil {
		return nil, err
	}

	if _, err := db.Exec(schemaFollowers); err != nil {
		return nil, err
	}

	return &DB{db}, nil
}

const schemaFollowers = `
CREATE TABLE IF NOT EXISTS followers (
    actor_uri TEXT PRIMARY KEY,
    inbox_uri TEXT NOT NULL,
    followed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`

const schemaNotes = `
CREATE TABLE IF NOT EXISTS notes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uri TEXT UNIQUE,
    cw TEXT NOT NULL,
    content TEXT NOT NULL,
    host TEXT NOT NULL,
    author_name TEXT NOT NULL,
    medias TEXT NOT NULL,
    public_range INTEGER NOT NULL,
    create_time DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
	author_finger TEXT NOT NULL
);`

const schemaProfiles = `
CREATE TABLE IF NOT EXISTS profile (
    finger TEXT PRIMARY KEY,
    password_hash TEXT NOT NULL,
    display_name TEXT NOT NULL,
    avatar_url TEXT NOT NULL,
    bio TEXT NOT NULL
);`

const schemaBookmarks = `
CREATE TABLE IF NOT EXISTS bookmarks (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	note_id INTEGER NOT NULL,
	created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);`
