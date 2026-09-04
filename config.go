// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: © 2026 Kryovyx

// Package cors provides a Rex extension implementing Cross-Origin Resource
// Sharing.
//
// This file defines the Config struct and its functional options.
package cors

import (
	"net/http"
	"time"

	rx "github.com/kryovyx/rextension"
)

// Config controls the CORS policy.
type Config struct {
	// Policy is the shared origin allowlist, also used by CSRF (D33/O11).
	//
	// One list, two consumers, because an application trusts one set of
	// origins. Nothing else is shared: CORS answers "may this origin read my
	// responses?" and CSRF answers "did this request come from my own
	// application?".
	Policy rx.OriginPolicy

	// AllowedMethods are the methods advertised in
	// Access-Control-Allow-Methods on a preflight response.
	//
	// Empty means the router's own Allow set for the path, which is the
	// better answer: it is derived from the routes that actually exist, so it
	// cannot drift out of step with them.
	AllowedMethods []string

	// AllowedHeaders are the request headers a client may send, advertised in
	// Access-Control-Allow-Headers.
	//
	// Empty echoes the requested headers back, which is permissive but
	// honest: a browser only asks for headers the page is actually trying to
	// send, and an allowlist that omits one produces a failure the developer
	// sees as "CORS is broken" rather than as a policy decision.
	AllowedHeaders []string

	// ExposedHeaders are response headers the browser makes readable to
	// script, via Access-Control-Expose-Headers.
	//
	// Without this a cross-origin script can read only the CORS-safelisted
	// response headers, so a client cannot see X-RateLimit-Remaining or a
	// pagination header even though they are sent.
	ExposedHeaders []string

	// MaxAge is how long a browser may cache a preflight result.
	//
	// Defaults to 10 minutes. Browsers cap this — Chrome at 2 hours, Firefox
	// at 24 — so a larger value is silently clamped rather than honoured.
	MaxAge time.Duration
}

// NewDefaultConfig returns a configuration that allows nothing.
//
// The zero policy allows no origin, which is the only safe default: a CORS
// extension that permits any origin out of the box turns adding the extension
// into a policy decision the author did not make.
func NewDefaultConfig() *Config {
	return &Config{
		MaxAge: 10 * time.Minute,
		ExposedHeaders: []string{
			// Sent by rextension-ratelimit on every response. Useless to a
			// cross-origin client unless exposed, and a client that cannot
			// read its remaining quota discovers the limit by exceeding it.
			"X-RateLimit-Limit",
			"X-RateLimit-Remaining",
			"X-RateLimit-Reset",
			"Retry-After",
		},
	}
}

// ConfigOption configures the extension.
type ConfigOption func(*Config)

// WithAllowedOrigins sets the exact origins permitted.
//
// Each entry must be a serialised origin — scheme, host and optional port:
//
//	cors.WithAllowedOrigins("https://app.example.com", "http://localhost:3000")
//
// Patterns are not supported, and a bare hostname is rejected at startup. Both
// mistakes otherwise produce a policy that looks configured and allows
// nothing.
func WithAllowedOrigins(origins ...string) ConfigOption {
	return func(c *Config) { c.Policy.AllowedOrigins = origins }
}

// WithAllowCredentials permits cookies and HTTP authentication on cross-origin
// requests.
//
// ⚠ Incompatible with a "*" allowlist; the combination is refused at startup
// rather than emitted, because browsers discard it and the resulting failure
// looks like a server bug.
func WithAllowCredentials(allow bool) ConfigOption {
	return func(c *Config) { c.Policy.AllowCredentials = allow }
}

// WithPolicy sets the whole origin policy at once, for sharing one value with
// the CSRF configuration.
func WithPolicy(p rx.OriginPolicy) ConfigOption {
	return func(c *Config) { c.Policy = p }
}

// WithAllowedMethods overrides the advertised methods.
//
// Prefer leaving it empty: the router's Allow set for the path is derived from
// the routes that exist, so it cannot drift.
func WithAllowedMethods(methods ...string) ConfigOption {
	return func(c *Config) { c.AllowedMethods = methods }
}

// WithAllowedHeaders sets the request headers a client may send.
func WithAllowedHeaders(headers ...string) ConfigOption {
	return func(c *Config) { c.AllowedHeaders = headers }
}

// WithExposedHeaders sets the response headers a cross-origin script may read.
//
// Replaces the defaults rather than adding to them; include the X-RateLimit-*
// headers explicitly if they are still wanted.
func WithExposedHeaders(headers ...string) ConfigOption {
	return func(c *Config) { c.ExposedHeaders = headers }
}

// WithMaxAge sets how long a browser may cache a preflight result.
func WithMaxAge(d time.Duration) ConfigOption {
	return func(c *Config) { c.MaxAge = d }
}

// NewConfig creates a config with the given options.
func NewConfig(opts ...ConfigOption) *Config {
	c := NewDefaultConfig()
	for _, opt := range opts {
		if opt != nil {
			opt(c)
		}
	}
	return c
}

// defaultPreflightMethods is used when the router advertises no Allow set.
var defaultPreflightMethods = []string{
	http.MethodGet, http.MethodHead, http.MethodPost,
	http.MethodPut, http.MethodPatch, http.MethodDelete,
}
