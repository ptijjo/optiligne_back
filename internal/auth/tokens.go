package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"strings"
	"time"
)

// Claims est le payload JWT access / refresh.
type Claims struct {
	Sub          string `json:"sub"`
	Email        string `json:"email,omitempty"`
	OperatorCode string `json:"operatorCode,omitempty"`
	DepotCode    string `json:"depotCode,omitempty"`
	Typ          string `json:"typ"`
	JTI          string `json:"jti,omitempty"`
	Exp          int64  `json:"exp"`
	Iat          int64  `json:"iat"`
}

func signHS256(secret []byte, c Claims) (string, error) {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))
	payload, err := json.Marshal(c)
	if err != nil {
		return "", err
	}
	body := header + "." + base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(body))
	return body + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}

func parseHS256(secret []byte, token string) (Claims, error) {
	var zero Claims
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return zero, ErrInvalidToken
	}
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	want := mac.Sum(nil)
	got, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || !hmac.Equal(want, got) {
		return zero, ErrInvalidToken
	}
	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return zero, ErrInvalidToken
	}
	var c Claims
	if err := json.Unmarshal(raw, &c); err != nil {
		return zero, ErrInvalidToken
	}
	if c.Exp > 0 && time.Now().Unix() >= c.Exp {
		return zero, ErrInvalidToken
	}
	return c, nil
}

func hashToken(raw string) string {
	sum := sha256.Sum256([]byte(raw))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}
