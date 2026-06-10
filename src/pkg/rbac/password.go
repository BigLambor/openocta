package rbac

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

const (
	argon2Time    = 3
	argon2Memory  = 64 * 1024
	argon2Threads = 4
	argon2KeyLen  = 32
	argon2SaltLen = 16
)

// HashPasswordLegacy hashes plain password with SHA256 using salt (legacy format).
func HashPasswordLegacy(password, salt string) string {
	hasher := sha256.New()
	hasher.Write([]byte(password + salt))
	return hex.EncodeToString(hasher.Sum(nil))
}

// HashPassword is kept as an alias for legacy hashing (migration tests).
func HashPassword(password, salt string) string {
	return HashPasswordLegacy(password, salt)
}

// HashPasswordArgon2id returns an Argon2id PHC-encoded hash; salt column is left empty.
func HashPasswordArgon2id(password string) (passwordHash, salt string) {
	saltBytes := make([]byte, argon2SaltLen)
	if _, err := rand.Read(saltBytes); err != nil {
		panic(fmt.Sprintf("argon2 salt: %v", err))
	}
	return encodeArgon2id(password, saltBytes), ""
}

func encodeArgon2id(password string, salt []byte) string {
	key := argon2.IDKey([]byte(password), salt, argon2Time, argon2Memory, argon2Threads, argon2KeyLen)
	return fmt.Sprintf("$argon2id$v=19$m=%d,t=%d,p=%d$%s$%s",
		argon2Memory, argon2Time, argon2Threads,
		hex.EncodeToString(salt), hex.EncodeToString(key))
}

// IsArgon2Hash reports whether the stored hash uses Argon2id.
func IsArgon2Hash(storedHash string) bool {
	return strings.HasPrefix(storedHash, "$argon2id$")
}

// VerifyPassword checks legacy SHA256 or Argon2id hashes.
func VerifyPassword(password, storedHash, salt string) bool {
	if IsArgon2Hash(storedHash) {
		return verifyArgon2id(password, storedHash)
	}
	return subtle.ConstantTimeCompare([]byte(HashPasswordLegacy(password, salt)), []byte(storedHash)) == 1
}

func verifyArgon2id(password, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false
	}
	salt, err := hex.DecodeString(parts[4])
	if err != nil {
		return false
	}
	expectedKey, err := hex.DecodeString(parts[5])
	if err != nil {
		return false
	}
	actualKey := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(expectedKey)))
	return subtle.ConstantTimeCompare(actualKey, expectedKey) == 1
}
