package piano

import (
	"encoding/binary"
	"image"
	"io"
	"log"
	"time"

	"github.com/fogleman/gg"
	"github.com/hajimehoshi/ebiten"
	"github.com/hajimehoshi/ebiten/audio"

	"github.com/pshvedko/piano/midi"
)

type draw struct {
	*gg.Context
}

type board [9][12]key

func (k *board) Update(i int, e *midi.Event) *midi.Event {
	switch e.Type {
	case midi.NoteOn:
		k[1+e.Octave()][e.Key()].On(i, e.Time, e.Value)
	case midi.NoteOff:
		k[1+e.Octave()][e.Key()].Off(e.Time)
	}
	return e
}

type progress struct {
	x, y, w float64
}

type button struct {
	state bool
	click int
	time  time.Time
}

type Piano struct {
	draw
	rgba   *image.RGBA
	midi   *midi.Context
	play   *audio.Player
	iter   []uint
	time   uint64
	rate   uint64
	tick   uint64
	next   uint64
	key    board
	bar    progress
	button map[ebiten.Key]bool
	mouse  map[ebiten.MouseButton]button
	flow   chan *midi.Event
	log    bool
}

func (p *Piano) Read(b []byte) (n int, err error) {
	_ = b[3]
	for i, t := range p.midi.Tracks {
		for p.iter[i] < t.NumEvents && p.next >= t.Events[p.iter[i]].Time*p.rate/p.tick {
			p.flow <- p.key.Update(i, t.Events[p.iter[i]])
			p.iter[i]++
		}
	}
	p.next++
	p.time = p.next * p.tick / p.rate
	var a [2]int16
	for o := range p.key {
		for _, k := range p.key[o] {
			if k != nil {
				k.Play(a[:], p.rate)
			}
		}
	}
	binary.LittleEndian.PutUint16(b[0:2], uint16(a[0]))
	binary.LittleEndian.PutUint16(b[2:4], uint16(a[1]))
	return 4, nil
}

func (p *Piano) KeyPressed(k ebiten.Key) bool {
	b := p.button[k]
	p.button[k] = ebiten.IsKeyPressed(k)
	return b && !p.button[k]
}

func (p *Piano) MouseClicked(m ebiten.MouseButton, t time.Time) int {
	o := p.mouse[m]
	b := o.state
	o.state = ebiten.IsMouseButtonPressed(m)
	if b && !o.state {
		o.click++
		o.time = t
	} else if t.Sub(o.time) > 250*time.Millisecond {
		o.click = 0
		o.time = t
	}
	p.mouse[m] = o
	return o.click
}

func (p *Piano) Update(e *ebiten.Image) error {
	switch ebiten.IsFullscreen() {
	case true:
		if p.KeyPressed(ebiten.KeyEscape) || p.KeyPressed(ebiten.KeyF) {
			ebiten.SetFullscreen(false)
		}
	case false:
		if p.MouseClicked(ebiten.MouseButtonLeft, time.Now()) == 2 || p.KeyPressed(ebiten.KeyF) {
			ebiten.SetFullscreen(true)
		}
	}
	if p.KeyPressed(ebiten.KeySpace) {
		if p.play.IsPlaying() {
			_ = p.play.Pause()
		} else {
			_ = p.play.Play()
		}
	}
	for {
		select {
		case e := <-p.flow:
			if p.log {
				log.Println(e)
			}
			continue
		default:
		}
		break
	}
	p.SetRGBA(1, 1, 1, 1)
	p.Clear()
	n := 0
	for o := range p.key {
		for _, k := range p.key[o] {
			if k != nil {
				n += k.Draw(p)
			}
		}
	}
	if p.time > p.midi.Time && n == 0 {
		return io.EOF
	}
	p.SetRGBA(0, 0, 0, 1)
	p.DrawPoint(p.bar.x+float64(p.time)/float64(p.midi.Time)*p.bar.w, p.bar.y, 3)
	p.Fill()
	return e.ReplacePixels(p.rgba.Pix)
}

func (p *Piano) Run(w, h, r int, m *midi.Context, v bool) (err error) {
	p.rgba = image.NewRGBA(image.Rectangle{
		Max: image.Point{
			X: w,
			Y: h,
		},
	})
	p.Context = gg.NewContextForRGBA(p.rgba)
	var a *audio.Context
	a, err = audio.NewContext(r)
	if err != nil {
		return
	}
	p.log = v
	p.midi = m
	p.rate = uint64(r)
	p.tick = 4 * uint64(p.midi.TicksPerQuarterNote)
	p.iter = make([]uint, m.NumTracks)
	hook := image.Point{X: w % 52 / 2, Y: h / 20 * 18}
	size := image.Point{X: w / 52, Y: h / 20}
	p.bar.x = float64(hook.X)
	p.bar.y = float64(h - hook.X)
	p.bar.w = float64(w - hook.X*2)
	for i := 0; i < 88; i++ {
		o := (i + 9) / 12
		n := (i + 9) % 12
		var w bool
		p.key[o][n], w = octave[n](i, hook, size)
		if w {
			hook.X += size.X
		}
	}
	p.flow = make(chan *midi.Event, 1024)
	p.mouse = map[ebiten.MouseButton]button{}
	p.button = map[ebiten.Key]bool{}
	p.play, _ = audio.NewPlayer(a, p)
	_ = p.play.Play()
	defer func() {
		_ = p.play.Close()
	}()
	ebiten.SetWindowIcon([]image.Image{ico})
	ebiten.SetWindowTitle("Piano")
	ebiten.SetRunnableOnUnfocused(true)
	ebiten.SetWindowSize(p.Width(), p.Height())
	err = ebiten.RunGame(p)
	if err != nil && err == io.EOF {
		err = nil
	}
	return
}

func (p *Piano) Layout(int, int) (int, int) {
	return p.Width(), p.Height()
}

func (p *Piano) Close() error {
	close(p.flow)
	return nil
}
