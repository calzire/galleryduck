package qr

import (
	"errors"
	"strings"
)

type qrSpec struct {
	version  int
	size     int
	dataCW   int
	eccCW    int
	alignPos []int
}

var specs = []qrSpec{
	{version: 1, size: 21, dataCW: 19, eccCW: 7, alignPos: nil},
	{version: 2, size: 25, dataCW: 34, eccCW: 10, alignPos: []int{6, 18}},
	{version: 3, size: 29, dataCW: 55, eccCW: 15, alignPos: []int{6, 22}},
	{version: 4, size: 33, dataCW: 80, eccCW: 20, alignPos: []int{6, 26}},
}

// ASCII returns a text QR for short payloads using byte mode + EC level L.
// It supports versions 1..4, enough for typical LAN URLs.
func ASCII(payload string) (string, error) {
	m, err := Matrix(payload)
	if err != nil {
		return "", err
	}
	return renderASCII(m), nil
}

// Matrix builds a QR matrix for short payloads using byte mode + EC level L.
// It supports versions 1..4, enough for typical LAN URLs.
func Matrix(payload string) ([][]int, error) {
	spec, err := chooseSpec(len(payload))
	if err != nil {
		return nil, err
	}

	data := buildDataCodewords(spec, []byte(payload))
	ecc := reedSolomon(data, spec.eccCW)
	codewords := append(data, ecc...)

	m := newMatrix(spec.size)
	drawFunctionPatterns(m, spec)
	writeData(m, codewords, 0)
	writeFormatInfo(m, spec.size, 0)

	return m, nil
}

func chooseSpec(n int) (qrSpec, error) {
	for _, s := range specs {
		if n <= byteCapacityL(s.version) {
			return s, nil
		}
	}
	return qrSpec{}, errors.New("payload too long for built-in QR encoder")
}

func byteCapacityL(version int) int {
	switch version {
	case 1:
		return 17
	case 2:
		return 32
	case 3:
		return 53
	case 4:
		return 78
	default:
		return 0
	}
}

func buildDataCodewords(spec qrSpec, payload []byte) []byte {
	bits := make([]int, 0, spec.dataCW*8)
	appendBits := func(val, count int) {
		for i := count - 1; i >= 0; i-- {
			bits = append(bits, (val>>i)&1)
		}
	}

	// Byte mode header.
	appendBits(0x4, 4)
	appendBits(len(payload), 8)
	for _, b := range payload {
		appendBits(int(b), 8)
	}

	// Terminator and byte alignment.
	capBits := spec.dataCW * 8
	if len(bits)+4 <= capBits {
		appendBits(0, 4)
	} else {
		for len(bits) < capBits {
			bits = append(bits, 0)
		}
	}
	for len(bits)%8 != 0 {
		bits = append(bits, 0)
	}

	data := make([]byte, 0, spec.dataCW)
	for i := 0; i < len(bits); i += 8 {
		b := 0
		for j := 0; j < 8; j++ {
			b = (b << 1) | bits[i+j]
		}
		data = append(data, byte(b))
	}

	pad := []byte{0xEC, 0x11}
	for len(data) < spec.dataCW {
		data = append(data, pad[len(data)%2])
	}
	return data
}

func newMatrix(size int) [][]int {
	m := make([][]int, size)
	for r := 0; r < size; r++ {
		m[r] = make([]int, size)
		for c := 0; c < size; c++ {
			m[r][c] = -1
		}
	}
	return m
}

func drawFunctionPatterns(m [][]int, spec qrSpec) {
	size := spec.size

	drawFinder(m, 0, 0)
	drawFinder(m, 0, size-7)
	drawFinder(m, size-7, 0)
	drawSeparators(m, 0, 0)
	drawSeparators(m, 0, size-7)
	drawSeparators(m, size-7, 0)

	// Timing patterns.
	for i := 8; i <= size-9; i++ {
		v := 0
		if i%2 == 0 {
			v = 1
		}
		if m[6][i] == -1 {
			m[6][i] = v
		}
		if m[i][6] == -1 {
			m[i][6] = v
		}
	}

	// Alignment patterns.
	if len(spec.alignPos) > 0 {
		for _, r := range spec.alignPos {
			for _, c := range spec.alignPos {
				if isFinderOverlap(size, r, c) {
					continue
				}
				drawAlignment(m, r-2, c-2)
			}
		}
	}

	// Dark module.
	m[4*spec.version+9][8] = 1

	// Reserve format info areas.
	for i := 0; i < 9; i++ {
		if m[8][i] == -1 {
			m[8][i] = 0
		}
		if m[i][8] == -1 {
			m[i][8] = 0
		}
	}
	for i := size - 8; i < size; i++ {
		if m[8][i] == -1 {
			m[8][i] = 0
		}
		if m[i][8] == -1 {
			m[i][8] = 0
		}
	}
}

func drawFinder(m [][]int, r0, c0 int) {
	for r := 0; r < 7; r++ {
		for c := 0; c < 7; c++ {
			rr := r0 + r
			cc := c0 + c
			switch {
			case r == 0 || r == 6 || c == 0 || c == 6:
				m[rr][cc] = 1
			case r == 1 || r == 5 || c == 1 || c == 5:
				m[rr][cc] = 0
			default:
				m[rr][cc] = 1
			}
		}
	}
}

func drawSeparators(m [][]int, r0, c0 int) {
	size := len(m)
	for i := -1; i <= 7; i++ {
		setIfInside(m, r0-1, c0+i, size, 0)
		setIfInside(m, r0+7, c0+i, size, 0)
		setIfInside(m, r0+i, c0-1, size, 0)
		setIfInside(m, r0+i, c0+7, size, 0)
	}
}

func setIfInside(m [][]int, r, c, size, val int) {
	if r >= 0 && c >= 0 && r < size && c < size && m[r][c] == -1 {
		m[r][c] = val
	}
}

func drawAlignment(m [][]int, r0, c0 int) {
	for r := 0; r < 5; r++ {
		for c := 0; c < 5; c++ {
			rr := r0 + r
			cc := c0 + c
			switch {
			case r == 0 || r == 4 || c == 0 || c == 4:
				m[rr][cc] = 1
			case r == 2 && c == 2:
				m[rr][cc] = 1
			default:
				m[rr][cc] = 0
			}
		}
	}
}

func isFinderOverlap(size, r, c int) bool {
	return (r == 6 && c == 6) || (r == 6 && c == size-7) || (r == size-7 && c == 6)
}

func writeData(m [][]int, codewords []byte, mask int) {
	bits := make([]int, 0, len(codewords)*8)
	for _, cw := range codewords {
		for i := 7; i >= 0; i-- {
			bits = append(bits, int((cw>>i)&1))
		}
	}

	size := len(m)
	bitIdx := 0
	up := true

	for col := size - 1; col > 0; col -= 2 {
		if col == 6 {
			col--
		}

		for i := 0; i < size; i++ {
			var row int
			if up {
				row = size - 1 - i
			} else {
				row = i
			}

			for dc := 0; dc < 2; dc++ {
				c := col - dc
				if m[row][c] != -1 {
					continue
				}
				b := 0
				if bitIdx < len(bits) {
					b = bits[bitIdx]
				}
				if applyMask(mask, row, c) {
					b ^= 1
				}
				m[row][c] = b
				bitIdx++
			}
		}
		up = !up
	}
}

func applyMask(mask, r, c int) bool {
	switch mask {
	case 0:
		return (r+c)%2 == 0
	default:
		return (r+c)%2 == 0
	}
}

func writeFormatInfo(m [][]int, size, mask int) {
	_ = mask                   // only mask 0 supported for now
	format := formatBits(0x08) // L + mask 0

	p1 := [][2]int{
		{8, 0}, {8, 1}, {8, 2}, {8, 3}, {8, 4}, {8, 5}, {8, 7}, {8, 8},
		{7, 8}, {5, 8}, {4, 8}, {3, 8}, {2, 8}, {1, 8}, {0, 8},
	}
	p2 := [][2]int{
		{size - 1, 8}, {size - 2, 8}, {size - 3, 8}, {size - 4, 8}, {size - 5, 8},
		{size - 6, 8}, {size - 7, 8}, {8, size - 8}, {8, size - 7}, {8, size - 6},
		{8, size - 5}, {8, size - 4}, {8, size - 3}, {8, size - 2}, {8, size - 1},
	}

	for i := 0; i < 15; i++ {
		b := (format >> (14 - i)) & 1
		m[p1[i][0]][p1[i][1]] = b
		m[p2[i][0]][p2[i][1]] = b
	}
}

func formatBits(data int) int {
	// BCH(15,5), generator 0x537, then XOR with 0x5412.
	v := data << 10
	for i := 14; i >= 10; i-- {
		if ((v >> i) & 1) == 1 {
			v ^= 0x537 << (i - 10)
		}
	}
	code := (data << 10) | (v & 0x3FF)
	return code ^ 0x5412
}

func renderASCII(m [][]int) string {
	var b strings.Builder
	quiet := 2
	size := len(m)

	white := "  "
	black := "██"

	for i := 0; i < quiet; i++ {
		for j := 0; j < size+2*quiet; j++ {
			b.WriteString(white)
		}
		b.WriteByte('\n')
	}

	for r := 0; r < size; r++ {
		for i := 0; i < quiet; i++ {
			b.WriteString(white)
		}
		for c := 0; c < size; c++ {
			if m[r][c] == 1 {
				b.WriteString(black)
			} else {
				b.WriteString(white)
			}
		}
		for i := 0; i < quiet; i++ {
			b.WriteString(white)
		}
		b.WriteByte('\n')
	}

	for i := 0; i < quiet; i++ {
		for j := 0; j < size+2*quiet; j++ {
			b.WriteString(white)
		}
		b.WriteByte('\n')
	}

	return b.String()
}

func reedSolomon(data []byte, eccLen int) []byte {
	gen := rsGenerator(eccLen)
	msg := make([]byte, len(data)+eccLen)
	copy(msg, data)
	for i := 0; i < len(data); i++ {
		coef := msg[i]
		if coef == 0 {
			continue
		}
		for j := 0; j < len(gen); j++ {
			msg[i+j] ^= gfMul(gen[j], coef)
		}
	}
	return msg[len(data):]
}

func rsGenerator(degree int) []byte {
	g := []byte{1}
	for i := 0; i < degree; i++ {
		g = polyMul(g, []byte{1, gfPow(2, i)})
	}
	return g
}

func polyMul(a, b []byte) []byte {
	out := make([]byte, len(a)+len(b)-1)
	for i, av := range a {
		for j, bv := range b {
			out[i+j] ^= gfMul(av, bv)
		}
	}
	return out
}

func gfPow(x byte, p int) byte {
	out := byte(1)
	for i := 0; i < p; i++ {
		out = gfMul(out, x)
	}
	return out
}

func gfMul(a, b byte) byte {
	var p byte
	for b > 0 {
		if b&1 != 0 {
			p ^= a
		}
		hi := a & 0x80
		a <<= 1
		if hi != 0 {
			a ^= 0x1D // 0x11D without the high bit in byte arithmetic
		}
		b >>= 1
	}
	return p
}
