package passport

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type Passport struct {
	UserID    string   `json:"uid"`
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	IssuedAt  int64    `json:"iat"`
	ExpiresAt int64    `json:"exp"`
}

const (
	HeaderName = "X-Internal-Passport"
)

func SignPassport(passport Passport, secret []byte) string {
	// 1. JSON encode
	jsonBytes, _ := json.Marshal(passport)

	// 2. Base64 encode the JSON
	payload := base64.URLEncoding.EncodeToString(jsonBytes)

	// 3. Create HMAC-SHA256 signature
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	signature := base64.URLEncoding.EncodeToString(mac.Sum(nil))

	// 4. Return payload.signature
	return payload + "." + signature
}

func VerifyPassport(tokenString string, secret []byte) (*Passport, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 2 {
		return nil, fmt.Errorf("bad format")
	}

	payload, sig := parts[0], parts[1]

	// Verify signature
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))
	expectedSig := base64.URLEncoding.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(sig), []byte(expectedSig)) {
		return nil, fmt.Errorf("bad signature")
	}

	// Decode payload
	jsonBytes, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return nil, err
	}

	var p Passport
	if err := json.Unmarshal(jsonBytes, &p); err != nil {
		return nil, err
	}

	// Check expiry
	if time.Now().Unix() > p.ExpiresAt {
		return nil, fmt.Errorf("expired")
	}

	return &p, nil
}
