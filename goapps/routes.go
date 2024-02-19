package goapps

import (
	"crypto/md5"
	"fmt"
	"log"
	"net/http"
	"slices"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func (s *Server) Routes() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.CleanPath)
	r.Use(middleware.StripSlashes)
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestID)
	r.Use(middleware.Maybe(
		middleware.RequestLogger(&middleware.DefaultLogFormatter{
			Logger:  log.Default(),
			NoColor: true,
		}),
		func(r *http.Request) bool {
			// Don't log requests for assets
			return !strings.HasPrefix(r.URL.Path, "/assets") && !strings.HasPrefix(r.URL.Path, "/icons")
		},
	))
	r.Use(cacheControl)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/ping"))

	if s.cfg.Debug {
		r.Mount("/debug", middleware.Profiler())
	}

	r.Mount("/assets", http.FileServer(s.assets))
	r.With(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, "/icons")
			next.ServeHTTP(w, r)
		})
	}).Mount("/", http.FileServer(s.icons))
	r.Handle("/favicon.ico", s.file("/assets/favicon.png"))
	r.Handle("/favicon.png", s.file("/assets/favicon.png"))
	r.Handle("/favicon-light.png", s.file("/assets/favicon-light.png"))
	r.Handle("/robots.txt", s.file("/assets/robots.txt"))

	r.Get("/version", s.GetVersion)

	r.Group(func(r chi.Router) {
		if s.cfg.Auth != nil {
			r.Use(s.Auth)
			r.Group(func(r chi.Router) {
				r.Get("/login", s.Login)
				r.Get("/callback", s.Callback)
				r.Get("/logout", s.Logout)
			})
		}

		r.Group(func(r chi.Router) {
			if s.cfg.Auth != nil {
				r.Use(s.CheckAuth)
			}
			r.Get("/", s.GetServices)
			r.Head("/", s.GetServices)
		})
	})
	r.NotFound(s.redirectRoot)

	return r
}

func (s *Server) GetVersion(w http.ResponseWriter, _ *http.Request) {
	_, _ = w.Write([]byte(s.version))
}

func (s *Server) redirectRoot(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

type TemplateVars struct {
	Theme    string
	Title    string
	Icon     string
	User     User
	Services []ServiceConfig
}

type User struct {
	Name      string
	Email     string
	AvatarURL string
}

func gravatarURL(email string) string {
	return fmt.Sprintf("https://www.gravatar.com/avatar/%x?s=%d&d=retro", md5.Sum([]byte(strings.ToLower(email))), 80)
}

func (s *Server) GetServices(w http.ResponseWriter, r *http.Request) {
	vars := TemplateVars{
		Theme: "dark",
		Title: s.cfg.Server.Title,
		Icon:  s.cfg.Server.Icon,
	}

	userInfo := GetUserInfo(r)
	if userInfo != nil {
		vars.User = User{
			Name:  userInfo.Username,
			Email: userInfo.Email,
		}

		for _, service := range s.cfg.Services {
			if slices.Contains(service.Users, userInfo.Username) {
				vars.Services = append(vars.Services, service)
				continue
			}

			for _, group := range service.Groups {
				if slices.Contains(userInfo.Groups, group) {
					vars.Services = append(vars.Services, service)
					continue
				}
			}

			if len(service.Users) == 0 && len(service.Groups) == 0 {
				vars.Services = append(vars.Services, service)
			}
		}
	} else {
		vars.Services = s.cfg.Services
	}

	vars.User.AvatarURL = gravatarURL(userInfo.Email)

	if err := s.tmpl(w, "index.gohtml", vars); err != nil {
		s.error(w, r, err, http.StatusInternalServerError)
	}
}
