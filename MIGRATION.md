# rextension-cors — new in this release

**v0.1.0** · Go 1.27 · still pre-1.0 (**alpha**)

There is nothing to migrate: this module did not exist before. This file
records what it is and how to adopt it, so that the whole framework has one
predictable place to look, and so the REX v0.3.0 upgrade has no gap where a
module's guide should be.

---

## What it is

CORS for a Rex application. It was never a module before; the behaviour existed
inside `rextension-security`, where it could not be used without also taking on
authentication.

```go
import (
    "github.com/kryovyx/rex"
    "github.com/kryovyx/rextension"
    cors "github.com/kryovyx/rextension-cors"
    sec  "github.com/kryovyx/rextension-security"
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
    )),
)
```

## Why it is worth adopting

Preflight is answered with an `Access-Control-Allow-Methods` derived from the
router's own `Allow` set, so the advertised methods cannot drift from the
routes that actually exist.

**CORS does not prevent CSRF.** A simple form POST gets no preflight at all;
CORS only stops the attacker *reading* the response, which is no consolation
for a transfer that already happened. Pair it with the CSRF protection in
[`rextension-security`](https://github.com/kryovyx/rextension-security/blob/main/MIGRATION.md), and give
both the same `OriginPolicy` value so the two can never disagree about who your
front end is.

## Adopting it

```sh
go get github.com/kryovyx/rextension-cors@v0.1.0
```

Nothing else changes. If you were relying on CORS handling inside
`rextension-security`, this is where it went.

---

## Verification

- [ ] `GOWORK=off go build ./...` passes — a workspace builds against sibling
      source and hides version skew, so this is the check that matters.
- [ ] `go test -race ./...` passes.

---

*Part of the REX v0.3.0 upgrade. The other guides:*

- [`rextension`](https://github.com/kryovyx/rextension/blob/main/MIGRATION.md) — v0.2.1 → **v0.3.0**
- [`rex`](https://github.com/kryovyx/rex/blob/main/MIGRATION.md) — v0.2.1 → **v0.3.0**
- [`dix`](https://github.com/kryovyx/dix/blob/main/MIGRATION.md) — v0.1.0 → **v0.2.0**
- [`rextension-security`](https://github.com/kryovyx/rextension-security/blob/main/MIGRATION.md) — v0.5.0 → **v0.6.0**
- [`rextension-validation`](https://github.com/kryovyx/rextension-validation/blob/main/MIGRATION.md) — v0.2.0 → **v0.3.0**
- [`rextension-openapi`](https://github.com/kryovyx/rextension-openapi/blob/main/MIGRATION.md) — v0.2.1 → **v0.3.0**
- [`rextension-health`](https://github.com/kryovyx/rextension-health/blob/main/MIGRATION.md) — v0.2.1 → **v0.3.0**
- [`rextension-metric`](https://github.com/kryovyx/rextension-metric/blob/main/MIGRATION.md) — v0.2.1 → **v0.3.0**
- [`rextension-swagger`](https://github.com/kryovyx/rextension-swagger/blob/main/MIGRATION.md) — v0.2.1 → **v0.3.0**
- [`corex`](https://github.com/kryovyx/corex/blob/main/MIGRATION.md) — new in this release, **v0.1.0**
- [`rextension-ratelimit`](https://github.com/kryovyx/rextension-ratelimit/blob/main/MIGRATION.md) — new in this release, **v0.1.0**
- [`wsxtension`](https://github.com/kryovyx/wsxtension/blob/main/MIGRATION.md) — new in this release, **v0.1.0**
- [`wsx`](https://github.com/kryovyx/wsx/blob/main/MIGRATION.md) — new in this release, **v0.1.0**
- [`wsxtension-asyncapi`](https://github.com/kryovyx/wsxtension-asyncapi/blob/main/MIGRATION.md) — new in this release, **v0.1.0**
- [`wsxtension-lens`](https://github.com/kryovyx/wsxtension-lens/blob/main/MIGRATION.md) — new in this release, **v0.1.0**
- [`rextension-wsx`](https://github.com/kryovyx/rextension-wsx/blob/main/MIGRATION.md) — new in this release, **v0.1.0**

*Every one of them stands alone; read only the ones for modules you use.*
