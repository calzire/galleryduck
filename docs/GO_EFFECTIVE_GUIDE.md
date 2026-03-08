# Go Effective Guide For Codex

Source reference: [Effective Go](https://go.dev/doc/effective_go)

This project uses these conventions as the default for Go implementation decisions.

## 1) Formatting and layout
- Always format Go code with `gofmt`.
- Keep imports grouped and minimal; remove unused imports.
- Prefer short declarations (`:=`) where they improve clarity.

## 2) Package and file design
- Pick short, lower-case package names with no underscores.
- Avoid stutter: `package server` + `server.New`, not `server.NewServerServer`.
- Keep packages focused on a single responsibility.

## 3) Naming
- Use mixedCaps (camelCase), not snake_case.
- Keep names short but meaningful; prefer clarity over abbreviation.
- Keep common initialisms uppercase (`ID`, `URL`, `HTTP`, `API`).
- Name interfaces by behavior (`Reader`, `Store`, `Formatter`).
- Getter methods should usually omit `Get` (`User.ID()`, not `GetID()`).

## 4) Functions and methods
- Make function signatures small and explicit.
- Return early on error; keep the happy path unindented.
- Use named results only when they improve readability.
- Method receiver names should be short and consistent (`s *Server`, `r *Repo`).

## 5) Errors
- Return `error` values; avoid panics except unrecoverable startup/programmer errors.
- Error strings should be lowercase and usually without punctuation.
- Wrap errors with context using `%w` and `fmt.Errorf`.
- Do not lose root cause details when propagating errors.

## 6) Comments and docs
- Add doc comments for exported types, funcs, methods, and constants.
- Start doc comments with the identifier name.
- Keep comments focused on intent/constraints, not obvious mechanics.

## 7) Data and control flow
- Design types so zero values are useful when possible.
- Prefer slices over arrays for most APIs.
- Use maps for lookup, but initialize before writes.
- Keep conditionals simple; reduce nested branching with early returns.

## 8) Interfaces and composition
- Define small interfaces at point of use.
- Accept interfaces, return concrete types where practical.
- Prefer composition over inheritance-style layering.

## 9) Concurrency
- Do not communicate by sharing memory; share memory by communicating.
- Use channels and goroutines deliberately; avoid accidental goroutine leaks.
- Guard shared mutable state (`sync.Mutex`, `sync.RWMutex`, atomics) when needed.
- Ensure cancellation/timeouts are wired via `context.Context`.

## 10) Practical review checklist
- Is this code `gofmt`-clean?
- Are names idiomatic and free of stutter?
- Are errors wrapped with actionable context?
- Are exported APIs documented?
- Is concurrency safe and cancelable?
- Is there a simpler design with smaller interfaces?

## Notes
- Effective Go is old but still foundational.
- For language/library updates, also check current package docs and release notes.

