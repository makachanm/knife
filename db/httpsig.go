package db

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

type HTTPSig struct {
	ID         int64  `db:"id"`
	Actor      string `db:"actor"`
	PublicKey  string `db:"public_key"`
	PrivateKey string `db:"private_key"`
}

type HTTPSigModel struct {
	DB *DB
}

func NewHTTPSigModel(db *DB) *HTTPSigModel {
	return &HTTPSigModel{DB: db}
}

func (m *HTTPSigModel) generateRSAKeyPair() (string, string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", "", fmt.Errorf("failed to generate RSA key pair: %w", err)
	}

	publicKey := &privateKey.PublicKey

	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})

	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal public key: %w", err)
	}
	publicKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})

	return string(publicKeyPEM), string(privateKeyPEM), nil
}

func (m *HTTPSigModel) Create(actor string) (*HTTPSig, error) {
	publicKey, privateKey, err := m.generateRSAKeyPair()
	if err != nil {
		return nil, err
	}

	sig := &HTTPSig{
		Actor:      actor,
		PublicKey:  publicKey,
		PrivateKey: privateKey,
	}

	query := `
		INSERT INTO httpsigs (actor, public_key, private_key)
		VALUES (:actor, :public_key, :private_key)
	`
	result, err := m.DB.NamedExec(query, sig)
	if err != nil {
		return nil, fmt.Errorf("failed to insert httpsig: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("failed to get last insert id: %w", err)
	}
	sig.ID = id

	return sig, nil
}

func (m *HTTPSigModel) GetByActor(actor string) (*HTTPSig, error) {
	var sig HTTPSig
	query := "SELECT * FROM httpsigs WHERE actor = ?"
	err := m.DB.Get(&sig, query, actor)
	if err != nil {
		return nil, err
	}
	return &sig, nil
}
