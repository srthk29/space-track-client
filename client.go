package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/publicsuffix"
)

type Client struct {
	httpClient *http.Client
	baseURL    string

	username string
	password string

	mu        sync.Mutex
	expiresAt time.Time
}

func NewHttpClient() *Client {
	jar, _ := cookiejar.New(&cookiejar.Options{PublicSuffixList: publicsuffix.List})

	return &Client{
		httpClient: &http.Client{Jar: jar},
		baseURL:    "https://www.space-track.org",
		username:   os.Getenv("SPACETRACK_USERNAME"),
		password:   os.Getenv("SPACETRACK_PASSWORD"),
	}
}

/*
	{
	  "logged_in": true,
	  "identity": "upsidedown29@protonmail.com",
	  "session_expiration": "2026-04-08T20:53:49+00:00"
	}
*/
type Lifetime struct {
	LoggedIn bool      `json:"logged_in"`
	Identity string    `json:"identity"`
	ExpireAt time.Time `json:"session_expiration"`
}

func (c *Client) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/app/data/whoami",
		nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session extend failed: %s", resp.Status)
	}

	lifetime := &Lifetime{}
	if err := json.NewDecoder(resp.Body).Decode(lifetime); err != nil {
		return err
	}

	fmt.Printf("%#v", lifetime)

	if lifetime.LoggedIn {
		c.expiresAt = lifetime.ExpireAt.UTC()

		return nil
	}

	return fmt.Errorf("session expired")
}

func (c *Client) login(ctx context.Context) error {
	reqBody := strings.NewReader(fmt.Sprintf(
		"identity=%s&password=%s",
		c.username, c.password,
	))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		c.baseURL+"/ajaxauth/login",
		reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %s", resp.Status)
	}

	for _, cookie := range resp.Cookies() {
		c.expiresAt = cookie.Expires.UTC()
		fmt.Printf("%#v", cookie)
	}

	if c.expiresAt.IsZero() {
		// assume cookie TTL = 2 hours
		c.expiresAt = time.Now().Add(2 * time.Hour)
	}

	return nil
}

func (c *Client) logout(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		c.baseURL+"/ajaxauth/logout",
		nil)
	if err != nil {
		return err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("logout failed: %s", resp.Status)
	}

	return nil
}

func (c *Client) ensureAuth(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.expiresAt.IsZero() {
		return c.login(ctx)
	}

	// okay if > 30 minutes
	if time.Now().UTC().Before(c.expiresAt.Add(-30 * time.Minute)) {
		return nil
	}

	// refresh slightly before expiry (buffer)
	if err := c.refresh(ctx); err == nil {
		return nil
	}

	// session expired
	// login
	return c.login(ctx)
}

func (c *Client) Do(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, err
	}

	return c.httpClient.Do(req)
}

func (c *Client) DoRetry(ctx context.Context, req *http.Request) (*http.Response, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode == http.StatusUnauthorized {
		resp.Body.Close()

		// force re-login
		c.mu.Lock()
		c.expiresAt = time.Time{}
		c.mu.Unlock()

		if err := c.ensureAuth(ctx); err != nil {
			return nil, err
		}

		return c.httpClient.Do(req)
	}

	return resp, nil
}
