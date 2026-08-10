package auth

import "testing"

func TestPasswordHashVerify(t *testing.T) {
	hasher := PBKDF2Hasher{}
	hash, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatal(err)
	}
	if !hasher.Verify("correct horse battery staple", hash) {
		t.Fatal("expected password to verify")
	}
	if hasher.Verify("wrong", hash) {
		t.Fatal("wrong password verified")
	}
}
