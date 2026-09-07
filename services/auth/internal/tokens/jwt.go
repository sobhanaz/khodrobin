package tokens

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Access tokens are short-lived because they cannot be revoked: the only thing
// bounding a stolen one is its expiry. Fifteen minutes keeps that window small
// while the refresh token, which is revocable, carries the long session.
const (
	AccessTTL  = 15 * time.Minute
	RefreshTTL = 30 * 24 * time.Hour
)

var ErrInvalidToken = errors.New("token is invalid or expired")

type Claims struct {
	UserID   int64  `json:"uid"`
	Email    string `json:"email"`
	Verified bool   `json:"verified"`
	IsAdmin  bool   `json:"admin"`
	jwt.RegisteredClaims
}

type Issuer struct {
	secret []byte
	issuer string
}

func NewIssuer(secret, issuer string) *Issuer {
	return &Issuer{secret: []byte(secret), issuer: issuer}
}

func (i *Issuer) Access(userID int64, email string, verified, admin bool, now time.Time) (string, error) {
	claims := Claims{
		UserID: userID, Email: email, Verified: verified, IsAdmin: admin,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    i.issuer,
			Subject:   strconv.FormatInt(userID, 10),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(i.secret)
}

// Parse validates signature and expiry, and pins the algorithm.
//
// Pinning matters: without it a token claiming alg=none, or alg=HS256 against
// an RSA public key, is a well-known forgery route.
func (i *Issuer) Parse(raw string) (*Claims, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(raw, &claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method %v", t.Header["alg"])
		}
		return i.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer(i.issuer))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}
	return &claims, nil
}
