# rextension-cors — CORS for the router's own responses

`github.com/kryovyx/rextension-cors`. Part of the REX framework: developed in a `go.work` workspace
alongside its siblings, released as its own module.

## Boundaries

Depends on `rextension` only.

It decorates **the router's own responses**, including the ones the router
generates itself — 404, 405, problem details — so it must not assume it is
wrapping a user handler. Preflight replies are written by this module rather
than delegated onward.

This repo has no GitHub remote yet: the module was created in the workspace and
is committed locally but neither tagged nor pushed. The consumer that uses it
carries a temporary `replace` until it is.

## Working here

- **Never `go build`.** Syntax-check with `go vet ./...`.
- **`go test -race ./...` always.**
- **Tests are per branch, not per coverage number.** Every branch of every
  function gets its own case; the README's coverage figure is recomputed from
  a measurement, never hand-edited.
- **No `replace` directives** in `go.mod`.
- **Commits:** `<gitmoji><type>(<scope>): <description>` — feat, fix, docs,
  style, refactor, test, chore.
- `make check` here runs fmt, vet and race tests for this module alone.
- Default branch is `main`. **Never push without asking** — github
  authenticates with a hardware key that needs a physical tap, so an
  unattended push hangs and then fails.

Design decisions are numbered (D…/O…/W…) and recorded in the workspace this
module is developed in, not in this repo. If a rule here looks arbitrary, it is
load-bearing — ask before removing it.
