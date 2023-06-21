package goapps

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5/middleware"
	"golang.org/x/exp/slog"
	"io"
	"log"
	"math/rand"
	"net/http"
	"runtime"
	"strings"
	"time"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

type ExecuteTemplateFunc func(w io.Writer, name string, data any) error

func NewServer(version string, cfg Config, auth *Auth, assets http.FileSystem, icons http.FileSystem, tmpl ExecuteTemplateFunc) *Server {
	s := &Server{
		version: version,
		cfg:     cfg,
		auth:    auth,
		assets:  assets,
		icons:   icons,
		tmpl:    tmpl,
		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	s.server = &http.Server{
		Addr:    cfg.Server.ListenAddr,
		Handler: s.Routes(),
	}

	return s
}

type Server struct {
	version string
	cfg     Config
	server  *http.Server
	auth    *Auth
	assets  http.FileSystem
	icons   http.FileSystem
	tmpl    ExecuteTemplateFunc
	rand    *rand.Rand
}

func (s *Server) Start() {
	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		slog.Error("Error while listening", slog.Any("err", err))
	}
}

func (s *Server) Close() {
	if err := s.server.Close(); err != nil {
		slog.Error("Error while closing server", slog.Any("err", err))
	}
}

func (s *Server) newID(length int) string {
	b := make([]rune, length)
	for i := range b {
		b[i] = letters[s.rand.Intn(len(letters))]
	}
	return string(b)
}

func (s *Server) log(r *http.Request, logType string, err error) {
	if errors.Is(err, context.DeadlineExceeded) {
		return
	}
	log.Printf("Error while handling %s(%s) %s: %s\n", logType, middleware.GetReqID(r.Context()), r.RequestURI, err)
}

func (s *Server) error(w http.ResponseWriter, r *http.Request, err error, status int) {
	if status == http.StatusInternalServerError {
		s.log(r, "pretty request", err)
	}
	w.WriteHeader(status)

	vars := map[string]any{
		"Error":     err.Error(),
		"Status":    status,
		"RequestID": middleware.GetReqID(r.Context()),
		"Path":      r.URL.Path,
		"Theme":     "dark",
	}
	if tmplErr := s.tmpl(w, "error.gohtml", vars); tmplErr != nil {
		s.log(r, "template", tmplErr)
	}
}

func (s *Server) file(path string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, err := s.assets.Open(path)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()
		_, _ = io.Copy(w, file)
	}
}

func cacheControl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/assets/") {
			w.Header().Set("Cache-Control", "public, max-age=86400")
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
		next.ServeHTTP(w, r)
	})
}

func FormatBuildVersion(version string, commit string, buildTime time.Time) string {
	if len(commit) > 7 {
		commit = commit[:7]
	}

	buildTimeStr := "unknown"
	if !buildTime.IsZero() {
		buildTimeStr = buildTime.Format(time.ANSIC)
	}
	return fmt.Sprintf("Go Version: %s\nVersion: %s\nCommit: %s\nBuild Time: %s\nOS/Arch: %s/%s\n", runtime.Version(), version, commit, buildTimeStr, runtime.GOOS, runtime.GOARCH)
}
