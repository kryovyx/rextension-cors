# Commit conventions

Every repo in the REX framework follows this, and the file is identical in each
of them. It applies to pull requests too, once contributions open — each
package's CONTRIBUTING.md explains why they are not open yet.

## Format

```text
<gitmoji> <short imperative summary>

<optional body: why, not what>

Co-Authored-By: Claude <model name> <noreply@anthropic.com>
```

## Subject

- **One gitmoji, then a single space, then the summary.** The gitmoji carries
  the type, which is why there is no `feat:` / `fix:` prefix — it would repeat
  what the emoji already says.
- **Imperative mood**, as if completing "this commit will …": *add*, *fix*,
  *bound*, *reject*. Not "added", not "adds".
- **Lower case** first word, except identifiers and proper nouns (`New`,
  `Run`, `OpenMetrics`).
- **No trailing period.**
- **72 characters at most**, gitmoji and its space included.
- Scope is not a field. When it matters, say it in the summary's own words —
  *bound the series count*, not *(metric): bound the count*.

## Gitmoji

The canonical list is [gitmoji.dev](https://gitmoji.dev). The ones in use
here, and what they mean:

| | meaning | | meaning |
|---|---|---|---|
| 🎉 | initialise a module or repo | ✨ | new capability |
| 🐛 | fix a defect | 💥 | breaking change |
| ♻️ | restructure without behaviour change | 🎨 | structure, formatting, ignores |
| 📝 | documentation | ✅ | tests |
| 👷 | build tooling, Makefiles | 🔒 | security fix |
| ➕ | add a dependency or submodule | 📌 | pin a version or pointer |

Pick the one that describes the *change*, not the file touched: a Makefile
edit that fixes a bug is 🐛, not 👷.

## Body

Optional, and only worth writing when the *why* is not obvious from the diff.

- Blank line after the subject. Wrap at 72.
- **Why, not what.** The diff already says what changed. The body says what
  problem it solves, what was considered and rejected, and what will break.
- Reference a decision by number (D…/O…/W…/S…) rather than restating it.

## Trailers

Exactly one trailer is permitted, and only on commits an agent co-authored:

```text
Co-Authored-By: Claude <model name> <noreply@anthropic.com>
```

Plain model name — `Claude Opus 5`, not `Claude Opus 5 (1M context)`. The
context window is a session setting, not an identity, and it reads as noise in
a year.

**Banned, without exception:**

- `Claude-Session: <url>` — a link only one person can open, which will rot.
  Some tooling appends it by default; the default is wrong here.
- `🤖 Generated with Claude Code` and any accompanying link.
- Any other generated footer, badge or advertisement.

A commit written without an agent carries no trailer at all.

`Signed-off-by` is reserved: a DCO sign-off is expected to become a requirement
when contributions open, and it will be the one other permitted trailer.

## Pull requests

Target `main`. One logical change per pull request; if the subject needs an
"and", it is probably two.
