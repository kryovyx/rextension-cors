// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: © 2026 Kryovyx

// Package cors tests the extension's startup behaviour.
package cors

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	rx "github.com/kryovyx/rextension"
	rxevent "github.com/kryovyx/rextension/event"
)

// ---------------------------------------------------------------------------
// Test doubles
// ---------------------------------------------------------------------------

type mockLogger struct {
	warns  []string
	errors []string
}

func (l *mockLogger) Info(string, ...interface{})  {}
func (l *mockLogger) Debug(string, ...interface{}) {}
func (l *mockLogger) Trace(string, ...interface{}) {}
func (l *mockLogger) Warn(f string, _ ...interface{}) {
	l.warns = append(l.warns, f)
}
func (l *mockLogger) Error(f string, _ ...interface{}) {
	l.errors = append(l.errors, f)
}
func (l *mockLogger) SetLogLevel(rx.LogLevel)                     {}
func (l *mockLogger) WithField(string, interface{}) rx.Logger     { return l }
func (l *mockLogger) WithFields(map[string]interface{}) rx.Logger { return l }
func (l *mockLogger) WithError(error) rx.Logger                   { return l }

type mockBus struct{}

func (mockBus) Subscribe(string, rxevent.EventHandler) {}
func (mockBus) Emit(rxevent.Event)                     {}
func (mockBus) SetLogger(rxevent.BusLogger)            {}
func (mockBus) Close()                                 {}

type mockContainer struct{}

func (mockContainer) Resolve(any) error        { return nil }
func (mockContainer) ResolveAll(any) error     { return nil }
func (mockContainer) Singleton(any) error      { return nil }
func (mockContainer) Scoped(any) error         { return nil }
func (mockContainer) Transient(any) error      { return nil }
func (mockContainer) Instance(any) error       { return nil }
func (mockContainer) Unbind(any) (bool, error) { return false, nil }

// perRouterMW records a UsePerRouter call.
type perRouterMW struct {
	factory  rx.PerRouterMiddleware
	priority int
}

type mockRex struct {
	logger    *mockLogger
	perRouter []perRouterMW
	global    []rx.Middleware
}

func newMockRex() *mockRex { return &mockRex{logger: &mockLogger{}} }

func (m *mockRex) Logger() rx.Logger                      { return m.logger }
func (m *mockRex) Container() rx.Container                { return mockContainer{} }
func (m *mockRex) EventBus() rxevent.EventBus             { return mockBus{} }
func (m *mockRex) Use(mw rx.Middleware)                   { m.global = append(m.global, mw) }
func (m *mockRex) UseOnRouter(string, rx.Middleware, int) {}
func (m *mockRex) UsePerRoute(rx.PerRouteMiddleware, int) {}
func (m *mockRex) UsePerRouter(f rx.PerRouterMiddleware, priority int) {
	m.perRouter = append(m.perRouter, perRouterMW{factory: f, priority: priority})
}
func (m *mockRex) RegisterRoute(rx.Route) error                 { return nil }
func (m *mockRex) RegisterRouteToRouter(rx.Route, string) error { return nil }
func (m *mockRex) CreateRouter(string, rx.RouterConfig) error   { return nil }

// The mock satisfies the contract real extensions are written against.
//
// Note it asserts rx.Container, not dix.Container: this module has no
// dependency on dix at all, which is the point of the contract living in
// rextension (D23).
var (
	_ rx.Rex       = (*mockRex)(nil)
	_ rx.Container = (*mockContainer)(nil)
)

// ---------------------------------------------------------------------------
// Startup
// ---------------------------------------------------------------------------

// TestExtension_rejects_a_contradictory_policy is the O11 guard, at the level
// that matters: the boot fails.
//
// A wildcard allowlist with credentials is not merely unusual — browsers
// reject Access-Control-Allow-Origin: * on a credentialed response, so the
// combination silently blocks every cross-origin request it appears to permit.
// The resulting failure looks like a server bug from the client's side, which
// is exactly the kind of misconfiguration that should stop a deployment.
func TestExtension_rejects_a_contradictory_policy(t *testing.T) {
	ext := NewCORSExtension(NewConfig(
		WithAllowedOrigins("*"),
		WithAllowCredentials(true),
	))
	r := newMockRex()

	err := ext.OnInitialize(context.Background(), r)
	if err == nil {
		t.Fatal("a wildcard policy with credentials was accepted; browsers would discard every response")
	}

	var pe *rx.OriginPolicyError
	if !errors.As(err, &pe) {
		t.Fatalf("expected an OriginPolicyError, got %T: %v", err, err)
	}
	// The message has to explain the interaction, or the owner just sees a
	// rejection of something that looks reasonable.
	if !strings.Contains(err.Error(), "AllowCredentials") {
		t.Errorf("the error should explain the interaction: %v", err)
	}
}

// TestExtension_rejects_a_malformed_origin covers the three mistakes that all
// fail the same silent way: the policy looks configured and allows nothing.
func TestExtension_rejects_a_malformed_origin(t *testing.T) {
	cases := []struct {
		name   string
		origin string
		reason string
	}{
		{"no scheme", "app.example.com", "never matches an Origin header, which always carries a scheme"},
		{"trailing path", "https://app.example.com/", "an Origin header never carries a path"},
		{"wildcard pattern", "https://*.example.com", "patterns are not supported and would match nothing"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ext := NewCORSExtension(NewConfig(WithAllowedOrigins(tc.origin)))
			if err := ext.OnInitialize(context.Background(), newMockRex()); err == nil {
				t.Fatalf("%q was accepted, but %s", tc.origin, tc.reason)
			}
		})
	}
}

// TestExtension_warns_when_nothing_is_allowed. Not an error — the origins may
// be configured elsewhere — but worth saying out loud rather than leaving to
// be discovered from a browser console.
func TestExtension_warns_when_nothing_is_allowed(t *testing.T) {
	ext := NewCORSExtension(nil)
	r := newMockRex()

	if err := ext.OnInitialize(context.Background(), r); err != nil {
		t.Fatalf("an empty policy should not fail startup: %v", err)
	}
	if len(r.logger.warns) == 0 {
		t.Fatal("no warning for a CORS extension that allows nothing")
	}
}

// TestExtension_attaches_at_PriorityCORS is the ordering that makes error
// responses readable to a cross-origin client.
//
// Outside authentication and rate limiting, because a browser will not let
// script read a response whose CORS headers are missing — including a 401 or a
// 429.
func TestExtension_attaches_at_PriorityCORS(t *testing.T) {
	ext := NewCORSExtension(NewConfig(WithAllowedOrigins("https://app.example.com")))
	r := newMockRex()

	if err := ext.OnInitialize(context.Background(), r); err != nil {
		t.Fatalf("OnInitialize failed: %v", err)
	}

	if len(r.perRouter) != 1 {
		t.Fatalf("expected 1 per-router middleware, got %d", len(r.perRouter))
	}
	if got := r.perRouter[0].priority; got != rx.PriorityCORS {
		t.Fatalf("priority = %d, want PriorityCORS (%d) — the headers must be present on error responses too",
			got, rx.PriorityCORS)
	}
	if got := r.perRouter[0].priority; got >= rx.PriorityAuth {
		t.Error("CORS must compose outside authentication, or a 401 reaches the browser with no CORS headers " +
			"and the client sees an opaque failure instead of the status")
	}

	// The factory must produce a working middleware.
	mw := r.perRouter[0].factory("default")
	if mw == nil {
		t.Fatal("the factory returned no middleware")
	}
	if _, ok := interface{}(mw(okHandler)).(http.Handler); !ok {
		t.Fatal("the middleware did not produce an http.Handler")
	}
}

// TestExtension_does_not_register_routes is the O11 decision.
//
// The router already answers OPTIONS from its Allow set (D35/P2.7). A second
// handler competing for the same path would either conflict at registration
// or shadow the router's answer — which is the accurate one, because it is
// derived from the routes that exist.
func TestExtension_does_not_register_routes(t *testing.T) {
	ext := NewCORSExtension(NewConfig(WithAllowedOrigins("https://app.example.com")))
	r := newMockRex()

	if err := ext.OnInitialize(context.Background(), r); err != nil {
		t.Fatalf("OnInitialize failed: %v", err)
	}
	if len(r.global) != 0 {
		t.Error("a global middleware was registered; CORS attaches per router")
	}
}

// TestExtension_exposes_its_policy_for_sharing_with_CSRF covers O11's shared
// allowlist: one list, two consumers, neither importing the other.
func TestExtension_exposes_its_policy_for_sharing_with_CSRF(t *testing.T) {
	ext := NewCORSExtension(NewConfig(
		WithAllowedOrigins("https://app.example.com"),
		WithAllowCredentials(true),
	))

	pe, ok := ext.(*Extension)
	if !ok {
		t.Fatalf("unexpected type %T", ext)
	}
	policy := pe.Policy()
	if !policy.Allows("https://app.example.com") {
		t.Fatal("the exposed policy does not allow the configured origin")
	}
	if !policy.AllowCredentials {
		t.Error("the exposed policy lost AllowCredentials")
	}
}

// ---------------------------------------------------------------------------
// Construction, config options and the remaining lifecycle
// ---------------------------------------------------------------------------

func TestWithCORS_returns_an_option(t *testing.T) {
	if WithCORS(NewConfig()) == nil {
		t.Fatal("WithCORS returned a nil Option")
	}
}

func TestExtension_lifecycle_noops(t *testing.T) {
	ext := NewCORSExtension(NewConfig(WithAllowedOrigins("https://app.example.com")))
	r := newMockRex()
	ctx := context.Background()

	if err := ext.OnInitialize(ctx, r); err != nil {
		t.Fatalf("OnInitialize: %v", err)
	}
	for name, fn := range map[string]func(context.Context, rx.Rex) error{
		"OnStart":    ext.OnStart,
		"OnReady":    ext.OnReady,
		"OnStop":     ext.OnStop,
		"OnShutdown": ext.OnShutdown,
	} {
		if err := fn(ctx, r); err != nil {
			t.Errorf("%s: %v", name, err)
		}
	}
}

// WithPolicy sets the whole policy at once, which is how one value is shared
// with the CSRF configuration (O11). Writing the origin list twice is how the
// two drift apart.
func TestWithPolicy_sets_the_whole_policy(t *testing.T) {
	shared := rx.OriginPolicy{
		AllowedOrigins:   []string{"https://app.example.com", "https://admin.example.com"},
		AllowCredentials: true,
	}
	cfg := NewConfig(WithPolicy(shared))

	if len(cfg.Policy.AllowedOrigins) != 2 {
		t.Fatalf("AllowedOrigins = %v", cfg.Policy.AllowedOrigins)
	}
	if !cfg.Policy.AllowCredentials {
		t.Error("AllowCredentials was not carried over")
	}
	// And the extension exposes it back, so CSRF can be handed the same value.
	ext := NewCORSExtension(cfg)
	p, ok := ext.(interface{ Policy() rx.OriginPolicy })
	if !ok {
		t.Fatal("the extension does not expose Policy()")
	}
	if len(p.Policy().AllowedOrigins) != 2 {
		t.Fatalf("Policy() = %v", p.Policy())
	}
}

// WithPolicy replaces rather than merges, so a later call wins outright.
func TestWithPolicy_replaces_earlier_origin_options(t *testing.T) {
	cfg := NewConfig(
		WithAllowedOrigins("https://first.example"),
		WithPolicy(rx.OriginPolicy{AllowedOrigins: []string{"https://second.example"}}),
	)
	if len(cfg.Policy.AllowedOrigins) != 1 || cfg.Policy.AllowedOrigins[0] != "https://second.example" {
		t.Fatalf("AllowedOrigins = %v, want only the value from WithPolicy", cfg.Policy.AllowedOrigins)
	}
}

// WithExposedHeaders replaces the defaults rather than adding to them, so the
// X-RateLimit-* headers have to be listed again if they are still wanted.
func TestWithExposedHeaders_replaces_the_defaults(t *testing.T) {
	defaults := NewDefaultConfig().ExposedHeaders
	if len(defaults) == 0 {
		t.Fatal("expected some default exposed headers to replace")
	}

	cfg := NewConfig(WithExposedHeaders("X-Total-Count"))
	if len(cfg.ExposedHeaders) != 1 || cfg.ExposedHeaders[0] != "X-Total-Count" {
		t.Fatalf("ExposedHeaders = %v, want exactly the configured list", cfg.ExposedHeaders)
	}
}

func TestWithExposedHeaders_with_no_arguments_clears_them(t *testing.T) {
	cfg := NewConfig(WithExposedHeaders())
	if len(cfg.ExposedHeaders) != 0 {
		t.Fatalf("ExposedHeaders = %v, want none", cfg.ExposedHeaders)
	}
}
