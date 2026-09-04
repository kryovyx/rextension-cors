# Contributing

> ## ⚠ Alpha — contributions are not open yet
>
> This module is part of the **REX** framework, which is in **alpha**. Exported
> interfaces change without a deprecation cycle, and several are expected to
> change again before they settle.
>
> **External contributions open at `v1.0.0`.** Until then, pull requests will be
> closed unmerged. That is not a judgement on the work — it is that a patch
> written against an interface which is about to change costs its author more
> than it saves anyone, and reviewing it commits the project to a shape it has
> not chosen yet.
>
> The rules below are published **now, in advance**, so that nothing about them
> is a surprise when contributions do open.
>
> **What is welcome today: issues.** Bug reports, questions, and feature
> requests all feed directly into what `v1.0.0` looks like, and cost nothing to
> act on later.

---

## Reporting a bug

Open an issue with:

- the module and version (`go list -m all | grep kryovyx`),
- your Go version (`go version`),
- what you expected and what happened instead,
- a **minimal** reproducer — a failing test is ideal, a `main.go` is fine.

"Minimal" matters more than "complete". A twenty-line reproducer gets fixed;
an application does not.

## Requesting a feature

Open an issue describing the **problem**, not the API you have in mind. The
framework's shape is still being decided, and a problem statement can be
solved in a way that fits; a proposed method signature usually cannot.

Say what you are doing today instead, and why it is unsatisfactory.

## Reporting a security issue

**Do not open a public issue.**

Use **Security → Report a vulnerability** on this repository, which opens a
private advisory. If that is not available, email `mail@kryovyx.net` with
`SECURITY` in the subject.

Expect an acknowledgement within a few days. This is alpha software with a
single maintainer — there is no formal response-time commitment, and saying so
plainly is better than publishing one that will not be met.

---

# The rules

These apply from `v1.0.0`. They are the same rules the maintainer works under.

## Setup

Go **1.27** or later; every module declares `go 1.27`. Modules are developed
together in a `go.work` workspace and released independently, so a change here
can break a sibling without breaking this module.

## Verifying

```sh
make check      # gofmt, go vet, and go test -race for this module
```

`make check` must pass before a pull request is opened. Two rules inside it are
not stylistic:

- **Never `go build` a module.** Use `go vet ./...` to syntax-check. A library
  has no binary to produce, and `go build` hides vet's findings.
- **`go test -race` always.** Not "when it looks concurrent". A router is
  concurrent by construction, and every extension runs inside one.

## Tests

**Tests are written per branch, not per coverage number.** Every branch of
every function gets its own case: a function with one `if` needs at least two
tests, one for each side of it.

Coverage is the byproduct of doing that, not the target. Do not write a test
whose purpose is to move the percentage, and **do not hand-edit the coverage
figure in `README.md`** — it is recomputed from a measurement, and a number
nobody remeasures is read as a fact when it is only a memory of one.

## Dependencies

**The standard library, and nothing else.** One module has a third-party
runtime dependency, for reasons argued at the time; it is the exception and it
is not a precedent.

A pull request that adds a dependency will normally be declined. If you believe
one is genuinely necessary, open an issue and argue it **before** writing the
code — including what it costs every consumer downstream.

**No `replace` directives** in `go.mod`. A `replace` in a published module
breaks every consumer that does not have the same directory layout.

## Architecture

The dependency direction is fixed, and a pull request that crosses it will be
declined regardless of how well it is written:

- The **contract** module holds interfaces and types only, and imports the
  standard library only. Its import list is every extension's dependency graph.
- The **core** may depend on the contract.
- An **extension** may depend on the contract. Never on the core, and never on
  another extension.

If a change seems to require crossing one of those lines, the design needs
discussing in an issue first — that is usually a sign the contract is missing
something.

## Commits

See [COMMIT-CONVENTIONS.md](COMMIT-CONVENTIONS.md): a gitmoji, a space, and an
imperative summary, with an optional body explaining **why** rather than what.

## Pull requests

- Target `main`.
- One logical change per pull request. If the subject needs an "and", it is
  probably two pull requests.
- Link the issue it addresses. Changes without a prior issue are likely to be
  asked for one, because the discussion is the part that is hard to redo.
- Explain the reasoning in the description. The diff already says what changed.

## Licensing

This project is MIT licensed. By submitting a contribution you agree that it is
provided under the same licence and that you have the right to submit it.

A DCO `Signed-off-by` requirement is expected to be added when contributions
open, which will make that agreement explicit per commit rather than implicit.

## Conduct

Be straightforward and assume good faith. Critique the code, not the person
who wrote it; if you are told no, ask for the reasoning rather than repeating
the request.

Harassment, personal attacks, and demanding unpaid work from volunteers are not
tolerated, and the maintainer will close or block without debate. There is no
separate code of conduct document at this stage because there is not yet a
community to govern — that changes with `v1.0.0`.
