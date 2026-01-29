package jwtAuthentication

import (
	"time"

	"github.com/bwmarrin/snowflake"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

//NOTE: bcrypt.DefaultCost  // use default cost of bcrypt otherwise it will slow down

var JwtSecret = []byte("discore")

type TokenType string

const (
	AccessToken  TokenType = "access"
	RefreshToken TokenType = "refresh"
)

const (
	AccessTokenValidity  time.Duration = 24 * time.Hour
	RefreshTokenValidity time.Duration = 7 * 24 * time.Hour
)

// JWT Claims with Email
// Claims is the struct that contains the email of the user
// Email is the email of the user
// RegisteredClaims is the struct that contains the registered claims of the token
// ExpiresAt is the expiration time of the token
type JwtClaims struct {
	Email  string       `json:"email"`
	UserId snowflake.ID `json:"user_id"`
	jwt.RegisteredClaims
}

// Helper - Hash password
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// Helper - Check password
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// Generate JWT tokens
func GenerateToken(email string, userId snowflake.ID, duration time.Duration, tokenType TokenType) (string, error) {
	// Create claims
	// If we didn't use the pointer, the changes we make to the struct would not be reflected in the returned value.
	claims := &JwtClaims{
		Email:  email,
		UserId: userId,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duration)),
			Subject:   string(tokenType),
		},
	}
	// Create a new token with the claims
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	// signed the token with the secret key and return it
	return token.SignedString(JwtSecret)
}
