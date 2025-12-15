package db

type Profile struct {
	Finger      string `db:"finger" json:"finger"`
	DisplayName string `db:"display_name" json:"display_name"`
	AvatarURL   string `db:"avatar_url" json:"avatar_url"`
	Bio         string `db:"bio" json:"bio"`
}

type ProfileModel struct {
	DB *DB
}

func NewProfileModel(db *DB) *ProfileModel {
	return &ProfileModel{DB: db}
}

func (m *ProfileModel) Create(profile *Profile) error {
	query := `
		INSERT INTO profiles (finger, display_name, avatar_url, bio)
		VALUES (:finger, :display_name, :avatar_url, :bio)
	`
	_, err := m.DB.NamedExec(query, profile)
	return err
}

func (m *ProfileModel) Get() (*Profile, error) {
	var profile Profile
	query := "SELECT finger, display_name, avatar_url, bio FROM profiles LIMIT 1"
	err := m.DB.Get(&profile, query)
	return &profile, err
}

func (m *ProfileModel) Update(profile *Profile) error {
	query := `
		UPDATE profiles
		SET display_name = :display_name, avatar_url = :avatar_url, bio = :bio
		WHERE finger = :finger
	`
	_, err := m.DB.NamedExec(query, profile)
	return err
}
