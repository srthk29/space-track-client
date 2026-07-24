package spacetrackclient

import (
	"log/slog"
	"net/http"

	"golang.org/x/time/rate"
)

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
			Base:    next,
		}
	}
}

func NewLog(l *slog.Logger) Middleware {
	return func(next http.RoundTripper) http.RoundTripper {
		return &LoggingTransport{
			Logger: l,
			Base:   next,
		}
	}
}
