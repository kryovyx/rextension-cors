# Commit conventions

Every repo in this project follows this. The file is byte-identical in each of
them, and `make claude-md` at the workspace root fails if a copy drifts.

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

## Signing

**Every commit is GPG-signed.** `commit.gpgsign` and `tag.gpgsign` are on, so
this is automatic — but it is worth knowing how to check and how to repair,
because one path bypasses it.

```sh
git log --format='%G? %h %s'          # G = good signature
git log --format='%G?' | grep -cv '^G'  # must print 0
```

**Wiki edits made in a browser are unsigned.** gollum commits through rugged,
which ignores `commit.gpgsign`. After editing a wiki in the browser, and
*before* pushing, re-sign:

```sh
git commit --amend --no-edit -S              # just the tip
git rebase --exec 'git commit --amend --no-edit -n -S' <base>   # a range
```

Re-signing rewrites commits and so changes their SHAs. Only ever do it to
commits that have not been pushed.

## Rewriting history

- **Never rewrite pushed history.** Not to reword, not to re-sign.
- Unpushed commits are fair game, and rewording them onto this convention is
  preferred over leaving a mixed history.
- On **2026-09-04** the 57 then-unpushed commits were reworded onto this
  format. Everything at or below each repo's remote tip predates it and uses
  the older `<gitmoji><type>(<scope>): <description>` form. That boundary is
  deliberate and is not to be "fixed".

## Branches

Modules commit to `main`. Wiki repos use `master`, because GitHub renders a
wiki only from `master` — a new wiki repo needs `git init -b master`.
