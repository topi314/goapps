package goboard

import (
	"fmt"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
)

var defaultConfig = Config{
	Log: LogConfig{
		Level:     slog.LevelInfo,
		Format:    "json",
		AddSource: false,
	},
	DevMode: false,
	Debug:   false,
	Server: ServerConfig{
		ListenAddr: ":8080",
		Title:      "homepage",
		Icon:       "",
		IconsDir:   "/var/lib/goboard/icons",
	},
	Auth:     nil,
	Services: nil,
}

func LoadConfig(path string) (Config, error) {
	cfg := defaultConfig
	file, err := os.Open(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed to open config: %w", err)
	}
	if err = yaml.NewDecoder(file).Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("failed to decode config: %w", err)
	}
	return cfg, nil
}

type Config struct {
	Log      LogConfig      `yaml:"log"`
	DevMode  bool           `yaml:"dev_mode"`
	Debug    bool           `yaml:"debug"`
	Server   ServerConfig   `yaml:"server"`
	Auth     *AuthConfig    `yaml:"auth"`
	Services ServicesConfig `yaml:"services"`
}

func (c Config) String() string {
	return fmt.Sprintf("\n Log: %s\n DevMode: %t\n Server: %s\n Debug: %t\n Auth: %s\n",
		c.Log,
		c.DevMode,
		c.Server,
		c.Debug,
		c.Auth,
	)
}

type LogConfig struct {
	Level     slog.Level `yaml:"level"`
	Format    string     `yaml:"format"`
	AddSource bool       `yaml:"add_source"`
}

func (c LogConfig) String() string {
	return fmt.Sprintf("\n  Level: %s\n  Format: %s\n  AddSource: %t\n",
		c.Level,
		c.Format,
		c.AddSource,
	)
}

type ServerConfig struct {
	ListenAddr string `yaml:"listen_addr"`
	Title      string `yaml:"title"`
	Icon       string `yaml:"icon"`
	IconsDir   string `yaml:"icons_dir"`
}

func (c ServerConfig) String() string {
	return fmt.Sprintf("\n  ListenAddr: %s\n  Title: %s\n  Icon: %s\n  IconsDir: %s\n",
		c.ListenAddr,
		c.Title,
		c.Icon,
		c.IconsDir,
	)
}

type AuthConfig struct {
	Secure       bool   `yaml:"secure"`
	Issuer       string `yaml:"issuer"`
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
	RedirectURL  string `yaml:"redirect_url"`
}

func (c AuthConfig) String() string {
	return fmt.Sprintf("\n  Secure: %t\n  Issuer: %s\n  ClientID: %s\n  ClientSecret: %s\n  RedirectURL: %s",
		c.Secure,
		c.Issuer,
		c.ClientID,
		strings.Repeat("*", len(c.ClientSecret)),
		c.RedirectURL,
	)
}

type ServicesConfig []ServiceConfig

func (s ServicesConfig) String() string {
	var str string
	for _, service := range s {
		str += fmt.Sprintf("\n  %s", service)
	}
	return str
}

type ServiceConfig struct {
	Name        string   `yaml:"name"`
	Description string   `yaml:"description"`
	Icon        string   `yaml:"icon"`
	URL         string   `yaml:"url"`
	Groups      []string `yaml:"groups"`
	Users       []string `yaml:"users"`
}

func (s ServiceConfig) String() string {
	return fmt.Sprintf("\n   Name: %s\n   Description: %s\n   Icon: %s\n   URL: %s\n   Groups: %s\n   Users: %s",
		s.Name,
		s.Description,
		s.Icon,
		s.URL,
		s.Groups,
		s.Users,
	)
}
