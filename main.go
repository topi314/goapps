package main

import (
	"context"
	"embed"
	"flag"
	"html/template"
	"io"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/topi314/goapps/goapps"
	"golang.org/x/oauth2"
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
	cfgPath := flag.String("config", "goapps.yml", "path to goapps.yml config file")
	flag.Parse()

	cfg, err := goapps.LoadConfig(*cfgPath)
	if err != nil {
		slog.Error("Failed to load config", slog.Any("err", err))
		os.Exit(1)
	}

	setupLogger(cfg.Log)
	slog.Info("Starting homepage", slog.Any("cfg", cfg))

	var auth *goapps.Auth
	if cfg.Auth != nil {
		provider, err := oidc.NewProvider(context.Background(), cfg.Auth.Issuer)
		if err != nil {
			log.Fatalln("Error while creating Auth provider:", err)
		}

		auth = &goapps.Auth{
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
			Sessions:  map[string]*goapps.Session{},
			States:    map[string]string{},
			Verifiers: map[string]string{},
		}
	}

	var (
		tmplFunc goapps.ExecuteTemplateFunc
		assets   http.FileSystem
	)
	if cfg.DevMode {
		slog.Info("Development mode enabled")
		tmplFunc = func(wr io.Writer, name string, data any) error {
			tmpl, err := template.New("").ParseGlob("templates/*")
			if err != nil {
				return err
			}
			return tmpl.ExecuteTemplate(wr, name, data)
		}
		assets = http.Dir(".")
	} else {
		tmpl, err := template.New("").ParseFS(Templates, "templates/*")
		if err != nil {
			slog.Error("Error while parsing templates", slog.Any("err", err))
			os.Exit(1)
		}
		tmplFunc = tmpl.ExecuteTemplate
		assets = http.FS(Assets)
	}

	icons := http.Dir(cfg.Server.IconsDir)

	buildTime, _ := time.Parse(time.RFC3339, BuildTime)
	s := goapps.NewServer(goapps.FormatBuildVersion(Version, Commit, buildTime), cfg, auth, assets, icons, tmplFunc)
	go s.Start()
	defer s.Close()

	si := make(chan os.Signal, 1)
	signal.Notify(si, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	slog.Info("Started homepage", slog.String("addr", cfg.Server.ListenAddr))
	<-si
}

func setupLogger(cfg goapps.LogConfig) {
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
