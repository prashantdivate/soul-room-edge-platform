package identity

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileSignerPermissionsAndPersistence(t *testing.T) {
	dir := t.TempDir()
	s1, err := LoadOrCreateFileSigner(dir)
	if err != nil {
		t.Fatal(err)
	}
	fp1, _ := s1.Fingerprint()
	s2, err := LoadOrCreateFileSigner(dir)
	if err != nil {
		t.Fatal(err)
	}
	fp2, _ := s2.Fingerprint()
	if fp1 != fp2 {
		t.Fatal("fingerprint changed after reload")
	}
	info, err := os.Stat(filepath.Join(dir, "device_key.pem"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("private key mode = %o", info.Mode().Perm())
	}
}
