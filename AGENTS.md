# AGENTS.md

## Go Development Reference
- For any Go code changes, consult `./docs/GO_EFFECTIVE_GUIDE.md` first.
- Treat it as the default style and design guidance for code structure, naming, errors, comments, and concurrency choices.
- For package/module placement decisions, also consult `./docs/STANDARD_PROJECT_LAYOUT.md`.

## Routing Convention
- Serve the web app from root routes (`/` and non-API page paths).
- Keep machine/API endpoints under `/api/*`.

## Path Style
- In repository docs, comments, plans, and config examples, use workspace-relative paths (e.g. `cmd/api/main.go`) instead of machine-specific absolute paths.

## UI Stack Rules
- For server-rendered UI, default to `templ` components (avoid inline HTML string templates in Go files).
- For UI interactions, default to HTMX-driven flows before adding custom client-side `fetch` logic.
- Build reusable UI components (layout, cards, forms, collapsibles, pagination) and reuse them across pages.
- Prefer extending shared assets/modules (e.g. `internal/web/assets/js/ui.js`) over page-local duplicated scripts.
- Keep page templates/view models in `internal/web/pages` and reusable view pieces in `internal/web/components`.

## HTTP Organization
- Keep route registration in `internal/transport/http/routes.go`.
- Keep feature handlers split by file in `internal/transport/http/` (e.g. `gallery_page.go`, `settings_page.go`, `media_api.go`).
- Keep transport-layer middleware in `internal/transport/http/middleware.go` when it grows beyond simple route-local wrappers.
