// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: © 2026 Kryovyx

// Package cors provides a Rex extension implementing Cross-Origin Resource
// Sharing.
//
// This file implements the middleware.
package cors

import (
	"net/http"
	"strconv"
	"strings"

	rx "github.com/kryovyx/rextension"
)

// Middleware returns the CORS middleware for one router.
//
// # What it does, and does not do
//
// It **decorates** responses. It does not register OPTIONS routes of its own:
// the router already answers OPTIONS from its Allow set (D35/P2.7), and a
// second handler competing for the same path would either conflict at
// registration or shadow the router's answer — which is the accurate one,
// because it is derived from the routes that exist.
//
// So a preflight arrives, the router answers 204 with an Allow header, and
// this middleware adds the Access-Control-* headers on the way out.
//
// # Why the headers go on error responses too
//
// A browser will not let script read a response whose CORS headers are
// missing — including a 401, a 429 or a 500. Without the headers, a
// cross-origin client sees an opaque network failure instead of the status the
// server actually sent, and the developer sees "CORS error" instead of
// "unauthorized". That is why this runs at PriorityCORS, outside
// authentication and rate limiting: the headers must be present whatever the
// inner layers decide.
func Middleware(cfg *Config) rx.Middleware {
	// Precomputed once, at composition time.
	exposed := strings.Join(cfg.ExposedHeaders, ", ")
	allowedMethods := strings.Join(cfg.AllowedMethods, ", ")
	allowedHeaders := strings.Join(cfg.AllowedHeaders, ", ")
	maxAge := strconv.Itoa(int(cfg.MaxAge.Seconds()))

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := rx.RequestOrigin(r)

			// No Origin header: not a cross-origin browser request, so there
			// is no CORS decision to make and nothing to add.
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			h := w.Header()

			// Vary: Origin on every response with an Origin header, allowed or
			// not.
			//
			// Without it a shared cache can serve a response containing
			// `Access-Control-Allow-Origin: https://a.example` to a request
			// from https://b.example — which either leaks the response to an
			// origin that should not read it, or blocks an origin that
			// should. It has to be set even on the refusal path, because the
			// refusal is also origin-dependent.
			h.Add("Vary", "Origin")

			if !cfg.Policy.Allows(origin) {
				// Deliberately not an error response.
				//
				// The request proceeds and the browser enforces the refusal by
				// withholding the response from script. Returning 403 here
				// would be worse in two ways: a non-browser client, which is
				// not subject to CORS at all, would be refused for sending a
				// header it is free to send; and the browser would report a
				// server error rather than a policy decision.
				next.ServeHTTP(w, r)
				return
			}

			// Echo the concrete origin rather than "*".
			//
			// Required when credentials are allowed — browsers reject "*" on a
			// credentialed response — and better even without them, because it
			// keeps the response accurate for the origin it was produced for.
			h.Set("Access-Control-Allow-Origin", origin)
			if cfg.Policy.AllowCredentials {
				h.Set("Access-Control-Allow-Credentials", "true")
			}
			if exposed != "" {
				h.Set("Access-Control-Expose-Headers", exposed)
			}

			// A preflight is an OPTIONS request carrying
			// Access-Control-Request-Method. An OPTIONS request without it is
			// an ordinary one, and is not a preflight.
			requested := r.Header.Get("Access-Control-Request-Method")
			if r.Method == http.MethodOptions && requested != "" {
				h.Add("Vary", "Access-Control-Request-Method")
				h.Add("Vary", "Access-Control-Request-Headers")

				if allowedMethods != "" {
					h.Set("Access-Control-Allow-Methods", allowedMethods)
				}

				if allowedHeaders != "" {
					h.Set("Access-Control-Allow-Headers", allowedHeaders)
				} else if requestedHeaders := r.Header.Get("Access-Control-Request-Headers"); requestedHeaders != "" {
					// Echo what was asked for. A browser only asks for headers
					// the page is actually trying to send, and an allowlist
					// that omits one produces a failure the developer reads as
					// "CORS is broken" rather than as a policy decision.
					h.Set("Access-Control-Allow-Headers", requestedHeaders)
				}

				if cfg.MaxAge > 0 {
					h.Set("Access-Control-Max-Age", maxAge)
				}

				// Hand off to the router, which answers the OPTIONS from its
				// Allow set. If AllowedMethods was not configured, that Allow
				// header is the authoritative method list — copied into
				// Access-Control-Allow-Methods below, after the router has
				// written it.
				wrapped := &preflightWriter{
					ResponseWriter:  w,
					copyAllow:       allowedMethods == "",
					fallbackMethods: defaultPreflightMethods,
				}
				next.ServeHTTP(wrapped, r)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// preflightWriter copies the router's Allow header into
// Access-Control-Allow-Methods when the response is committed.
//
// It has to happen at WriteHeader time: the router sets Allow while handling
// the request, which is after this middleware has run its pre-handler code.
// Reading it earlier would read nothing.
//
// Using the router's own Allow set rather than a configured list means the
// advertised methods are derived from the routes that actually exist, so they
// cannot drift out of step with them — and a route added later is advertised
// without anyone remembering to update a list.
type preflightWriter struct {
	http.ResponseWriter

	copyAllow       bool
	fallbackMethods []string
	wroteHeader     bool
}

func (p *preflightWriter) WriteHeader(status int) {
	if !p.wroteHeader {
		p.wroteHeader = true
		if p.copyAllow {
			h := p.Header()
			if allow := h.Get("Allow"); allow != "" {
				h.Set("Access-Control-Allow-Methods", allow)
			} else if h.Get("Access-Control-Allow-Methods") == "" {
				// The router answered without an Allow header — a 404, most
				// likely. Advertise the common methods rather than nothing:
				// an empty Access-Control-Allow-Methods fails the preflight
				// for every method, including ones the application does serve
				// elsewhere.
				h.Set("Access-Control-Allow-Methods", strings.Join(p.fallbackMethods, ", "))
			}
		}
	}
	p.ResponseWriter.WriteHeader(status)
}

func (p *preflightWriter) Write(b []byte) (int, error) {
	if !p.wroteHeader {
		p.WriteHeader(http.StatusOK)
	}
	return p.ResponseWriter.Write(b)
}

// Unwrap returns the wrapped writer.
func (p *preflightWriter) Unwrap() http.ResponseWriter { return p.ResponseWriter }

// Flush forwards to the underlying Flusher.
//
// A preflight response is never streamed, but the wrapper still has to forward
// the optional interfaces: dropping one is a behaviour regression that only
// appears when something downstream needs it (D45).
func (p *preflightWriter) Flush() {
	if f, ok := p.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

var (
	_ http.ResponseWriter = (*preflightWriter)(nil)
	_ http.Flusher        = (*preflightWriter)(nil)
)
