package main

import (
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

type AuthTransport struct {
	Base   http.RoundTripper
	Client *Client
}

func (t *AuthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.Client.ensureAuth(req.Context()); err != nil {
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

type rateLimitTransport struct {
	limiter *rate.Limiter
	base    http.RoundTripper
}

func (t *rateLimitTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if err := t.limiter.Wait(req.Context()); err != nil {
		return nil, err
	}

	return t.base.RoundTrip(req)
}

func NewRateLimiterHttpClient() *http.Client {
	return &http.Client{
		Transport: &rateLimitTransport{
			// 10 requests per minute with a burst of 10
			limiter: rate.NewLimiter(rate.Every(time.Minute/2), 1),
			base:    http.DefaultTransport,
		},
	}
}

/*
type Middleware func(http.RoundTripper) http.RoundTripper

func Chain(base http.RoundTripper, m ...Middleware) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}

	for i := len(m) - 1; i >= 0; i-- {
		base = m[i](base)
	}

	return base
}

func NewAuth(auth *Auth) Middleware {
    return func(next http.RoundTripper) http.RoundTripper {
        return &AuthTransport{
            Auth: auth,
            Base: next,
        }
    }
}

func NewRateLimiter(l *rate.Limiter) Middleware {
    return func(next http.RoundTripper) http.RoundTripper {
        return &RateLimitTransport{
            Limiter: l,
            Base: next,
        }
    }
}

client := &http.Client{
    Jar: jar,
    Transport: Chain(
        http.DefaultTransport,
        NewRateLimiter(limiter),
        NewAuth(auth),
        NewLogging(logger),
        NewRetry(),
    ),
}
*/
