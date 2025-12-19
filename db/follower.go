package db

type Follower struct {
	ActorURI   string `db:"actor_uri"`
	InboxURI   string `db:"inbox_uri"`
	FollowedAt string `db:"followed_at"`
}

type FollowerModel struct {
	db *DB
}

func NewFollowerModel(db *DB) *FollowerModel {
	return &FollowerModel{db: db}
}

func (m *FollowerModel) AddFollower(actorURI, inboxURI string) error {
	_, err := m.db.Exec("INSERT INTO followers (actor_uri, inbox_uri) VALUES (?, ?)", actorURI, inboxURI)
	return err
}

func (m *FollowerModel) RemoveFollower(actorURI string) error {
	_, err := m.db.Exec("DELETE FROM followers WHERE actor_uri = ?", actorURI)
	return err
}

func (m *FollowerModel) ListFollowers() ([]Follower, error) {
	var followers []Follower
	err := m.db.Select(&followers, "SELECT * FROM followers ORDER BY followed_at DESC")
	return followers, err
}
