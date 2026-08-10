package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"time"
)

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	TokenHash string    `json:"-"`
	ExpiresAt time.Time `json:"expires_at"`
	RevokedAt time.Time `json:"revoked_at,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func NewSession(userID string, ttl time.Duration) (Session, string, error) {
	token, err := randomToken(32)
	if err != nil {
		return Session{}, "", err
	}
	now := time.Now()
	return Session{ID: newID("sess"), UserID: userID, TokenHash: HashToken(token), ExpiresAt: now.Add(ttl), CreatedAt: now}, token, nil
}

func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func newID(prefix string) string {
	token, _ := randomToken(12)
	return prefix + "_" + token
}
