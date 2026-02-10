package server

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type AuthAPI struct {
	username string
	password string
	secret   string
}

func NewAuthAPI(username, password, secret string) *AuthAPI {
	return &AuthAPI{username: username, password: password, secret: secret}
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

func (api *AuthAPI) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "用户名和密码不能为空"})
		return
	}

	if req.Username != api.username || req.Password != api.password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户名或密码错误"})
		return
	}

	token := generateToken(api.secret, req.Username)
	c.JSON(http.StatusOK, gin.H{"token": token})
}

// generateToken creates a simple HMAC-based token: base64(username|expiry|signature)
func generateToken(secret, username string) string {
	expiry := time.Now().Add(24 * time.Hour).Unix()
	payload := fmt.Sprintf("%s|%d", username, expiry)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s|%s", payload, sig)))
}

// ValidateToken checks if a token is valid and not expired.
func ValidateToken(secret, token string) bool {
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return false
	}

	parts := splitLast(string(raw), "|")
	if len(parts) != 2 {
		return false
	}
	payload, sig := parts[0], parts[1]

	// Verify signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(payload))
	expectedSig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if sig != expectedSig {
		return false
	}

	// Check expiry
	var username string
	var expiry int64
	fmt.Sscanf(payload, "%s|%d", &username, &expiry)

	// Parse expiry from payload
	for i := len(payload) - 1; i >= 0; i-- {
		if payload[i] == '|' {
			fmt.Sscanf(payload[i+1:], "%d", &expiry)
			break
		}
	}

	return time.Now().Unix() < expiry
}

// splitLast splits s by the last occurrence of sep.
func splitLast(s, sep string) []string {
	for i := len(s) - 1; i >= 0; i-- {
		if string(s[i]) == sep {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}
