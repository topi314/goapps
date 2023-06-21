package goapps

import (
	"context"
	"errors"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/exp/slog"
	"golang.org/x/oauth2"
	"net/http"
	"sync"
	"time"
)

type authKey struct{}

var UserInfoKey = authKey{}

type Auth struct {
	Verifier *oidc.IDTokenVerifier
	Config   *oauth2.Config
	Provider *oidc.Provider
	// session id <-> id token
	Sessions   map[string]*Session
	SessionsMu sync.Mutex
	// state <-> nonce
	States   map[string]string
	StatesMu sync.Mutex
}

type Session struct {
	AccessToken  string
	Expiry       time.Time
	RefreshToken string
	IDToken      string
}

type UserInfo struct {
	Subject  string   `json:"sub"`
	Profile  string   `json:"profile"`
	Email    string   `json:"email"`
	Audience []string `json:"aud"`
	Groups   []string `json:"groups"`
	Username string   `json:"preferred_username"`
}

const SessionCookieName = "X-Session-ID"

func (s *Server) setSession(w http.ResponseWriter, sessionID string, session *Session) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    sessionID,
		Path:     "/",
		Secure:   s.cfg.Auth.Secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	s.auth.SessionsMu.Lock()
	defer s.auth.SessionsMu.Unlock()
	s.auth.Sessions[sessionID] = session
}

func (s *Server) removeSession(w http.ResponseWriter, sessionID string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Path:     "/",
		MaxAge:   -1,
		Secure:   s.cfg.Auth.Secure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})

	s.auth.SessionsMu.Lock()
	defer s.auth.SessionsMu.Unlock()
	delete(s.auth.Sessions, sessionID)
}

func (s *Server) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sessionID string
		if cookie, err := r.Cookie(SessionCookieName); err == nil {
			sessionID = cookie.Value
		}

		if sessionID == "" {
			next.ServeHTTP(w, r)
			return
		}

		s.auth.SessionsMu.Lock()
		session, ok := s.auth.Sessions[sessionID]
		s.auth.SessionsMu.Unlock()
		if !ok {
			slog.Debug("session not found", slog.Any("sessionID", sessionID))
			s.removeSession(w, sessionID)
			next.ServeHTTP(w, r)
			return
		}

		tokenSource := s.auth.Config.TokenSource(r.Context(), &oauth2.Token{
			AccessToken:  session.AccessToken,
			TokenType:    "bearer",
			RefreshToken: session.RefreshToken,
			Expiry:       session.Expiry,
		})

		token, err := tokenSource.Token()
		if err != nil {
			slog.Error("failed to get token", slog.Any("err", err))
			s.removeSession(w, sessionID)
			next.ServeHTTP(w, r)
			return
		}

		if token.AccessToken != session.AccessToken {
			session.AccessToken = token.AccessToken
			session.Expiry = token.Expiry
			session.RefreshToken = token.RefreshToken
			session.IDToken = token.Extra("id_token").(string)
		}

		idToken, err := s.auth.Verifier.Verify(r.Context(), session.IDToken)
		if err != nil {
			slog.Error("failed to verify ID Token: %w", slog.Any("err", err), slog.Any("rawIDToken", session.IDToken))
			s.removeSession(w, sessionID)
			next.ServeHTTP(w, r)
			return
		}

		var info UserInfo
		if err = idToken.Claims(&info); err != nil {
			slog.Error("failed to parse claims: %w", slog.Any("err", err))
			s.removeSession(w, sessionID)
			s.error(w, r, err, http.StatusInternalServerError)
			return
		}

		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), UserInfoKey, &info)))
	})
}

func (s *Server) CheckAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUserInfo(r) == nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func GetUserInfo(r *http.Request) *UserInfo {
	userInfo := r.Context().Value(UserInfoKey)
	if userInfo == nil {
		return nil
	}
	return userInfo.(*UserInfo)
}

func (s *Server) Login(w http.ResponseWriter, r *http.Request) {
	state := s.newID(16)
	nonce := s.newID(16)
	s.auth.States[state] = nonce
	http.Redirect(w, r, s.auth.Config.AuthCodeURL(state, oidc.Nonce(nonce)), http.StatusFound)
}

func (s *Server) Logout(w http.ResponseWriter, r *http.Request) {
	sessionID, err := r.Cookie("X-Session-ID")
	if err == nil {
		s.removeSession(w, sessionID.Value)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func (s *Server) Callback(w http.ResponseWriter, r *http.Request) {
	nonce, ok := s.auth.States[r.URL.Query().Get("state")]
	if !ok {
		s.error(w, r, errors.New("invalid state"), http.StatusBadRequest)
		return
	}

	token, err := s.auth.Config.Exchange(r.Context(), r.URL.Query().Get("code"))
	if err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
		return
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		s.error(w, r, errors.New("no id_token in token response"), http.StatusInternalServerError)
		return
	}
	idToken, err := s.auth.Verifier.Verify(r.Context(), rawIDToken)
	if err != nil {
		s.error(w, r, fmt.Errorf("failed to verify ID Token: %w", err), http.StatusInternalServerError)
		return
	}

	if idToken.Nonce != nonce {
		s.error(w, r, errors.New("invalid nonce"), http.StatusBadRequest)
		return
	}
	sessionID := s.newID(32)
	slog.Debug("new session", slog.Any("sessionID", sessionID), slog.Any("idToken", rawIDToken), slog.Any("token", token))
	s.setSession(w, sessionID, &Session{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		Expiry:       token.Expiry,
		IDToken:      rawIDToken,
	})
	http.Redirect(w, r, "/", http.StatusFound)
}
