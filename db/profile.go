package db

type Profile struct {
	Finger       string `db:"finger" json:"finger"`
	DisplayName  string `db:"display_name" json:"display_name"`
	AvatarURL    string `db:"avatar_url" json:"avatar_url"`
	Bio          string `db:"bio" json:"bio"`
	PasswordHash string `db:"password_hash" json:"-"`
}

type ProfileModel struct {
	DB *DB
}

func NewProfileModel(db *DB) *ProfileModel {
	return &ProfileModel{DB: db}
}

func (m *ProfileModel) Create(profile *Profile) error {
	query := `
        INSERT INTO profile (finger, display_name, avatar_url, bio, password_hash)
        VALUES (:finger, :display_name, :avatar_url, :bio, :password_hash)
    `
	_, err := m.DB.NamedExec(query, profile)
	return err
}

func (m *ProfileModel) Get() (*Profile, error) {
	var profile Profile
	query := `SELECT finger, display_name, avatar_url, bio, password_hash FROM profile LIMIT 1`
	err := m.DB.Get(&profile, query)
	return &profile, err
}

func (m *ProfileModel) Update(profile *Profile) error {
	query := `
        UPDATE profile
        SET display_name = :display_name,
            avatar_url = :avatar_url,
            bio = :bio
        WHERE finger = :finger
    `
	_, err := m.DB.NamedExec(query, profile)
	return err
}

func (m *ProfileModel) CountProfiles() (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM profile`
	err := m.DB.Get(&count, query)
	return count, err
}
