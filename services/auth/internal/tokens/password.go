// Package tokens holds the credential primitives: password hashing, opaque
// secrets, and JWT issuance.
package tokens

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters.
//
// argon2id rather than bcrypt because it resists GPU attack through memory
// cost, which bcrypt's fixed 4 KB working set does not. 64 MB and one pass over
// four lanes is the OWASP-recommended floor and costs roughly 50 ms on this
// box — slow enough to make offline cracking expensive, fast enough that a
// login does not feel broken.
const (
	argonTime    = 1
	argonMemory  = 64 * 1024 // KiB
	argonThreads = 4
	argonKeyLen  = 32
	saltLen      = 16
)

var ErrBadHashFormat = errors.New("password hash is not in the expected format")

// HashPassword returns a self-describing PHC string, so the parameters travel
// with the hash and can be raised later without invalidating existing users.
func HashPassword(password string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("read salt: %w", err)
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		base64.RawStdEncoding.EncodeToString(salt),
		base64.RawStdEncoding.EncodeToString(key),
	), nil
}

// VerifyPassword re-derives the key with the stored parameters and compares in
// constant time, so a wrong password cannot be found by timing the reply.
func VerifyPassword(password, encoded string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, ErrBadHashFormat
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, ErrBadHashFormat
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false, ErrBadHashFormat
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[5])
	if err != nil {
		return false, ErrBadHashFormat
	}
	got := argon2.IDKey([]byte(password), salt, time, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}

// Secret returns a URL-safe random string for email links and refresh tokens.
//
// 32 bytes from crypto/rand: these travel in emails and cookies, so guessing
// one must be infeasible rather than merely unlikely.
func Secret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("read secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// Fingerprint is what gets stored for a secret we hand out.
//
// SHA-256 rather than argon2 on purpose: these are 256-bit random values, not
// human-chosen passwords, so there is no dictionary to slow down — and a
// refresh check on every request cannot afford 50 ms. A database dump still
// yields no usable token.
func Fingerprint(secret string) string {
	sum := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(sum[:])
}
