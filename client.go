package main

import (
	"net/http"
	"net/http/cookiejar"
	"os"
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

	auth := &Auth{
		rawclient: &http.Client{Jar: jar, Transport: http.DefaultTransport},
		baseURL:   "https://www.space-track.org",
		username:  os.Getenv("SPACETRACK_USERNAME"),
		password:  os.Getenv("SPACETRACK_PASSWORD"),
	}

	limiter := rate.NewLimiter(rate.Every(time.Minute/2), 1)

	return &Client{
		client: &http.Client{
			Jar: jar,
			Transport: Chain(
				http.DefaultTransport,
				NewRateLimiter(limiter),
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
