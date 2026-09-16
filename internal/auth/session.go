// Package auth gates the admin web UI behind a single admin login (username
// + password configured on carol-ops itself — same shared-secret posture as
// carol-bot's carolSharedSecret, not a multi-user account system) and an
// HMAC-signed session cookie.
package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const CookieName = "carol_ops_session"
const sessionTTL = 12 * time.Hour

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrInvalidSession = errors.New("invalid or expired session")

type Authenticator struct {
	username      string
	password      string
	sessionSecret []byte
}

func New(username, password, sessionSecret string) *Authenticator {
	return &Authenticator{username: username, password: password, sessionSecret: []byte(sessionSecret)}
}

// Verify checks the given credentials in constant time to avoid leaking
// match-length via timing.
func (a *Authenticator) Verify(username, password string) error {
	userOK := subtle.ConstantTimeCompare([]byte(username), []byte(a.username)) == 1
	passOK := subtle.ConstantTimeCompare([]byte(password), []byte(a.password)) == 1
	if !userOK || !passOK {
		return ErrInvalidCredentials
	}
	return nil
}

// IssueCookie signs a session token (username + expiry) and returns a cookie
// to set on the login response.
func (a *Authenticator) IssueCookie() *http.Cookie {
	expires := time.Now().Add(sessionTTL)
	token := a.sign(a.username, expires.Unix())
	return &http.Cookie{
		Name:     CookieName,
		Value:    token,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
	}
}

func (a *Authenticator) ClearCookie() *http.Cookie {
	return &http.Cookie{Name: CookieName, Path: "/", MaxAge: -1}
}

// ValidateRequest checks the session cookie on r, if any.
func (a *Authenticator) ValidateRequest(r *http.Request) error {
	cookie, err := r.Cookie(CookieName)
	if err != nil {
		return ErrInvalidSession
	}
	return a.validate(cookie.Value)
}

func (a *Authenticator) sign(username string, expiresUnix int64) string {
	payload := fmt.Sprintf("%s|%d", username, expiresUnix)
	mac := hmac.New(sha256.New, a.sessionSecret)
	mac.Write([]byte(payload))
	sig := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func (a *Authenticator) validate(token string) error {
	parts := strings.SplitN(token, ".", 2)
	if len(parts) != 2 {
		return ErrInvalidSession
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ErrInvalidSession
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ErrInvalidSession
	}
	mac := hmac.New(sha256.New, a.sessionSecret)
	mac.Write(payloadRaw)
	expectedSig := mac.Sum(nil)
	if !hmac.Equal(sig, expectedSig) {
		return ErrInvalidSession
	}
	username, expiresStr, found := strings.Cut(string(payloadRaw), "|")
	if !found {
		return ErrInvalidSession
	}
	if subtle.ConstantTimeCompare([]byte(username), []byte(a.username)) != 1 {
		return ErrInvalidSession
	}
	expiresUnix, err := strconv.ParseInt(expiresStr, 10, 64)
	if err != nil || time.Now().Unix() > expiresUnix {
		return ErrInvalidSession
	}
	return nil
}

// Middleware rejects any request without a valid session cookie.
func (a *Authenticator) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := a.ValidateRequest(r); err != nil {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
