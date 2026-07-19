package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Auth struct {
	// for login, logout, extend session
	rawclient *http.Client

	baseURL string

	username string
	password string

	mu        sync.Mutex
	expiresAt time.Time
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

func (a *Auth) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		a.baseURL+"/app/data/whoami",
		nil)
	if err != nil {
		return err
	}

	resp, err := a.rawclient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("session extend failed: %s", resp.Status)
	}

	var lifetime Lifetime
	if err := json.NewDecoder(resp.Body).Decode(&lifetime); err != nil {
		return err
	}

	fmt.Printf("%#v", lifetime)

	if !lifetime.LoggedIn {
		return fmt.Errorf("session expired")
	}

	a.expiresAt = lifetime.ExpireAt.UTC()
	return nil
}

func (a *Auth) login(ctx context.Context) error {
	reqBody := strings.NewReader(fmt.Sprintf(
		"identity=%s&password=%s",
		a.username, a.password,
	))

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodPost,
		a.baseURL+"/ajaxauth/login",
		reqBody)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := a.rawclient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("login failed: %s", resp.Status)
	}

	for _, cookie := range resp.Cookies() {
		a.expiresAt = cookie.Expires.UTC()
		fmt.Printf("%#v\n", cookie)
	}

	if a.expiresAt.IsZero() {
		// assume cookie TTL = 2 hours
		a.expiresAt = time.Now().UTC().Add(2 * time.Hour)
	}

	return nil
}

func (a *Auth) logout(ctx context.Context) error {
	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		a.baseURL+"/ajaxauth/logout",
		nil)
	if err != nil {
		return err
	}

	resp, err := a.rawclient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("logout failed: %s", resp.Status)
	}

	return nil
}

func (a *Auth) ensureAuth(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.expiresAt.IsZero() {
		return a.login(ctx)
	}

	// okay if > 30 minutes
	if time.Now().UTC().Before(a.expiresAt.Add(-30 * time.Minute)) {
		return nil
	}

	// refresh slightly before expiry (buffer)
	if err := a.refresh(ctx); err == nil {
		return nil
	}

	// session expired
	// login
	return a.login(ctx)
}
