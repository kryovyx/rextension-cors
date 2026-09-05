# rextension-cors

[![Go Version](https://img.shields.io/badge/go-1.27+-blue.svg)](https://golang.org/dl/)
[![Coverage](https://img.shields.io/badge/coverage-100.0%25-brightgreen.svg)](#)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Cross-Origin Resource Sharing for the [rex](https://github.com/kryovyx/rex)
framework.

```go
import (
    "github.com/kryovyx/rex"
    "github.com/kryovyx/rextension"
    cors "github.com/kryovyx/rextension-cors"
    sec "github.com/kryovyx/rextension-security"
)

// One allowlist, shared by CORS and CSRF.
browserOrigins := rextension.OriginPolicy{
    AllowedOrigins:   []string{"https://app.example.com"},
    AllowCredentials: true,
}

app := rex.New(
    cors.WithCORS(cors.NewConfig(
        cors.WithPolicy(browserOrigins),
    )),
    sec.WithSecurity(sec.NewConfig(
        sec.WithCSRFPolicy(browserOrigins),
        // …schemes…
    )),
)
```

## CORS is not CSRF protection

Worth stating first, because assuming otherwise is how applications get
compromised.

A **"simple" cross-origin request** — a form POST with
`application/x-www-form-urlencoded`, `multipart/form-data` or `text/plain` —
gets **no preflight**. The browser sends it, your server receives it, and your
server commits the state change. CORS then prevents the attacker's page from
*reading* the response, which is no consolation for a transfer that already
happened.

| Question | Answered by |
|---|---|
| May this origin **read** my responses? | CORS (this module) |
| Did this request really come **from my application**? | CSRF (`rextension-security`) |

They share an origin allowlist — `rextension.OriginPolicy` — and nothing else.
Configure both.

## What it does

- Adds `Access-Control-*` headers to cross-origin responses, **including error
  responses**. A browser will not let script read a response whose CORS headers
  are missing, so a 401 without them reaches the client as an opaque network
  failure rather than as "unauthorized".
- **Decorates the router's own `OPTIONS` response** rather than registering
  competing `OPTIONS` routes. The router already answers `OPTIONS` from its
  `Allow` set, and that answer is the accurate one — it is derived from the
  routes that actually exist, so `Access-Control-Allow-Methods` cannot drift
  out of step with them.
- Sets `Vary: Origin` on every origin-dependent response, allowed or refused.
  Without it a shared cache can serve a response carrying
  `Access-Control-Allow-Origin: https://a.example` to a request from
  `https://b.example`.
- Exposes the `X-RateLimit-*` headers by default, so a cross-origin client can
  read its remaining quota instead of discovering the limit by exceeding it.

## Configuration

| Option | Default | Notes |
|---|---|---|
| `WithAllowedOrigins(…)` | none | Exact origins: scheme, host, optional port |
| `WithAllowCredentials(bool)` | `false` | Incompatible with `"*"` |
| `WithPolicy(policy)` | — | Share one policy with CSRF |
| `WithAllowedMethods(…)` | router's `Allow` | Prefer the default |
| `WithAllowedHeaders(…)` | echoes requested | An allowlist that omits one reads as "CORS is broken" |
| `WithExposedHeaders(…)` | `X-RateLimit-*`, `Retry-After` | Replaces, not appends |
| `WithMaxAge(d)` | 10m | Browsers cap this (Chrome 2h, Firefox 24h) |

### Origins are matched exactly

No patterns. Pattern matching on origins is a recurring source of
over-permissive policies: a suffix check for `.example.com` matches
`evil-example.com`, and a prefix check for `https://app.example` matches
`https://app.example.attacker.com`. Both have appeared in real advisories.

Three mistakes are rejected at startup, because each one produces a policy that
looks configured and allows nothing:

| Written | Problem |
|---|---|
| `app.example.com` | No scheme; an `Origin` header always has one |
| `https://app.example.com/` | Trailing path; an `Origin` header never has one |
| `https://*.example.com` | Patterns are not supported |

### `"*"` with credentials is refused

Browsers reject `Access-Control-Allow-Origin: *` on a credentialed response, so
the combination silently blocks every cross-origin request it appears to
permit — and the failure reads as a server bug from the client's side. The
extension refuses to start rather than emit it.

### The default allows nothing

An empty allowlist refuses every cross-origin request, and the extension warns
about it at startup. That is deliberate: an extension that permitted any origin
out of the box would turn adding it into a policy decision its author did not
make.

## Ordering

Attached at `rextension.PriorityCORS` (200) — outside authentication (400) and
rate limiting (300). The headers have to be present whatever the inner layers
decide, or a cross-origin client cannot see the status they produced.

## Testing

```
make check      # gofmt + vet + go test -race
```

## Where this fits

New in the REX v0.3.0 release, at **v0.1.0** — there is nothing to migrate
from. [MIGRATION.md](MIGRATION.md) says what this module is, what it costs to
adopt, and links to the guide for every other module in that release.

## Contributing

**The framework is in alpha, and external contributions open at `v1.0.0`.**
Until then pull requests will be closed unmerged — but issues are very welcome.
Bug reports, questions and feature requests all feed into what `v1.0.0` looks
like.

See [CONTRIBUTING.md](CONTRIBUTING.md) for the rules that will apply, and
[COMMIT-CONVENTIONS.md](COMMIT-CONVENTIONS.md) for the commit format.
