package piano

import (
	"image"
	"math"
	"sync"
)

type mode byte

const (
	Off mode = iota
	On
)

var octave = [12]func(int, image.Point, image.Point) (key, bool){
	newWhite,
	newBlack,
	newWhite,
	newBlack,
	newWhite,
	newWhite,
	newBlack,
	newWhite,
	newBlack,
	newWhite,
	newBlack,
	newWhite,
}

type key interface {
	On(int, uint64, byte)
	Off(uint64)
	Draw(*Piano) int
	Play(a []int16, rate uint64)
}

type note struct {
	b, e uint64
	y    int
	r    byte
}

type keyGeneric struct {
	sync.Mutex
	mode
	volume    byte
	wave      float64
	mix       int
	pinch     float64
	frequency float64
	rectangle image.Rectangle
	trace     []note
	size      int
}

func (k *keyGeneric) Sample(r uint64) int16 {
	t := math.Pi * k.frequency * k.wave / float64(r/2)
	f := math.Sin(1*t) * math.Exp(-.001*t) / 2
	f += math.Sin(2*t) * math.Exp(-.002*t) / 4
	f += math.Sin(4*t) * math.Exp(-.003*t) / 8
	f += math.Sin(8*t) * math.Exp(-.004*t) / 16
	f += f * f * f
	f *= .3
	k.wave++
	return int16(f * float64(k.volume) * math.MaxInt16 / 127)
}

func (k *keyGeneric) Play(a []int16, r uint64) {
	switch k.mode {
	case Off:
		if k.pinch < 0.005 {
			break
		}
		fallthrough
	case On:
		_ = a[1]
		z := k.Sample(r)
		a[k.mix%2] /= 2
		a[k.mix%2] += z / 2
	}
}

func (k *keyGeneric) On(i int, t uint64, n byte) {
	k.Lock()
	defer k.Unlock()
	if k.mode == On {
		return
	}
	k.mode = On
	k.volume = n
	k.wave = 0
	k.mix = i
	k.pinch = 0
	k.trace = append(k.trace, note{
		b: t,
		e: 0,
		y: 0,
		r: n,
	})
	k.size++
}

func (k *keyGeneric) Off(t uint64) {
	k.Lock()
	defer k.Unlock()
	if k.mode == Off {
		return
	}
	k.mode = Off
	k.trace[k.size-1].e = t
}

func (k *keyGeneric) Y() float64 {
	return float64(k.rectangle.Min.Y) + k.pinch*k.H()/3
}

func (k *keyGeneric) X() float64 {
	return float64(k.rectangle.Min.X)
}

func (k *keyGeneric) W() float64 {
	return float64(k.rectangle.Max.X)
}

func (k *keyGeneric) H() float64 {
	return float64(k.rectangle.Max.Y)
}

func (k *keyGeneric) Pinch(p *Piano) {
	k.Lock()
	defer k.Unlock()
	switch k.mode {
	case On:
		if k.pinch < 1 {
			k.pinch += .25
		}
	case Off:
		k.pinch /= 4
	}
	p.SetRGBA(0, 0, 0, 1)
	i := k.size
	for i > 0 {
		i--
		r := k.trace[i].r >> 4
		y := k.trace[i].y + k.rectangle.Min.Y - k.rectangle.Max.Y
		if k.trace[i].e != 0 && y < 0 {
			if k.trace[k.size-1].e != 0 {
				k.size--
				k.trace[i] = k.trace[k.size]
				k.trace = k.trace[:k.size]
			}
		} else {
			p.DrawCircle(k.X()+k.W()/2, float64(y), float64(r))
			p.Stroke()
			k.trace[i].y -= 2 * int(r)
			for j := i + 1; j < k.size; j++ {
				if 0x7FFFFFFF&(k.trace[i].y-k.trace[j].y) < 2*int(r) {
					if k.trace[i].r > k.trace[j].r {
						k.trace[j].y -= k.rectangle.Min.Y
						k.trace[i].r += k.trace[j].r / 16
					} else {
						k.trace[i].y -= k.rectangle.Min.Y
						k.trace[j].r += k.trace[i].r / 16
					}
				}
			}
		}
	}
}

type keyWhite struct {
	keyGeneric
}

func (k *keyWhite) Draw(p *Piano) int {
	k.Pinch(p)
	p.SetRGBA(0, 0, 0, 1)
	p.SetLineWidth(1)
	p.DrawRectangle(k.X(), k.Y(), k.W(), k.H())
	p.Stroke()
	return k.size
}

func newWhite(f int, h, s image.Point) (key, bool) {
	return &keyWhite{
		keyGeneric{
			frequency: 440 * math.Exp2(float64(f-48)/12),
			rectangle: image.Rectangle{
				Min: h.Div(1),
				Max: s,
			},
		},
	}, true
}

type keyBlack struct {
	keyGeneric
}

func (k *keyBlack) Draw(p *Piano) int {
	k.Pinch(p)
	p.SetRGBA(0, 0, 0, 1)
	p.SetLineWidth(1)
	p.DrawRectangle(k.X(), k.Y(), k.W(), k.H())
	p.Fill()
	return k.size
}

func newBlack(f int, h, s image.Point) (key, bool) {
	s = s.Div(5).Mul(4)
	s.X /= 4
	s.X *= 4
	s.Y /= 8
	s.Y *= 8
	return &keyBlack{
		keyGeneric{
			frequency: 440 * math.Exp2(float64(f-48)/12),
			rectangle: image.Rectangle{
				Min: h.Sub(s.Div(2)),
				Max: s,
			},
		},
	}, false
}
