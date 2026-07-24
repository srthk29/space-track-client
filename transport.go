package main

import (
	"log/slog"
	"net/http"
	"net/http/httputil"
	"time"

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

	return t.Base.RoundTrip(req)
}

type LoggingTransport struct {
	Base   http.RoundTripper
	Logger *slog.Logger
}

func (t *LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	start := time.Now()

	t.Logger.Debug(
		"sending HTTP request",
		"method", req.Method,
		"url", req.URL.String(),
	)

	dump, err := httputil.DumpRequestOut(req, true)
	if err == nil {
		t.Logger.Debug(
			"HTTP request",
			"request", string(dump),
		)
	}

	resp, err := t.Base.RoundTrip(req)
	elapsed := time.Since(start)

	if err != nil {
		t.Logger.Debug(
			"HTTP request failed",
			"method", req.Method,
			"url", req.URL.String(),
			"elapsed", elapsed,
			"error", err,
		)
		return nil, err
	}

	t.Logger.Debug(
		"received HTTP response",
		"method", req.Method,
		"url", req.URL.String(),
		"status", resp.Status,
		"elapsed", elapsed,
	)

	return resp, nil
}
