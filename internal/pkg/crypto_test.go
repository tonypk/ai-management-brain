package pkg_test

import (
	"crypto/rand"
	"testing"

	"github.com/tonypk/ai-management-brain/internal/pkg"
)

func testKey() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	key := testKey()
	plaintext := "sk-ant-api03-secret-key"

	ciphertext, err := pkg.Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if ciphertext == plaintext {
		t.Fatal("ciphertext should differ from plaintext")
	}

	result, err := pkg.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if result != plaintext {
		t.Errorf("got %q, want %q", result, plaintext)
	}
}

func TestDecrypt_WrongKey(t *testing.T) {
	key1 := testKey()
	key2 := testKey()

	ciphertext, _ := pkg.Encrypt("secret", key1)
	_, err := pkg.Decrypt(ciphertext, key2)
	if err == nil {
		t.Fatal("should fail with wrong key")
	}
}

func TestEncrypt_DifferentNonce(t *testing.T) {
	key := testKey()
	c1, _ := pkg.Encrypt("same", key)
	c2, _ := pkg.Encrypt("same", key)
	if c1 == c2 {
		t.Fatal("same plaintext should produce different ciphertext (random nonce)")
	}
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	key := testKey()
	ciphertext, err := pkg.Encrypt("", key)
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}
	result, err := pkg.Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}
	if result != "" {
		t.Errorf("got %q, want empty", result)
	}
}
