# `internal/qr` Implementation Notes

This package provides a small, dependency-free QR encoder used by GalleryDuck.

## Why this exists
- We wanted QR generation without adding a third-party QR library.
- It is currently used to show LAN URL QR in settings and startup logs.

## Files
- `ascii.go`
  - Core QR encoding logic.
  - Exposes:
    - `Matrix(payload string) ([][]int, error)` - builds QR module matrix.
    - `ASCII(payload string) (string, error)` - renders matrix as terminal text blocks.
- `svg.go`
  - SVG rendering layer on top of `Matrix`.
  - Exposes:
    - `SVGDataURI(payload string, moduleSize int) (string, error)` - returns `data:image/svg+xml;base64,...`.

## Encoding approach
- Mode: **Byte mode**
- Error correction: **Level L**
- Mask: **Mask 0 only**
- Versions supported: **1..4**

This is intentionally scoped for short payloads like local LAN URLs.

## High-level flow
1. Pick QR version based on payload length (`chooseSpec`, `byteCapacityL`).
2. Build data codewords:
   - Mode indicator (byte mode)
   - Length
   - Payload bytes
   - Terminator + pad bytes (`0xEC`, `0x11`)
3. Generate Reed-Solomon error correction codewords (`reedSolomon`).
4. Build matrix and draw function patterns:
   - Finder patterns + separators
   - Timing patterns
   - Alignment patterns (for higher versions)
   - Dark module
   - Reserved format info areas
5. Write data bits in QR zig-zag pattern with mask 0.
6. Write format info bits (BCH + mask).
7. Render:
   - ASCII text (`renderASCII`) for logs
   - SVG (`renderSVG`) for browser UI

## Math internals
- RS and BCH calculations are implemented in-package.
- Galois Field multiplication is done with polynomial reduction (`gfMul`).

## Current limits / tradeoffs
- Only EC level L.
- Only mask pattern 0.
- Only versions 1..4.
- No advanced mask scoring/selection.

These limits keep implementation small and stable for current URL use-cases.

## Failure behavior
- If payload is too long for supported versions, functions return an error.
- Callers (startup logs/settings page) fall back gracefully:
  - no crash
  - QR just not shown

## Where it is used
- Startup logs:
  - `cmd/api/main.go`
- Settings page QR card:
  - `internal/transport/http/pages.go`

## Extension ideas
- Add version support beyond 4.
- Add mask scoring and automatic best-mask selection.
- Add PNG renderer (if needed for downloads/sharing).
- Add tests with known-good vectors.
