package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"

	"golang.org/x/crypto/argon2"
)

// Configuration for Argon2id
// These settings make it computationally expensive to brute-force
const (
	ArgonTime    = 1         // Iterations
	ArgonMemory  = 64 * 1024 // 64MB memory usage
	ArgonThreads = 4         // Parallelism
	KeyLength    = 32        // AES-256 needs 32 bytes
	SaltLength   = 16        // standard salt size
	NonceLength  = 12        // standard AES-GCM nonce size
)

// Encrypt takes a password and plaintext, and returns a Base64 encoded secure string.
// Output Format: Base64( Salt + Nonce + Ciphertext )
func Encrypt(password, plaintext string) (string, error) {
	// 1. Generate a random Salt
	salt := make([]byte, SaltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return "", err
	}

	// 2. Derive the AES Key from the Password + Salt using Argon2id
	key := argon2.IDKey([]byte(password), salt, ArgonTime, ArgonMemory, ArgonThreads, KeyLength)

	// 3. Initialize AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 4. Generate a random Nonce
	nonce := make([]byte, NonceLength)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	// 5. Encrypt (Seal)
	// The first 'nil' is the destination buffer. We can append directly to nonce if we want to save allocation,
	// but here we keep them separate for clarity, then concat.
	ciphertext := aesGCM.Seal(nil, nonce, []byte(plaintext), nil)

	// 6. Pack everything: Salt + Nonce + Ciphertext
	finalData := make([]byte, len(salt)+len(nonce)+len(ciphertext))
	copy(finalData, salt)
	copy(finalData[len(salt):], nonce)
	copy(finalData[len(salt)+len(nonce):], ciphertext)

	// Return as Base64 string for easy storage/text compatibility
	return base64.StdEncoding.EncodeToString(finalData), nil
}

// Decrypt takes the password and the Base64 string, and returns the original text.
func Decrypt(password, encryptedData string) (string, error) {
	// 1. Decode Base64
	data, err := base64.StdEncoding.DecodeString(encryptedData)
	if err != nil {
		return "", err
	}

	// Check minimum length (Salt + Nonce + Tag)
	// Tag in GCM is usually 16 bytes.
	if len(data) < SaltLength+NonceLength+16 {
		return "", errors.New("invalid data length")
	}

	// 2. Extract Salt, Nonce, and Ciphertext
	salt := data[:SaltLength]
	nonce := data[SaltLength : SaltLength+NonceLength]
	ciphertext := data[SaltLength+NonceLength:]

	// 3. Re-derive the AES Key using the extracted Salt
	key := argon2.IDKey([]byte(password), salt, ArgonTime, ArgonMemory, ArgonThreads, KeyLength)

	// 4. Initialize AES-GCM
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	// 5. Decrypt (Open)
	plaintext, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		// This error occurs if the password is wrong OR if the data was tampered with
		return "", errors.New("decryption failed: wrong password or corrupted data")
	}

	return string(plaintext), nil
}
