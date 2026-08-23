package webapi

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"
	"strings"
	"sync"
	"time"
)

const sessionCookieName = "dockflow_session"

type credentialChecker func(username, password string) error

type sessionStore struct {
	mu       sync.Mutex
	sessions map[string]session
	check    credentialChecker
	now      func() time.Time
	ttl      time.Duration
}

type session struct {
	username string
	expires  time.Time
}

func newSessionStore(check credentialChecker) *sessionStore {
	return &sessionStore{sessions: make(map[string]session), check: check, now: time.Now, ttl: 12 * time.Hour}
}

func (s *sessionStore) login(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := decodeJSON(w, r, &in); err != nil {
		return
	}
	in.Username = strings.TrimSpace(in.Username)
	if in.Username == "" || in.Password == "" {
		writeBadRequest(w, "username and password are required")
		return
	}
	if err := s.check(in.Username, in.Password); err != nil {
		// Do not expose PAM details or whether the account exists.
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "用户名或密码错误"})
		return
	}
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "无法创建登录会话"})
		return
	}
	token := base64.RawURLEncoding.EncodeToString(tokenBytes)
	expires := s.now().Add(s.ttl)
	s.mu.Lock()
	s.sessions[token] = session{username: in.Username, expires: expires}
	s.mu.Unlock()
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: token, Path: "/", Expires: expires, MaxAge: int(s.ttl.Seconds()), HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: requestIsHTTPS(r)})
	writeJSON(w, http.StatusOK, map[string]string{"username": in.Username})
}

func (s *sessionStore) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(sessionCookieName); err == nil {
		s.mu.Lock()
		delete(s.sessions, cookie.Value)
		s.mu.Unlock()
	}
	http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteStrictMode, Secure: requestIsHTTPS(r)})
	w.WriteHeader(http.StatusNoContent)
}

func (s *sessionStore) current(r *http.Request) (string, bool) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}
	now := s.now()
	s.mu.Lock()
	defer s.mu.Unlock()
	value, ok := s.sessions[cookie.Value]
	if !ok {
		return "", false
	}
	if !value.expires.After(now) {
		delete(s.sessions, cookie.Value)
		return "", false
	}
	return value.username, true
}

func (s *sessionStore) require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, ok := s.current(r); !ok {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "请先登录"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func requestIsHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(strings.Split(r.Header.Get("X-Forwarded-Proto"), ",")[0]), "https")
}
