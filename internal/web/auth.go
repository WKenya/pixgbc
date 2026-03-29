package web

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const sessionCookieName = "pixgbc_session"

type sessionLoginRequest struct {
	Token string `json:"token"`
}

func (s *Server) handleSessionStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, SessionStatusResponse{
		AuthRequired:  s.cfg.Token != "",
		Authenticated: s.authorized(r),
	})
}

func (s *Server) handleSessionLogin(w http.ResponseWriter, r *http.Request) {
	if s.cfg.Token == "" {
		writeJSON(w, http.StatusOK, SessionStatusResponse{
			AuthRequired:  false,
			Authenticated: true,
		})
		return
	}

	token, err := readSessionLoginToken(r)
	if err != nil {
		http.Error(w, fmt.Sprintf("invalid login request: %v", err), http.StatusBadRequest)
		return
	}
	if token != s.cfg.Token {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	http.SetCookie(w, s.sessionCookie(r, time.Now()))
	writeJSON(w, http.StatusOK, SessionStatusResponse{
		AuthRequired:  true,
		Authenticated: true,
	})
}

func (s *Server) handleSessionLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
	})
	writeJSON(w, http.StatusOK, SessionStatusResponse{
		AuthRequired:  s.cfg.Token != "",
		Authenticated: false,
	})
}

func (s *Server) authorized(r *http.Request) bool {
	if s.cfg.Token == "" {
		return true
	}
	if s.requestTokenQuery(r) != "" {
		return true
	}
	authHeader := strings.TrimSpace(r.Header.Get("Authorization"))
	if strings.HasPrefix(authHeader, "Bearer ") && strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer ")) == s.cfg.Token {
		return true
	}
	return s.validSession(r, time.Now())
}

func (s *Server) requestTokenQuery(r *http.Request) string {
	if s.cfg.Token == "" {
		return ""
	}
	token := strings.TrimSpace(r.URL.Query().Get("token"))
	if token == s.cfg.Token {
		return token
	}
	return ""
}

func withTokenQuery(urlPath string, token string) string {
	if token == "" {
		return urlPath
	}
	return urlPath + "?token=" + token
}

func (s *Server) validSession(r *http.Request, now time.Time) bool {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return false
	}
	return validateSessionValue(cookie.Value, s.cfg.Token, now)
}

func (s *Server) sessionCookie(r *http.Request, now time.Time) *http.Cookie {
	return &http.Cookie{
		Name:     sessionCookieName,
		Value:    signSessionValue(s.cfg.Token, now.Add(s.cfg.SessionTTL)),
		Path:     "/",
		HttpOnly: true,
		MaxAge:   int(s.cfg.SessionTTL.Seconds()),
		SameSite: http.SameSiteLaxMode,
		Secure:   requestIsSecure(r),
	}
}

func readSessionLoginToken(r *http.Request) (string, error) {
	if strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "application/json") {
		defer r.Body.Close()
		var payload sessionLoginRequest
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			return "", err
		}
		return strings.TrimSpace(payload.Token), nil
	}
	if err := r.ParseForm(); err != nil {
		return "", err
	}
	return strings.TrimSpace(r.FormValue("token")), nil
}

func signSessionValue(secret string, expiresAt time.Time) string {
	payload := strconv.FormatInt(expiresAt.UTC().Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return payload + "." + hex.EncodeToString(mac.Sum(nil))
}

func validateSessionValue(value, secret string, now time.Time) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 2 {
		return false
	}
	expiresUnix, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return false
	}
	if now.UTC().After(time.Unix(expiresUnix, 0).UTC()) {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(parts[0]))
	want := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(parts[1]), []byte(want))
}

func requestIsSecure(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	switch strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))) {
	case "https", "wss":
		return true
	default:
		return false
	}
}
