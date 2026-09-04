// Command tuigif rasterises the swallow TUI into an animated GIF (docs/tour.gif)
// entirely offline — no terminal recorder, no ffmpeg. It drives the real model
// through tui.DemoFrames, parses each frame's ANSI into a cell grid, draws the
// grid with a monospaced system font, and encodes the frames with image/gif.
//
// Usage:
//
//	go run ./cmd/tuigif -out ../docs/tour.gif
//
// It reads a monospaced TTF/TTC with box-drawing + block glyphs (Menlo by
// default) so the wordmark and borders render faithfully.
package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"

	"github.com/ParkWardRR/swallow-ap-hk07-firmware-tools/internal/tui"
)

func main() {
	out := flag.String("out", "../docs/tour.gif", "output GIF path")
	shots := flag.String("shots", "", "if set, write per-step PNG screenshots to this dir instead of a GIF")
	fontPath := flag.String("font", "/System/Library/Fonts/Menlo.ttc", "monospaced TTF/TTC path")
	size := flag.Float64("size", 13, "font point size")
	cols := flag.Int("cols", 106, "terminal columns")
	rows := flag.Int("rows", 30, "terminal rows")
	delay := flag.Int("delay", 7, "per-frame delay in 100ths of a second")
	flag.Parse()

	// Force 24-bit colour: output isn't a TTY, so lipgloss would otherwise strip it.
	lipgloss.SetColorProfile(termenv.TrueColor)

	r, err := newRenderer(*fontPath, *size, *cols, *rows)
	if err == nil {
		if *shots != "" {
			err = r.writeShots(*shots)
		} else {
			err = r.writeGIF(*out, *delay)
		}
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "tuigif:", err)
		os.Exit(1)
	}
}

type renderer struct {
	face                 font.Face
	cols, rows           int
	cellW, cellH, baseUp int
	pal                  color.Palette
}

func newRenderer(fontPath string, size float64, cols, rows int) (*renderer, error) {
	face, cellW, cellH, baseline, err := loadFace(fontPath, size)
	if err != nil {
		return nil, err
	}
	return &renderer{face: face, cols: cols, rows: rows, cellW: cellW, cellH: cellH, baseUp: baseline, pal: buildPalette()}, nil
}

// frameRGBA rasterises one ANSI frame to an RGBA image.
func (r *renderer) frameRGBA(frame string) *image.RGBA {
	grid := parseANSI(frame, r.cols, r.rows)
	img := image.NewRGBA(image.Rect(0, 0, r.cols*r.cellW, r.rows*r.cellH))
	drawer := &font.Drawer{Dst: img, Face: r.face}
	for y, line := range grid {
		for x, c := range line {
			rect := image.Rect(x*r.cellW, y*r.cellH, x*r.cellW+r.cellW, y*r.cellH+r.cellH)
			draw.Draw(img, rect, &image.Uniform{c.bg}, image.Point{}, draw.Src)
			if c.r == ' ' || c.r == 0 {
				continue
			}
			if c.r == '█' { // fill the whole cell so neon blocks tile seamlessly
				draw.Draw(img, rect, &image.Uniform{c.fg}, image.Point{}, draw.Src)
				continue
			}
			drawer.Src = &image.Uniform{c.fg}
			drawer.Dot = fixed.P(x*r.cellW, y*r.cellH+r.baseUp)
			drawer.DrawString(string(c.r))
		}
	}
	return img
}

func (r *renderer) writeGIF(out string, delay int) error {
	frames := tui.DemoFrames("0.4.0", r.cols, r.rows)
	if len(frames) == 0 {
		return fmt.Errorf("no frames produced")
	}
	g := &gif.GIF{LoopCount: 0}
	for _, f := range frames {
		pimg := image.NewPaletted(image.Rect(0, 0, r.cols*r.cellW, r.rows*r.cellH), r.pal)
		draw.Draw(pimg, pimg.Bounds(), r.frameRGBA(f), image.Point{}, draw.Src)
		g.Image = append(g.Image, pimg)
		g.Delay = append(g.Delay, delay)
		g.Disposal = append(g.Disposal, gif.DisposalNone)
	}
	f, err := os.Create(out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := gif.EncodeAll(f, g); err != nil {
		return err
	}
	fmt.Printf("wrote %s — %d frames, %dx%d px (cell %dx%d)\n", out, len(g.Image), r.cols*r.cellW, r.rows*r.cellH, r.cellW, r.cellH)
	return nil
}

func (r *renderer) writeShots(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	names := []string{"01-discover", "02-connect", "03-backup", "04-safeguards", "05-identity", "06-install", "07-verify"}
	for step, name := range names {
		img := r.frameRGBA(tui.StepFrame("0.4.0", r.cols, r.rows, step))
		path := dir + "/" + name + ".png"
		f, err := os.Create(path)
		if err != nil {
			return err
		}
		if err := png.Encode(f, img); err != nil {
			f.Close()
			return err
		}
		f.Close()
		fmt.Printf("shot %s\n", path)
	}
	return nil
}

func loadFace(path string, size float64) (font.Face, int, int, int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	coll, err := opentype.ParseCollection(data)
	if err != nil {
		return nil, 0, 0, 0, fmt.Errorf("parse font %s: %w", path, err)
	}
	ft, err := coll.Font(0)
	if err != nil {
		return nil, 0, 0, 0, err
	}
	face, err := opentype.NewFace(ft, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingFull})
	if err != nil {
		return nil, 0, 0, 0, err
	}
	adv, ok := face.GlyphAdvance('M')
	if !ok {
		return nil, 0, 0, 0, fmt.Errorf("font has no 'M' glyph")
	}
	m := face.Metrics()
	cellW := adv.Ceil()
	cellH := (m.Ascent + m.Descent).Ceil()
	baseline := m.Ascent.Ceil()
	return face, cellW, cellH, baseline, nil
}

// ---- ANSI → cell grid ---------------------------------------------------------

type cell struct {
	r      rune
	fg, bg color.RGBA
}

var (
	defFg = color.RGBA{0xc0, 0xca, 0xf5, 0xff}
	defBg = color.RGBA{0x1a, 0x1b, 0x26, 0xff}
)

func parseANSI(s string, cols, rows int) [][]cell {
	grid := make([][]cell, rows)
	for y := range grid {
		grid[y] = make([]cell, cols)
		for x := range grid[y] {
			grid[y][x] = cell{r: ' ', fg: defFg, bg: defBg}
		}
	}
	fg, bg := defFg, defBg
	x, y := 0, 0
	rs := []rune(s)
	for i := 0; i < len(rs); i++ {
		c := rs[i]
		switch {
		case c == '\n':
			x = 0
			y++
		case c == '\x1b' && i+1 < len(rs) && rs[i+1] == '[':
			// read until the terminating 'm'
			j := i + 2
			for j < len(rs) && rs[j] != 'm' {
				j++
			}
			applySGR(string(rs[i+2:j]), &fg, &bg)
			i = j
		default:
			if y < rows && x < cols {
				grid[y][x] = cell{r: c, fg: fg, bg: bg}
			}
			x++
		}
	}
	return grid
}

func applySGR(body string, fg, bg *color.RGBA) {
	if body == "" || body == "0" {
		*fg, *bg = defFg, defBg
		return
	}
	toks := strings.Split(body, ";")
	for i := 0; i < len(toks); i++ {
		switch toks[i] {
		case "0":
			*fg, *bg = defFg, defBg
		case "1", "22", "39":
			// bold / not-bold / default-fg — ignore (default-fg is rare here)
		case "49":
			*bg = defBg
		case "38", "48":
			if i+4 < len(toks) && toks[i+1] == "2" {
				col := color.RGBA{atob(toks[i+2]), atob(toks[i+3]), atob(toks[i+4]), 0xff}
				if toks[i] == "38" {
					*fg = col
				} else {
					*bg = col
				}
				i += 4
			}
		}
	}
}

func atob(s string) uint8 {
	n, _ := strconv.Atoi(s)
	if n < 0 {
		n = 0
	}
	if n > 255 {
		n = 255
	}
	return uint8(n)
}

// ---- palette ------------------------------------------------------------------

// buildPalette seeds a 256-colour palette with the exact theme + shimmer colours
// (so neon blocks stay crisp) and fills the rest with anti-aliasing blends of the
// text colours over the two backgrounds plus a grey ramp.
func buildPalette() color.Palette {
	var pal color.Palette
	seen := map[color.RGBA]bool{}
	add := func(c color.RGBA) {
		if len(pal) >= 256 || seen[c] {
			return
		}
		seen[c] = true
		pal = append(pal, c)
	}

	for _, hx := range tui.PaletteHex() {
		add(hexRGBA(hx))
	}
	// AA blends: each foreground fading into each background.
	bgs := []color.RGBA{hexRGBA("#1a1b26"), hexRGBA("#24283b"), hexRGBA("#33467c")}
	fgs := []color.RGBA{hexRGBA("#c0caf5"), hexRGBA("#a9b1d6"), hexRGBA("#565f89"),
		hexRGBA("#7aa2f7"), hexRGBA("#7dcfff"), hexRGBA("#9ece6a"), hexRGBA("#f7768e"),
		hexRGBA("#e0af68"), hexRGBA("#bb9af7"), hexRGBA("#3b4261")}
	for _, b := range bgs {
		for _, fgc := range fgs {
			for s := 1; s <= 4; s++ {
				add(blend(b, fgc, float64(s)/5))
			}
		}
	}
	for i := 0; i < 256 && len(pal) < 256; i++ {
		g := uint8(i)
		add(color.RGBA{g, g, g, 0xff})
	}
	return pal
}

func blend(a, b color.RGBA, t float64) color.RGBA {
	lerp := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.RGBA{lerp(a.R, b.R), lerp(a.G, b.G), lerp(a.B, b.B), 0xff}
}

func hexRGBA(h string) color.RGBA {
	h = strings.TrimPrefix(h, "#")
	if len(h) != 6 {
		return defFg
	}
	v, _ := strconv.ParseUint(h, 16, 32)
	return color.RGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
}
