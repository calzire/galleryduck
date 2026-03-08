# Standard Project Layout (GalleryDuck)

This document defines the default layout for this repository.

## Design Principles

1. Keep `cmd/` tiny.
- `main.go` should only load config, wire dependencies, and start/stop services.

2. Keep domain and application logic independent from transport/framework details.
- HTTP/router/template/db packages depend inward on app/domain; not the reverse.

3. Prefer `internal/` for all app code unless intentionally publishing a reusable library.
- Do not add `/pkg` unless code is meant for external consumption.

4. Group by responsibility and dependency flow, not by technical layer sprawl.
- Avoid generic buckets like `utils`, `helpers`, `common`.

## Standard Layout

```text
galleryduck/
  cmd/
    api/
      main.go                # process bootstrap only

  internal/
    app/
      gallery/
        service.go           # use-cases, orchestration, interfaces used by app

    domain/
      media/
        model.go             # entities/value objects (framework-agnostic)

    transport/
      http/
        server.go            # http.Server construction
        routes.go            # route registration
        middleware.go        # HTTP middleware
        gallery_page.go      # gallery page handlers
        settings_page.go     # settings page handlers
        media_api.go         # API handlers

    store/
      db/
        database.go          # db wiring, adapters

    web/
      pages/                 # top-level page templates/view models
      components/            # reusable UI components
      assets/
        css/
        js/

  docs/
    GO_EFFECTIVE_GUIDE.md
    STANDARD_PROJECT_LAYOUT.md

  Makefile
  go.mod
  go.sum
  tailwind.config.js
```

## Rules For New Code

- `cmd/*` is bootstrap only: config loading, dependency wiring, process lifecycle.
- `internal/transport/http` owns routing, handlers, HTTP middleware, and HTTP request/response DTOs.
- Within `internal/transport/http`, keep handlers split by feature file (`gallery_page.go`, `settings_page.go`, `media_api.go`); keep route wiring in `routes.go`.
- `internal/store/*` owns persistence adapters and database integration only.
- `internal/app/*` owns use-cases and orchestration; it coordinates transport/store via interfaces.
- `internal/domain/*` owns core entities/value objects and domain rules; it must not import transport or store packages.
- `internal/web/pages` owns page-level templ files/view data.
- `internal/web/components` owns reusable templ components used by multiple pages.
- `internal/web/assets` owns static assets used by the HTTP layer.
- API endpoints must be mounted under `/api/*`.
- Web pages must be served from root routes (`/` and non-API paths).
- Keep dependencies one-way where possible: `transport` -> `app` -> `domain`, and `store` -> `domain`.

## What We Intentionally Avoid

- Adding `/pkg` prematurely.
- Fat `main.go` files.
- Cross-import cycles between transport/store/domain.
- Mixing web page routes and API routes outside the `/` vs `/api/*` split.
- Re-introducing legacy paths (`internal/server`, `internal/database`, `cmd/web`) for new code.
