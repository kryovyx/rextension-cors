// SPDX-License-Identifier: MIT
// SPDX-FileCopyrightText: © 2026 Kryovyx

// Package cors tests the CORS middleware (D33/O11).
package cors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	rx "github.com/kryovyx/rextension"
)

// okHandler answers 200.
var okHandler = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
})

// allowHandler answers like the router's automatic OPTIONS: 204 with an Allow
// header (D35/P2.7).
func allowHandler(allow string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Allow", allow)
		w.WriteHeader(http.StatusNoContent)
	})
}

// appConfig is a typical single-origin credentialed policy.
func appConfig() *Config {
	return NewConfig(
		WithAllowedOrigins("https://app.example.com"),
		WithAllowCredentials(true),
	)
}

// ---------------------------------------------------------------------------
// Simple requests
// ---------------------------------------------------------------------------

func TestMiddleware_allows_a_listed_origin(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/things", nil)
	req.Header.Set("Origin", "https://app.example.com")

	Middleware(appConfig())(okHandler).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin = %q", got)
	}
	if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
		t.Errorf("Access-Control-Allow-Credentials = %q, want true", got)
	}
	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
}

// TestMiddleware_echoes_the_concrete_origin. Echoing rather than sending "*"
// is required when credentials are allowed, and keeps the response accurate
// for the origin it was produced for even when they are not.
func TestMiddleware_echoes_the_concrete_origin(t *testing.T) {
	cfg := NewConfig(WithAllowedOrigins("https://a.example", "https://b.example"))

	for _, origin := range []string{"https://a.example", "https://b.example"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", origin)

		Middleware(cfg)(okHandler).ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("origin %q → Access-Control-Allow-Origin %q", origin, got)
		}
	}
}

// TestMiddleware_refuses_an_unlisted_origin_without_erroring is a decision
// worth stating.
//
// The request proceeds and the browser enforces the refusal by withholding the
// response from script. Returning 403 would refuse a non-browser client — which
// is not subject to CORS at all — for sending a header it is free to send, and
// would make the browser report a server error rather than a policy decision.
func TestMiddleware_refuses_an_unlisted_origin_without_erroring(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/things", nil)
	req.Header.Set("Origin", "https://evil.example")

	served := false
	handler := Middleware(appConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		served = true
		w.WriteHeader(http.StatusOK)
	}))
	handler.ServeHTTP(rec, req)

	if !served {
		t.Fatal("the request was refused server-side; the browser enforces CORS, not the server")
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("an unlisted origin received Access-Control-Allow-Origin: %q", got)
	}
	// Vary must be set even on the refusal, because the refusal is itself
	// origin-dependent.
	if !strings.Contains(rec.Header().Get("Vary"), "Origin") {
		t.Error("Vary: Origin is missing from a refused response, so a shared cache could reuse it for another origin")
	}
}

// TestMiddleware_ignores_a_request_with_no_Origin. A same-origin request
// carries no Origin header for most methods and needs no CORS decision.
func TestMiddleware_ignores_a_request_with_no_Origin(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/things", nil)

	Middleware(appConfig())(okHandler).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("a request with no Origin received CORS headers: %q", got)
	}
	if got := rec.Header().Get("Vary"); got != "" {
		t.Errorf("Vary = %q; there is no origin-dependent decision to vary on", got)
	}
}

// TestMiddleware_always_sets_Vary_Origin.
//
// Without it a shared cache can serve a response carrying
// `Access-Control-Allow-Origin: https://a.example` to a request from
// https://b.example — leaking the response to an origin that should not read
// it, or blocking one that should.
func TestMiddleware_always_sets_Vary_Origin(t *testing.T) {
	for _, origin := range []string{"https://app.example.com", "https://evil.example"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", origin)

		Middleware(appConfig())(okHandler).ServeHTTP(rec, req)

		if !strings.Contains(rec.Header().Get("Vary"), "Origin") {
			t.Errorf("origin %q: Vary: Origin is missing", origin)
		}
	}
}

// TestMiddleware_exposes_headers. Without Access-Control-Expose-Headers a
// cross-origin script can read only the CORS-safelisted response headers, so
// a client cannot see its remaining rate-limit quota even though it is sent.
func TestMiddleware_exposes_headers(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Origin", "https://app.example.com")

	Middleware(appConfig())(okHandler).ServeHTTP(rec, req)

	exposed := rec.Header().Get("Access-Control-Expose-Headers")
	for _, want := range []string{"X-RateLimit-Remaining", "Retry-After"} {
		if !strings.Contains(exposed, want) {
			t.Errorf("Access-Control-Expose-Headers %q should include %s", exposed, want)
		}
	}
}

// TestMiddleware_decorates_error_responses is why this runs at PriorityCORS.
//
// A browser will not let script read a response whose CORS headers are
// missing, including a 401 or a 429. Without them a cross-origin client sees
// an opaque network failure instead of the status the server sent — and the
// developer sees "CORS error" instead of "unauthorized".
func TestMiddleware_decorates_error_responses(t *testing.T) {
	for _, status := range []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusTooManyRequests,
		http.StatusInternalServerError,
	} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", "https://app.example.com")

		handler := Middleware(appConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(status)
		}))
		handler.ServeHTTP(rec, req)

		if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "" {
			t.Errorf("status %d: no CORS headers, so the browser hides the status from script", status)
		}
	}
}

// ---------------------------------------------------------------------------
// Preflight
// ---------------------------------------------------------------------------

// TestPreflight_uses_the_routers_Allow_set is the O11 decision: this extension
// decorates the router's OPTIONS response rather than registering its own.
//
// Deriving the advertised methods from the router's Allow header means they
// come from the routes that actually exist, so they cannot drift out of step
// with them — and a route added later is advertised without anyone updating a
// list.
func TestPreflight_uses_the_routers_Allow_set(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/things", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	Middleware(appConfig())(allowHandler("OPTIONS, POST, PUT")).ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204 — the router answers the OPTIONS", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "OPTIONS, POST, PUT" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want the router's Allow set", got)
	}
	if got := rec.Header().Get("Allow"); got == "" {
		t.Error("the router's own Allow header was dropped")
	}
}

// TestPreflight_configured_methods_win covers the override.
func TestPreflight_configured_methods_win(t *testing.T) {
	cfg := NewConfig(
		WithAllowedOrigins("https://app.example.com"),
		WithAllowedMethods("GET", "POST"),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/things", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	Middleware(cfg)(allowHandler("OPTIONS, PUT, DELETE")).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want the configured list", got)
	}
}

// TestPreflight_falls_back_when_the_router_sends_no_Allow.
//
// An empty Access-Control-Allow-Methods fails the preflight for every method,
// including ones the application does serve elsewhere — so a 404 on the
// preflight path must not produce one.
func TestPreflight_falls_back_when_the_router_sends_no_Allow(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/nowhere", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	handler := Middleware(appConfig())(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound) // no Allow header
	}))
	handler.ServeHTTP(rec, req)

	got := rec.Header().Get("Access-Control-Allow-Methods")
	if got == "" {
		t.Fatal("Access-Control-Allow-Methods is empty, which fails the preflight for every method")
	}
	if !strings.Contains(got, "POST") {
		t.Errorf("Access-Control-Allow-Methods = %q, want the common methods", got)
	}
}

// TestPreflight_echoes_requested_headers. A browser only asks for headers the
// page is trying to send, and an allowlist that omits one produces a failure
// the developer reads as "CORS is broken".
func TestPreflight_echoes_requested_headers(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/things", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type, x-csrf-token")

	Middleware(appConfig())(allowHandler("OPTIONS, POST")).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "content-type, x-csrf-token" {
		t.Fatalf("Access-Control-Allow-Headers = %q", got)
	}
}

func TestPreflight_configured_headers_win(t *testing.T) {
	cfg := NewConfig(
		WithAllowedOrigins("https://app.example.com"),
		WithAllowedHeaders("Content-Type", "Authorization"),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/things", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "x-anything")

	Middleware(cfg)(allowHandler("OPTIONS, POST")).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Headers"); got != "Content-Type, Authorization" {
		t.Fatalf("Access-Control-Allow-Headers = %q, want the configured list", got)
	}
}

func TestPreflight_sets_MaxAge(t *testing.T) {
	cfg := NewConfig(
		WithAllowedOrigins("https://app.example.com"),
		WithMaxAge(600*time.Second),
	)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/things", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	Middleware(cfg)(allowHandler("OPTIONS, POST")).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Max-Age"); got != "600" {
		t.Fatalf("Access-Control-Max-Age = %q, want 600", got)
	}
}

// TestPreflight_varies_on_the_request_headers. The response depends on
// Access-Control-Request-Method and -Headers, so a cache must not reuse it for
// a preflight asking about something else.
func TestPreflight_varies_on_the_request_headers(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/things", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	Middleware(appConfig())(allowHandler("OPTIONS, POST")).ServeHTTP(rec, req)

	vary := strings.Join(rec.Header().Values("Vary"), ", ")
	for _, want := range []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"} {
		if !strings.Contains(vary, want) {
			t.Errorf("Vary %q should include %s", vary, want)
		}
	}
}

// TestOPTIONS_without_a_request_method_is_not_a_preflight. An ordinary OPTIONS
// request is not a preflight and must not be treated as one.
func TestOPTIONS_without_a_request_method_is_not_a_preflight(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/things", nil)
	req.Header.Set("Origin", "https://app.example.com")
	// No Access-Control-Request-Method.

	Middleware(appConfig())(allowHandler("OPTIONS, POST")).ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Max-Age"); got != "" {
		t.Errorf("a non-preflight OPTIONS received Access-Control-Max-Age: %q", got)
	}
	// The origin headers are still correct — it is a cross-origin request.
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got == "" {
		t.Error("a cross-origin OPTIONS should still carry the origin headers")
	}
}

// ---------------------------------------------------------------------------
// Wildcards
// ---------------------------------------------------------------------------

// TestWildcard_without_credentials_allows_any_origin.
func TestWildcard_without_credentials_allows_any_origin(t *testing.T) {
	cfg := NewConfig(WithAllowedOrigins("*"))

	for _, origin := range []string{"https://a.example", "http://localhost:3000"} {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, "/x", nil)
		req.Header.Set("Origin", origin)

		Middleware(cfg)(okHandler).ServeHTTP(rec, req)

		// The concrete origin is echoed, not "*". Accurate for the origin the
		// response was produced for, and required if credentials are ever
		// enabled later.
		if got := rec.Header().Get("Access-Control-Allow-Origin"); got != origin {
			t.Errorf("origin %q → %q", origin, got)
		}
	}
}

// TestWildcard_with_credentials_allows_nothing is the O11 guard.
//
// Browsers reject Access-Control-Allow-Origin: * on a credentialed response,
// so the combination silently blocks every request it appears to permit. The
// policy refuses rather than emitting it, and the extension refuses to start
// at all — see TestExtension_rejects_a_contradictory_policy.
func TestWildcard_with_credentials_allows_nothing(t *testing.T) {
	policy := rx.OriginPolicy{AllowedOrigins: []string{"*"}, AllowCredentials: true}

	if policy.Allows("https://a.example") {
		t.Fatal("a wildcard policy with credentials allowed an origin; browsers would discard the response")
	}
	if err := policy.Valid(); err == nil {
		t.Fatal("a wildcard policy with credentials was accepted as valid")
	}
}

// ---------------------------------------------------------------------------
// preflightWriter — the wrapper's own contract
// ---------------------------------------------------------------------------

// flushRecorder is an httptest.ResponseRecorder that counts flushes, so the
// forwarding can be observed rather than merely assumed.
type flushRecorder struct {
	*httptest.ResponseRecorder
	flushes int
}

func (f *flushRecorder) Flush() { f.flushes++; f.ResponseRecorder.Flush() }

// nonFlusher implements only http.ResponseWriter, so Flush has nothing to
// forward to and must not panic.
type nonFlusher struct{ hdr http.Header }

func (w *nonFlusher) Header() http.Header         { return w.hdr }
func (w *nonFlusher) WriteHeader(int)             {}
func (w *nonFlusher) Write(b []byte) (int, error) { return len(b), nil }

// Write must commit the header first, exactly as net/http does — otherwise a
// handler that writes a body without calling WriteHeader produces a preflight
// response with no Access-Control-Allow-Methods.
func TestPreflightWriter_Write_commits_the_header(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Allow", "GET, POST, OPTIONS")
	pw := &preflightWriter{ResponseWriter: rec, copyAllow: true, fallbackMethods: []string{"GET"}}

	if _, err := pw.Write([]byte("body")); err != nil {
		t.Fatal(err)
	}
	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("Access-Control-Allow-Methods = %q; Write did not commit the header", got)
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200 inferred from the first Write", rec.Code)
	}
	if rec.Body.String() != "body" {
		t.Fatalf("body = %q", rec.Body.String())
	}
}

// A Write after an explicit WriteHeader must not re-commit or change the
// status.
func TestPreflightWriter_Write_after_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Allow", "GET")
	pw := &preflightWriter{ResponseWriter: rec, copyAllow: true}

	pw.WriteHeader(http.StatusAccepted)
	if _, err := pw.Write([]byte("x")); err != nil {
		t.Fatal(err)
	}
	if rec.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want 202 — Write must not re-commit the header", rec.Code)
	}
}

// Only the first WriteHeader copies Allow; a later one must not overwrite what
// was already advertised.
func TestPreflightWriter_only_the_first_WriteHeader_copies_Allow(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Allow", "GET")
	pw := &preflightWriter{ResponseWriter: rec, copyAllow: true}

	pw.WriteHeader(http.StatusNoContent)
	rec.Header().Set("Allow", "DELETE") // as if something changed it afterwards
	pw.WriteHeader(http.StatusOK)

	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want the value from the first commit", got)
	}
}

// With no Allow header — a 404, most likely — the fallback list is advertised.
// An empty Access-Control-Allow-Methods fails the preflight for every method,
// including ones the application does serve elsewhere.
func TestPreflightWriter_falls_back_when_the_router_sets_no_Allow(t *testing.T) {
	rec := httptest.NewRecorder()
	pw := &preflightWriter{
		ResponseWriter:  rec,
		copyAllow:       true,
		fallbackMethods: []string{"GET", "POST", "OPTIONS"},
	}
	pw.WriteHeader(http.StatusNotFound)

	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "GET, POST, OPTIONS" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want the fallback list", got)
	}
}

// An Access-Control-Allow-Methods already set by configuration wins over the
// fallback.
func TestPreflightWriter_keeps_a_configured_methods_header(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Access-Control-Allow-Methods", "PATCH")
	pw := &preflightWriter{ResponseWriter: rec, copyAllow: true, fallbackMethods: []string{"GET"}}
	pw.WriteHeader(http.StatusNoContent)

	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "PATCH" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want the configured value", got)
	}
}

// With copyAllow off the wrapper touches nothing.
func TestPreflightWriter_does_nothing_when_not_copying(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("Allow", "GET, POST")
	pw := &preflightWriter{ResponseWriter: rec, copyAllow: false, fallbackMethods: []string{"GET"}}
	pw.WriteHeader(http.StatusOK)

	if got := rec.Header().Get("Access-Control-Allow-Methods"); got != "" {
		t.Fatalf("Access-Control-Allow-Methods = %q, want it left alone", got)
	}
}

// http.ResponseController walks Unwrap to find a writer supporting what it
// needs. Returning anything else breaks it for every handler below.
func TestPreflightWriter_Unwrap(t *testing.T) {
	rec := httptest.NewRecorder()
	pw := &preflightWriter{ResponseWriter: rec}
	if got := pw.Unwrap(); got != http.ResponseWriter(rec) {
		t.Fatal("Unwrap did not return the wrapped writer")
	}
}

// Dropping an optional interface is a behaviour regression that only shows up
// when something downstream needs it (D45).
func TestPreflightWriter_Flush_forwards(t *testing.T) {
	fr := &flushRecorder{ResponseRecorder: httptest.NewRecorder()}
	pw := &preflightWriter{ResponseWriter: fr}
	pw.Flush()
	if fr.flushes != 1 {
		t.Fatalf("flushes = %d, want 1", fr.flushes)
	}
}

// And a writer that cannot flush must not panic.
func TestPreflightWriter_Flush_on_a_non_flusher(t *testing.T) {
	pw := &preflightWriter{ResponseWriter: &nonFlusher{hdr: make(http.Header)}}
	pw.Flush() // must not panic
}
