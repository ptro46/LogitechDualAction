//go:build gui && darwin && cgo

package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"runtime"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"
)

// All window, drawing and animation code in this application is Go.
// IOKit remains confined to the existing HID backend shared with the console.
type graphicGame struct {
	state         graphicState
	states        <-chan graphicState
	demo          bool
	ticks         int
	preview       string
	exported      bool
	exportErr     error
	regular, bold *text.GoTextFaceSource
}

func graphicReader(ctx context.Context) (<-chan graphicState, <-chan struct{}) {
	states := make(chan graphicState, 1)
	done := make(chan struct{})
	go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		defer close(done)
		if err := openHID(0x046d, 0xc216); err != nil {
			fmt.Fprintln(os.Stderr, err)
			states <- graphicState{status: -2, hat: -1}
			return
		}
		defer closeHID()
		ticker := time.NewTicker(time.Second / 60)
		defer ticker.Stop()
		for {
			s, status := readHID()
			state := graphicFromSnapshot(s, status)
			select {
			case <-states:
			default:
			}
			states <- state
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return states, done
}

func (g *graphicGame) Update() error {
	if g.exported {
		if g.exportErr != nil {
			return g.exportErr
		}
		return ebiten.Termination
	}
	if ebiten.IsKeyPressed(ebiten.KeyEscape) || (ebiten.IsKeyPressed(ebiten.KeyMeta) && ebiten.IsKeyPressed(ebiten.KeyQ)) {
		return ebiten.Termination
	}
	if g.demo {
		g.state = graphicDemoState(float64(g.ticks) / 60)
	} else {
		select {
		case g.state = <-g.states:
		default:
		}
	}
	g.ticks++
	return nil
}
func (g *graphicGame) Layout(w, h int) (int, int) { return 960, 720 }
func (g *graphicGame) label(dst *ebiten.Image, s string, x, y, size float64, c color.Color, bold, centered bool) {
	source := g.regular
	if bold {
		source = g.bold
	}
	face := &text.GoTextFace{Source: source, Size: size}
	if centered {
		w, _ := text.Measure(s, face, size*1.2)
		x -= w / 2
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	text.Draw(dst, s, face, op)
}
func rgb(hex int) color.NRGBA { return color.NRGBA{uint8(hex >> 16), uint8(hex >> 8), uint8(hex), 255} }
func fillPath(dst *ebiten.Image, p *vector.Path, c color.Color) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.FillPath(dst, p, nil, op)
}
func strokePath(dst *ebiten.Image, p *vector.Path, c color.Color, width float32) {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	vector.StrokePath(dst, p, &vector.StrokeOptions{Width: width, LineJoin: vector.LineJoinRound}, op)
}
func circle(dst *ebiten.Image, x, y, r float32, fill, edge color.Color) {
	vector.DrawFilledCircle(dst, x, y, r, fill, true)
	vector.StrokeCircle(dst, x, y, r, 1.5, edge, true)
}
func box(dst *ebiten.Image, x, y, w, h, r float32, fill, edge color.Color) {
	var p vector.Path
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.QuadTo(x+w, y, x+w, y+r)
	p.LineTo(x+w, y+h-r)
	p.QuadTo(x+w, y+h, x+w-r, y+h)
	p.LineTo(x+r, y+h)
	p.QuadTo(x, y+h, x, y+h-r)
	p.LineTo(x, y+r)
	p.QuadTo(x, y, x+r, y)
	p.Close()
	fillPath(dst, &p, fill)
	strokePath(dst, &p, edge, 1.5)
}
func (g *graphicGame) button(dst *ebiten.Image, n int, x, y, r float32) {
	fill, edge, ink := rgb(0x202a36), rgb(0x536174), rgb(0x738094)
	if g.state.available[n] {
		ink = rgb(0xe5edf7)
	}
	if g.state.buttons[n] {
		circle(dst, x, y, r+7, color.NRGBA{86, 224, 180, 30}, color.Transparent)
		fill, edge, ink = rgb(0x66edbd), rgb(0xa8ffde), rgb(0x0c342a)
	}
	circle(dst, x, y, r, fill, edge)
	g.label(dst, fmt.Sprintf("B%d", n), float64(x), float64(y)-8, 13, ink, true, true)
}
func (g *graphicGame) shoulder(dst *ebiten.Image, n int, x, y float32, label string) {
	fill, edge, ink := rgb(0x222e3c), rgb(0x4c5b6e), rgb(0x738094)
	if g.state.available[n] {
		ink = rgb(0xe5edf7)
	}
	if g.state.buttons[n] {
		fill, edge, ink = rgb(0x66edbd), rgb(0xa8ffde), rgb(0x0c342a)
	}
	box(dst, x, y, 144, 34, 11, fill, edge)
	g.label(dst, fmt.Sprintf("%s  ·  B%d", label, n), float64(x+72), float64(y+9), 12, ink, true, true)
}
func (g *graphicGame) stick(dst *ebiten.Image, index int, x, y float32, n int) {
	circle(dst, x, y, 57, rgb(0x171e29), rgb(0x4b586c))
	vector.StrokeLine(dst, x-45, y, x+45, y, 1, rgb(0x303e50), true)
	vector.StrokeLine(dst, x, y-45, x, y+45, 1, rgb(0x303e50), true)
	dx, dy := float32(g.state.stick[index]*23), float32(g.state.stick[index+1]*23)
	vector.StrokeLine(dst, x, y, x+dx, y+dy, 3, rgb(0x66edbd), true)
	g.button(dst, n, x+dx, y+dy, 34)
	axes := "Axes indisponibles"
	if g.state.axes[index] && g.state.axes[index+1] {
		axes = fmt.Sprintf("X %+.2f   Y %+.2f", g.state.stick[index], g.state.stick[index+1])
	}
	g.label(dst, axes, float64(x), float64(y+68), 11, rgb(0xa4b4c8), true, true)
}
func (g *graphicGame) Draw(dst *ebiten.Image) {
	dst.Fill(rgb(0x10151e))
	muted, white, green := rgb(0x94a4ba), rgb(0xeaf0f8), rgb(0x66edbd)
	g.label(dst, "LOGITECH", 44, 28, 11, green, true, false)
	g.label(dst, "Dual Action", 44, 47, 30, white, true, false)
	g.label(dst, "Chaque geste, en direct.", 45, 88, 13, muted, false, false)
	status := "Manette absente · branchez le câble USB"
	led := rgb(0xf2bb6c)
	switch g.state.status {
	case 1:
		status = "Connectée · USB 046d:c216"
		led = green
	case 2:
		status = "DÉMONSTRATION · valeurs simulées"
		led = rgb(0x9ba7ff)
	case -2:
		status = "Accès HID refusé · lancer depuis Terminal"
		led = rgb(0xff8e8e)
	case -1:
		status = "Lecture HID indisponible"
		led = rgb(0xff8e8e)
	}
	circle(dst, 600, 64, 4, led, led)
	g.label(dst, status, 614, 55, 12, led, false, false)
	box(dst, 28, 128, 904, 499, 24, rgb(0x141d2a), rgb(0x263345))
	vector.StrokeLine(dst, 480, 220, 480, 158, 5, rgb(0x566376), true)
	g.shoulder(dst, 7, 171, 160, "L2")
	g.shoulder(dst, 8, 645, 160, "R2")
	g.shoulder(dst, 5, 171, 202, "L1")
	g.shoulder(dst, 6, 645, 202, "R1")
	var body vector.Path
	body.MoveTo(270, 245)
	body.CubicTo(180, 245, 125, 276, 126, 335)
	body.CubicTo(125, 395, 153, 520, 173, 548)
	body.CubicTo(190, 585, 272, 570, 326, 528)
	body.LineTo(384, 487)
	body.CubicTo(440, 500, 520, 500, 576, 487)
	body.LineTo(634, 528)
	body.CubicTo(690, 570, 770, 585, 787, 548)
	body.CubicTo(807, 520, 835, 395, 834, 335)
	body.CubicTo(835, 276, 780, 245, 690, 245)
	body.Close()
	fillPath(dst, &body, rgb(0x303b4d))
	strokePath(dst, &body, rgb(0x59657a), 2)
	g.label(dst, "DUAL ACTION", 480, 350, 14, muted, true, true)
	box(dst, 451, 388, 58, 23, 8, rgb(0x253142), rgb(0x57647a))
	g.label(dst, "MODE", 480, 393, 10, muted, true, true)
	g.button(dst, 9, 402, 294, 19)
	g.button(dst, 10, 558, 294, 19)
	g.label(dst, "Select", 402, 320, 11, muted, true, true)
	g.label(dst, "Start", 558, 320, 11, muted, true, true)
	g.button(dst, 1, 660, 351, 25)
	g.button(dst, 2, 717, 406, 25)
	g.button(dst, 3, 774, 351, 25)
	g.button(dst, 4, 717, 296, 25)
	circle(dst, 242, 351, 58, rgb(0x1a2330), rgb(0x566176))
	dirs := graphicDirections(g.state.hat)
	pads := [4][4]float32{{229, 304, 26, 34}, {255, 338, 34, 26}, {229, 364, 26, 34}, {195, 338, 34, 26}}
	for i, r := range pads {
		fill, ink := rgb(0x293547), muted
		if dirs[i] {
			fill, ink = green, rgb(0x12382e)
		}
		box(dst, r[0], r[1], r[2], r[3], 4, fill, rgb(0x566176))
		// Direction arrows are paths, avoiding font-specific symbol coverage.
		cx, cy := r[0]+r[2]/2, r[1]+r[3]/2
		points := [4][6]float32{{-4, 3, 0, -4, 4, 3}, {-3, -4, 4, 0, -3, 4}, {-4, -3, 0, 4, 4, -3}, {3, -4, -4, 0, 3, 4}}
		q := points[i]
		var p vector.Path
		p.MoveTo(cx+q[0], cy+q[1])
		p.LineTo(cx+q[2], cy+q[3])
		p.LineTo(cx+q[4], cy+q[5])
		p.Close()
		fillPath(dst, &p, ink)
	}
	box(dst, 229, 338, 26, 26, 2, rgb(0x293547), rgb(0x293547))
	g.stick(dst, 0, 357, 463, 11)
	g.stick(dst, 2, 603, 463, 12)
	circle(dst, 49, 659, 5, green, green)
	g.label(dst, "Appuyé", 62, 650, 12, white, false, false)
	circle(dst, 161, 659, 5, rgb(0x303b4d), muted)
	g.label(dst, "Relâché", 174, 650, 12, white, false, false)
	g.label(dst, "Sticks : X/Y et Z/Rz · la croix suit le Hat HID", 390, 650, 12, muted, false, false)
	g.label(dst, "MODE peut permuter la croix et le stick gauche ; son état interne n’est pas transmis.", 44, 687, 11, muted, false, false)
	if g.preview != "" && !g.exported {
		img := image.NewRGBA(image.Rect(0, 0, 960, 720))
		dst.ReadPixels(img.Pix)
		f, err := os.Create(g.preview)
		if err == nil {
			err = png.Encode(f, img)
			if closeErr := f.Close(); err == nil {
				err = closeErr
			}
		}
		g.exportErr = err
		g.exported = true
	}
}

func runGraphic() error {
	demo := flag.Bool("demo", false, "animer une démonstration sans lire la manette")
	preview := flag.String("preview", "", "exporter une image PNG de la fenêtre puis quitter")
	flag.Parse()
	regular, err := text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF))
	if err != nil {
		return err
	}
	bold, err := text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF))
	if err != nil {
		return err
	}
	g := &graphicGame{demo: *demo, preview: *preview, regular: regular, bold: bold, state: graphicState{hat: -1}}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if !*demo {
		var done <-chan struct{}
		g.states, done = graphicReader(ctx)
		defer func() { cancel(); <-done }()
	}
	ebiten.SetWindowTitle("Logitech Dual Action — moniteur")
	ebiten.SetWindowSize(960, 720)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowSizeLimits(800, 600, -1, -1)
	ebiten.SetScreenClearedEveryFrame(true)
	ebiten.SetRunnableOnUnfocused(true)
	return ebiten.RunGame(g)
}
func main() {
	if err := runGraphic(); err != nil {
		fmt.Fprintln(os.Stderr, "Erreur :", err)
		os.Exit(1)
	}
}
