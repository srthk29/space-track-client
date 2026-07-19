package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"

	"golang.org/x/time/rate"
)

type AuthTransport struct {
	Base http.RoundTripper
	Auth *Auth
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.Auth.ensureAuth(req.Context()); err != nil {
		return nil, err
	}

	dump, _ := httputil.DumpRequestOut(req, true)
	fmt.Printf("AuthTransport: %s\n", string(dump))

	return t.Base.RoundTrip(req)
}

type RateLimitTransport struct {
	Base    http.RoundTripper
	Limiter *rate.Limiter
}

func (t *RateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.Limiter.Wait(req.Context()); err != nil {
		return nil, err
	}

	dump, _ := httputil.DumpRequestOut(req, true)
	fmt.Printf("RateLimitTransport: %s\n", string(dump))

	return t.Base.RoundTrip(req)
}
