// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: © 2026 Kryovyx

// Package cors provides a Rex extension implementing Cross-Origin Resource
// Sharing (D33/O11).
//
// # CORS is not CSRF protection
//
// Worth stating at the top of the package, because the belief that it is
// causes real vulnerabilities.
//
// A "simple" cross-origin request — a form POST with
// application/x-www-form-urlencoded, multipart/form-data or text/plain — gets
// **no preflight**. The browser sends it, the server receives it, and the
// server commits the state change. CORS then stops the attacker from *reading*
// the response, which does nothing about the transfer that already happened.
//
// CORS answers "may this origin read my responses?". Use rextension-security's
// CSRF protection to answer "did this request come from my own application?".
// They share an origin allowlist (rextension.OriginPolicy) and nothing else.
package cors

import (
	"context"
	"fmt"

	rx "github.com/kryovyx/rextension"
)

// Extension implements the Rex extension contract for CORS.
type Extension struct {
	cfg    Config
	logger rx.Logger
}

// NewCORSExtension constructs a CORS extension.
//
// A nil cfg takes the defaults, which allow **no** origin. That is the only
// safe default: an extension that permitted any origin out of the box would
// turn adding it into a policy decision its author did not make.
func NewCORSExtension(cfg *Config) rx.Extension {
	c := NewDefaultConfig()
	if cfg != nil {
		c = cfg
	}
	return &Extension{cfg: *c}
}

// WithCORS is a helper Option to attach the extension to Rex.
func WithCORS(cfg *Config) rx.Option {
	return rx.WithExtension(NewCORSExtension(cfg))
}

// OnInitialize validates the policy and attaches the middleware.
func (e *Extension) OnInitialize(_ context.Context, r rx.Rex) error {
	e.logger = r.Logger()

	// A contradictory policy stops the boot rather than producing responses
	// browsers silently discard — the failure mode that reads as a server bug
	// from the client's side.
	if err := e.cfg.Policy.Valid(); err != nil {
		e.logger.Error("Invalid CORS policy: %v", err)
		return fmt.Errorf("cors: %w", err)
	}

	if len(e.cfg.Policy.AllowedOrigins) == 0 {
		// Not an error: an application may attach the extension and configure
		// the origins elsewhere. But it allows nothing, which is worth saying
		// out loud rather than leaving to be discovered from the browser
		// console.
		e.logger.Warn("CORS is enabled with no allowed origins; every cross-origin request will be refused")
	}

	// Attached per router, at PriorityCORS — outside authentication and rate
	// limiting.
	//
	// The headers have to be present on error responses too: a browser will
	// not let script read a response whose CORS headers are missing,
	// including a 401 or a 429. Without them a cross-origin client sees an
	// opaque network failure instead of the status the server sent.
	r.UsePerRouter(func(routerName string) rx.Middleware {
		return Middleware(&e.cfg)
	}, rx.PriorityCORS)

	e.logger.Info("CORS extension initialized with %d allowed origin(s), credentials=%v",
		len(e.cfg.Policy.AllowedOrigins), e.cfg.Policy.AllowCredentials)
	return nil
}

// OnStart is a no-op.
func (e *Extension) OnStart(context.Context, rx.Rex) error { return nil }

// OnReady is a no-op.
func (e *Extension) OnReady(context.Context, rx.Rex) error { return nil }

// OnStop is a no-op.
func (e *Extension) OnStop(context.Context, rx.Rex) error { return nil }

// OnShutdown is a no-op.
func (e *Extension) OnShutdown(context.Context, rx.Rex) error { return nil }

// Policy returns the configured origin policy, for sharing with the CSRF
// configuration.
func (e *Extension) Policy() rx.OriginPolicy { return e.cfg.Policy }
