package main

import (
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"time"

	"golang.org/x/net/publicsuffix"
	"golang.org/x/time/rate"
)

type Client struct {
	client *http.Client
	auth   *Auth
}

func NewHttpClient() *Client {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: getLogLevelEnv().Slog()}))

	auth := &Auth{
		rawclient: &http.Client{Jar: jar, Transport: Chain(http.DefaultTransport, NewLog(logger))},
		baseURL:   "https://www.space-track.org",
		username:  os.Getenv("SPACETRACK_USERNAME"),
		password:  os.Getenv("SPACETRACK_PASSWORD"),
		logger:    logger,
	}

	limiter := rate.NewLimiter(rate.Every(time.Minute/2), 1)

	return &Client{
		client: &http.Client{
			Jar: jar,
			Transport: Chain(
				http.DefaultTransport,
				NewRateLimiter(limiter),
				NewLog(logger),
				// NewAuth(auth),
			),
		},
		auth: auth,
	}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if err := c.auth.ensureAuth(req.Context()); err != nil {
		return nil, err
	}

	return c.client.Do(req)
}

type LogLevel string

const (
	LevelDebug LogLevel = "DEBUG"
	LevelInfo  LogLevel = "INFO"
	LevelWarn  LogLevel = "WARN"
	LevelError LogLevel = "ERROR"
)

func (l LogLevel) Slog() slog.Level {
	switch l {
	case LevelDebug:
		return slog.LevelDebug
	case LevelInfo:
		return slog.LevelInfo
	case LevelWarn:
		return slog.LevelWarn
	case LevelError:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

func (l LogLevel) IsValid() bool {
	switch l {
	case LevelDebug, LevelError, LevelWarn, LevelInfo:
		return true
	default:
		return false
	}
}

func getLogLevelEnv() LogLevel {
	loglevel := os.Getenv("LOG_LEVEL")
	loglevel = strings.ToUpper(strings.TrimSpace(loglevel))

	if loglevel == "" {
		return LevelInfo
	}

	if !LogLevel(loglevel).IsValid() {
		return LevelInfo
	}

	return LogLevel(loglevel)
}
