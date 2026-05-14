package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Mirnda/mirandaclin/pkg/validator"
)

type Config struct {
	App    App      `json:"app" validate:"required"`
	DB     DataBase `json:"db" validate:"required"`
	Redis  Redis    `json:"redis"` // optional — if empty, cache is disabled
	JWT    JWT      `json:"jwt" validate:"required"`
	Mailer SMTP     `json:"mailer"` // optional — if empty, mailer is logged
}

type App struct {
	SocialName string `json:"social_name" validate:"required"`
	Name       string `json:"name" validate:"required"`
	Env        string `json:"env" validate:"required,oneof=production staging development"`
	HostName   string `json:"host" validate:"required,url|hostname|ip"`
	Port       string `json:"port" validate:"required"`

	RateLimitEnabled   bool   `json:"rate_limit_enabled" validate:"required"`
	CORSAllowedOrigins string `json:"cors" validate:"required"`

	Frontend Client `json:"frontend" validate:"required"`
}

type Client struct {
	HostName            string `json:"host" validate:"required,url|hostname|ip"`
	VerifyUserEmailPath string `json:"verify_email_url_path" validate:"required"`
}

type DataBase struct {
	Driver  string `json:"driver" validate:"required,oneof=postgres"`
	Host    string `json:"host" validate:"required,hostname|ip"`
	Port    string `json:"port" validate:"required"`
	User    string `json:"user" validate:"required"`
	Pass    string `json:"pass" validate:"required"`
	Name    string `json:"name" validate:"required"`
	SSLMode string `json:"sslmode" validate:"required,oneof=disable require allow prefer verify-ca verify-full"`
}

type Redis struct {
	Addr     string `json:"addr"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

type JWT struct {
	Secret  string `json:"secret" validate:"required"`
	Issuer  string `json:"issuer" validate:"required"`
	JWKSUrl string `json:"jwksUrl" validate:"required,url"`
}

type SMTP struct {
	Host string `json:"host"`
	Port string `json:"port"`
	User string `json:"user"`
	Pass string `json:"pass"`
	From string `json:"from"`
}

func Load() (*Config, error) {
	redisDB, _ := strconv.Atoi(env("REDIS_DB", "0"))
	rateLimitEnabled, _ := strconv.ParseBool(env("RATE_LIMIT_ENABLED", "true"))

	cfg := Config{
		App: App{
			SocialName:         env("APP_SOCIAL_NAME", ""),
			Name:               env("APP_NAME", ""),
			Env:                env("APP_ENV", ""),
			HostName:           env("APP_HOST", ""),
			Port:               env("APP_PORT", "8080"),
			RateLimitEnabled:   rateLimitEnabled,
			CORSAllowedOrigins: env("CORS_ALLOWED_ORIGINS", ""),
			Frontend: Client{
				HostName:            env("CLIENT_HOSTNAME", ""),
				VerifyUserEmailPath: env("WEB_VERIFY_EMAIL_URL", ""),
			},
		},
		DB: DataBase{
			Driver:  env("DB_DRIVER", "postgres"),
			Host:    env("DB_HOST", ""),
			Port:    env("DB_PORT", ""),
			User:    env("DB_USER", ""),
			Pass:    env("DB_PASS", ""),
			Name:    env("DB_NAME", ""),
			SSLMode: env("DB_SSLMODE", "disable"),
		},
		Redis: Redis{
			Addr:     env("REDIS_ADDR", "localhost:6379"),
			Password: env("REDIS_PASSWORD", ""),
			DB:       redisDB,
		},
		JWT: JWT{
			Secret:  env("JWT_SECRET", ""),
			Issuer:  env("JWT_ISSUER", ""),
			JWKSUrl: env("JWT_JWKS_URL", ""),
		},
		Mailer: SMTP{
			Host: env("SMTP_HOST", ""),
			Port: env("SMTP_PORT", "587"),
			User: env("SMTP_USER", ""),
			Pass: env("SMTP_PASS", ""),
			From: env("SMTP_FROM", ""),
		},
	}

	if err := validator.Validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation returned error: %v", err)
	}

	return &cfg, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func (d *DataBase) DSN() string {
	switch d.Driver {
	case "postgres":
		return fmt.Sprintf(
			"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=America/Sao_Paulo",
			d.Host, d.Port, d.User, d.Pass, d.Name, d.SSLMode,
		)
	default:
		return ""
	}
}

func (a *App) ClientUrl() string {
	return a.Frontend.HostName

}

func (a *App) VerifyEmailUrl(token string) string {
	cliUrl := a.ClientUrl()
	if !strings.HasPrefix(a.Frontend.VerifyUserEmailPath, "/") {
		return fmt.Sprintf("%s/%s?token=%s", cliUrl, a.Frontend.VerifyUserEmailPath, token)
	}
	return fmt.Sprintf("%s%s?token=%s", cliUrl, a.Frontend.VerifyUserEmailPath, token)
}
