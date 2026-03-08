package qr

import (
	"encoding/base64"
	"fmt"
	"strings"
)

// SVGDataURI renders a QR code as an SVG data URI.
// moduleSize controls pixel size of one QR module.
func SVGDataURI(payload string, moduleSize int) (string, error) {
	svg, err := SVG(payload, moduleSize)
	if err != nil {
		return "", err
	}
	return "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg)), nil
}

// SVG renders a QR code as raw SVG markup.
// moduleSize controls pixel size of one QR module.
func SVG(payload string, moduleSize int) (string, error) {
	if moduleSize <= 0 {
		moduleSize = 8
	}

	m, err := Matrix(payload)
	if err != nil {
		return "", err
	}
	return renderSVG(m, moduleSize), nil
}

func renderSVG(m [][]int, moduleSize int) string {
	quiet := 4
	size := len(m)
	total := (size + quiet*2) * moduleSize

	var b strings.Builder
	b.Grow(total * 2)

	b.WriteString(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" width="%d" height="%d" shape-rendering="crispEdges">`, total, total, total, total))
	b.WriteString(`<rect width="100%" height="100%" fill="white"/>`)

	for r := 0; r < size; r++ {
		for c := 0; c < size; c++ {
			if m[r][c] != 1 {
				continue
			}
			x := (c + quiet) * moduleSize
			y := (r + quiet) * moduleSize
			b.WriteString(fmt.Sprintf(`<rect x="%d" y="%d" width="%d" height="%d" fill="black"/>`, x, y, moduleSize, moduleSize))
		}
	}

	b.WriteString(`</svg>`)
	return b.String()
}
