package spacetrackclient

import (
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"os"
	"time"

	"github.com/joho/godotenv"
	"golang.org/x/net/publicsuffix"
	"golang.org/x/time/rate"
)

type Client struct {
	client *http.Client
	auth   *Auth
}

func NewHttpClient() (*Client, error) {
	if err := godotenv.Load(); err != nil {
		return nil, err
	}

	jar, err := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})
	if err != nil {
		return nil, err
	}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{AddSource: true, Level: getLogLevelEnv().Slog()}))

	auth := &Auth{
		rawclient: &http.Client{Jar: jar, Transport: Chain(http.DefaultTransport, NewLog(logger))},
		baseURL:   "https://www.space-track.org",
		username:  os.Getenv("SPACETRACK_USERNAME"),
		password:  os.Getenv("SPACETRACK_PASSWORD"),
		logger:    logger,
	}

	limiter := rate.NewLimiter(rate.Every(time.Minute/5), 1)

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
	}, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if err := c.auth.ensureAuth(req.Context()); err != nil {
		return nil, err
	}

	return c.client.Do(req)
}
