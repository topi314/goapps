package main

import (
	"context"
	"crypto/md5"
	"embed"
	"flag"
	"fmt"
	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/topi314/goboard/goboard"
	"golang.org/x/exp/slog"
	"golang.org/x/oauth2"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// These variables are set via the -ldflags option in go build
var (
	Version   = "unknown"
	Commit    = "unknown"
	BuildTime = "unknown"
)

var (
	//go:embed templates
	Templates embed.FS

	//go:embed assets
	Assets embed.FS
)

func main() {
	cfgPath := flag.String("config", "goboard.yml", "path to goboard.yml config file")
	flag.Parse()

	cfg, err := goboard.LoadConfig(*cfgPath)
	if err != nil {
		slog.Error("Failed to load config", slog.Any("err", err))
		os.Exit(1)
	}

	setupLogger(cfg.Log)
	slog.Info("Starting homepage", slog.Any("cfg", cfg))

	var auth *goboard.Auth
	if cfg.Auth != nil {
		provider, err := oidc.NewProvider(context.Background(), cfg.Auth.Issuer)
		if err != nil {
			log.Fatalln("Error while creating Auth provider:", err)
		}

		auth = &goboard.Auth{
			Provider: provider,
			Verifier: provider.Verifier(&oidc.Config{
				ClientID: cfg.Auth.ClientID,
			}),
			Config: &oauth2.Config{
				ClientID:     cfg.Auth.ClientID,
				ClientSecret: cfg.Auth.ClientSecret,
				Endpoint:     provider.Endpoint(),
				RedirectURL:  cfg.Auth.RedirectURL,
				Scopes:       []string{oidc.ScopeOpenID, "groups", "email", "profile", oidc.ScopeOfflineAccess},
			},
			Sessions: map[string]*goboard.Session{},
			States:   map[string]string{},
		}
	}

	funcs := template.FuncMap{
		"gravatarURL": func(email string) string {
			return fmt.Sprintf("https://www.gravatar.com/avatar/%x?s=%d&d=retro", md5.Sum([]byte(strings.ToLower(email))), 80)
		},
	}

	var (
		tmplFunc goboard.ExecuteTemplateFunc
		assets   http.FileSystem
	)
	if cfg.DevMode {
		slog.Info("Development mode enabled")
		tmplFunc = func(wr io.Writer, name string, data any) error {
			tmpl, err := template.New("").Funcs(funcs).ParseGlob("templates/*")
			if err != nil {
				return err
			}
			return tmpl.ExecuteTemplate(wr, name, data)
		}
		assets = http.Dir(".")
	} else {
		tmpl, err := template.New("").Funcs(funcs).ParseFS(Templates, "templates/*")
		if err != nil {
			slog.Error("Error while parsing templates", slog.Any("err", err))
			os.Exit(1)
		}
		tmplFunc = tmpl.ExecuteTemplate
		assets = http.FS(Assets)
	}

	buildTime, _ := time.Parse(time.RFC3339, BuildTime)
	s := goboard.NewServer(goboard.FormatBuildVersion(Version, Commit, buildTime), cfg, auth, assets, tmplFunc)
	go s.Start()
	defer s.Close()

	si := make(chan os.Signal, 1)
	signal.Notify(si, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	slog.Info("Started homepage", slog.String("addr", cfg.Server.ListenAddr))
	<-si
}

func setupLogger(cfg goboard.LogConfig) {
	opts := &slog.HandlerOptions{
		AddSource: cfg.AddSource,
		Level:     cfg.Level,
	}
	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	slog.SetDefault(slog.New(handler))
}
