package main

import (
	"net/http"

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

	return t.base().RoundTrip(req)
}

func (t *AuthTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}

	return http.DefaultTransport
}

type RateLimitTransport struct {
	Base    http.RoundTripper
	Limiter *rate.Limiter
}

func (t *RateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.Limiter.Wait(req.Context()); err != nil {
		return nil, err
	}

	return t.base().RoundTrip(req)
}

func (t *RateLimitTransport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}

	return http.DefaultTransport
}
