# GalleryDuck

GalleryDuck is a lightweight self-hosted media gallery written in Go. It scans local folders for images, videos, and audio files, then serves a clean web UI for browsing, filtering, and slideshow playback.

## Features

- Local-first gallery for images, videos, and audio
- Server-rendered UI using `templ` + HTMX patterns
- Media filtering by type, subtype, year, date, and search
- Slideshow mode with configurable defaults
- Settings page for library paths and UI/runtime preferences
- JSON API endpoints under `/api/*`
- SQLite health endpoint support

## Tech Stack

- Go (`go1.25`)
- `templ` for server-rendered components
- HTMX for interactions
- Tailwind CSS for styling
- SQLite (`github.com/mattn/go-sqlite3`)

## Requirements

- Go 1.25+
- `make`
- macOS (current `tailwind-install` target downloads the macOS binary)

Optional tools (auto-installed by Make targets when missing):

- `templ` (`github.com/a-h/templ/cmd/templ`)
- `air` (`github.com/air-verse/air`) for live reload

## Quick Start

1. Clone the repository.
2. Set environment variables (example below).
3. Run the app.

```bash
export GALLERYDUCK_DB_URL=galleryduck.db
# Optional: default scanned directory on first run
export GALLERYDUCK_LIBRARY_PATH=.
# Optional: server port (default is 8787)
export PORT=8787

make run
```

Open:

- `http://localhost:8787`
- Settings page: `http://localhost:8787/settings`

## Configuration

GalleryDuck persists runtime config to:

- macOS/Linux: `~/.galleryduck/config.json`
- Windows: `%APPDATA%/galleryduck/config.json`

Main settings include:

- `library_paths` (one or more media roots)
- `port` (used when `PORT` environment variable is not set)
- `theme`
- `default_sort`
- `default_view`
- `pagination_mode`
- slideshow defaults (`speed_ms`, `transition`, `autoplay`, `loop`, `fullscreen`)

`PORT` environment variable takes precedence over saved config port.

## Development

Run full build + tests:

```bash
make all
```

Build binary:

```bash
make build
```

Run application:

```bash
make run
```

Live reload (Go + Tailwind watch):

```bash
make watch
```

Run tests:

```bash
make test
```

Format Go files:

```bash
make fmt
```

Clean generated binary:

```bash
make clean
```

## API Endpoints

Selected endpoints:

- `GET /api/health`
- `GET /api/media`
- `GET /api/media/file?path=...`
- `POST /api/index/rebuild`
- `POST /api/theme`
- `GET /api/slideshow/items`
- `GET /api/qr.svg?url=...`

Web routes:

- `GET /`
- `GET /settings`
- `GET /slideshow`

## Project Layout

```text
cmd/api/main.go                     # app entrypoint
internal/app/gallery/               # media scanning, config, query logic
internal/transport/http/            # handlers, routes, HTTP server
internal/web/components/            # reusable templ components
internal/web/pages/                 # page templates/view models
internal/web/assets/                # static assets
internal/store/db/                  # sqlite connection + health
docs/                               # engineering guides
```

## Contributing

1. Fork and create a feature branch.
2. Keep routing conventions:
   - UI routes at `/...`
   - API routes under `/api/*`
3. Run checks before opening a PR:

```bash
make fmt
make test
make build
```

## License

Licensed under the terms in [LICENSE](LICENSE).
