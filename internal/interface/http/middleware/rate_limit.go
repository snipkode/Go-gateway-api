package middleware

import (
	"context"
	"net/http"
	"strconv"

	"go-enterprise-api/internal/domain/ratelimit"
	"go-enterprise-api/internal/domain/session"

	httpctx "go-enterprise-api/internal/interface/httpapi"
)

// RateLimiter enforces the dynamic rate limit rules against Redis. The
// strictest applicable rule wins, so a per-route login limit is respected even
// when the global limit is high.
//
// Rules are controlled from PostgreSQL (rate_limit_rules) and cached in Redis
// — they can change at runtime without a redeploy.
//
// Rate limiting runs *before* authentication (so login itself is protected),
// therefore user/role scoped limits are resolved by opportunistically parsing
// the bearer token without failing the request when it is absent/expired.
type RateLimiter struct {
	Evaluator ratelimit.Evaluator
	Tokens    session.TokenService
}

func NewRateLimiter(eval ratelimit.Evaluator, tokens session.TokenService) *RateLimiter {
	return &RateLimiter{Evaluator: eval, Tokens: tokens}
}

func (rl *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// auth_request subrequests from Nginx must never be throttled —
		// they are proxied with the client's IP and would produce false 429s.
		if r.URL.Path == "/internal/auth" {
			next.ServeHTTP(w, r)
			return
		}
		subject := rl.subject(r.Context(), r)

		decision, err := rl.Evaluator.Evaluate(r.Context(), subject)
		if err != nil {
			// Fail-open on infrastructure errors keeps the API available;
			// swap to fail-closed for strict compliance environments.
			next.ServeHTTP(w, r)
			return
		}

		if decision != nil && !decision.Rejected {
			w.Header().Set("X-RateLimit-Limit", strconv.FormatInt(decision.Rule.Requests, 10))
			w.Header().Set("X-RateLimit-Remaining", strconv.FormatInt(decision.Remaining, 10))
		}

		if decision != nil && decision.Rejected {
			w.Header().Set("Retry-After", strconv.FormatInt(int64(decision.RetryAfter.Seconds()), 10))
			httpctx.WriteError(w, http.StatusTooManyRequests, "rate_limited", "too many requests")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// subject builds the rate-limit subject. Prefers authenticated claims, but
// falls back to best-effort token parsing so login requests are still
// attributable by IP + route.
func (rl *RateLimiter) subject(ctx context.Context, r *http.Request) ratelimit.Subject {
	subject := ratelimit.Subject{
		IP:    clientIP(r),
		Route: r.Method + ":" + r.URL.Path,
	}
	if c, ok := httpctx.Claims(ctx); ok && c != nil {
		subject.UserID = c.UserID
		subject.Roles = c.Roles
		return subject
	}
	if rl.Tokens != nil {
		token := bearerToken(r)
		if token != "" {
			if c, err := rl.Tokens.Parse(ctx, token); err == nil && c != nil {
				subject.UserID = c.UserID
				subject.Roles = c.Roles
			}
		}
	}
	return subject
}

func bearerToken(r *http.Request) string {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if len(h) > len(prefix) && h[:len(prefix)] == prefix {
		return h[len(prefix):]
	}
	return ""
}

func clientIP(r *http.Request) string {
	// Prefer X-Forwarded-For set by Nginx, then the remote address.
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		for i := 0; i < len(fwd); i++ {
			if fwd[i] == ',' {
				return fwd[:i]
			}
		}
		return fwd
	}
	host := r.RemoteAddr
	for i := 0; i < len(host); i++ {
		if host[i] == ':' {
			return host[:i]
		}
	}
	return host
}
