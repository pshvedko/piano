package midi

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Context struct {
	Format              uint16
	NumTracks           uint16
	TicksPerQuarterNote uint16
	Tracks              []*Track
	Last                *Track
	Time                uint64
}

type Reader struct {
	io.Reader
}

func (r Reader) ReadByte() (byte, error) {
	var b byte
	return b, binary.Read(r, binary.BigEndian, &b)
}

func (r Reader) ReadVarUint64() (u uint64, err error) {
	var b byte
	for i := 0; ; i++ {
		b, err = r.ReadByte()
		if err != nil {
			return
		}
		u |= uint64(b & 0x7F)
		if b < 0x80 {
			if i > 9 || i == 9 && b > 1 {
				return u, fmt.Errorf("overflow")
			}
			return
		}
		u <<= 7
	}
}

func (c *Context) Read(r io.ReadCloser) error {
	defer func() {
		_ = r.Close()
	}()
	return c.read(Reader{r})
}

func (c *Context) read(r Reader) (err error) {
	var header [4]byte
	err = binary.Read(r, binary.BigEndian, &header)
	if err != nil {
		return
	} else if header != [4]byte{'M', 'T', 'h', 'd'} {
		return fmt.Errorf("header not supported %v", header)
	}
	var headerSize uint32
	err = binary.Read(r, binary.BigEndian, &headerSize)
	if err != nil {
		return
	} else if headerSize != 6 {
		return fmt.Errorf("expected header size to be 6, was %d", headerSize)
	}
	err = binary.Read(r, binary.BigEndian, &c.Format)
	if err != nil {
		return
	}
	err = binary.Read(r, binary.BigEndian, &c.NumTracks)
	if err != nil {
		return
	}
	err = binary.Read(r, binary.BigEndian, &c.TicksPerQuarterNote)
	if err != nil {
		return
	}
	err = c.readTrack(r)
	if err != nil && err == io.EOF {
		err = nil
	}
	return
}

func (c *Context) readTrack(r Reader) (err error) {
	var header [4]byte
	err = binary.Read(r, binary.BigEndian, &header)
	if err != nil {
		return
	} else if header != [4]byte{'M', 'T', 'r', 'k'} {
		return fmt.Errorf("track not supported %v", header)
	}
	t := &Track{}
	err = binary.Read(r, binary.BigEndian, &t.Size)
	if err != nil {
		return
	}
	c.Tracks = append(c.Tracks, t)
	c.Last = t
	return c.readEvent(r)
}

func (c *Context) readEvent(r Reader) (err error) {
	var at uint64
	at, err = r.ReadVarUint64()
	if err != nil {
		return
	}
	var status, note byte
	status, err = r.ReadByte()
	if err != nil {
		return
	}
	if status == 0xF0 || status == 0xF7 {
		at, err = r.ReadVarUint64()
		if err != nil {
			return
		}
		_, err = r.Read(make([]byte, at))
		if err != nil {
			return
		}
		return c.readEvent(r)
	} else if status&0x80 == 0 {
		if c.Last.Last == nil {
			return fmt.Errorf("format error")
		}
		note, status = status, c.Last.Last.Type<<4|c.Last.Last.Chan
	} else {
		note, err = r.ReadByte()
		if err != nil {
			return
		}
	}
	e := &Event{Delta: at, Type: status & 0xF0 >> 4, Chan: status & 0x0F, Note: note}
	c.Last.NumEvents++
	c.Last.Events = append(c.Last.Events, e)
	c.Last.Time += at
	c.Last.Last = e
	e.Time = c.Last.Time
	switch e.Type {
	case NoteOn:
		defer e.Fix()
		fallthrough
	case NoteOff, Polyphonic, Control, PitchBend:
		e.Value, err = r.ReadByte()
		if err != nil {
			return
		}
		fallthrough
	case Program, Channel:
	case Meta:
		return c.readMeta(r, e)
	default:
		return fmt.Errorf("unknown message type %X", e.Type)
	}
	return c.readEvent(r)
}

func (c *Context) readMeta(r Reader, e *Event) (err error) {
	switch e.Chan {
	case Meta:
		var n uint64
		n, err = r.ReadVarUint64()
		if err != nil {
			return
		}
		e.Data = make([]byte, n)
		_, err = r.Read(e.Data)
		if err != nil {
			return
		}
		switch e.Note {
		case 0x00:
			if n != 2 {
				return fmt.Errorf("key signature length not 2 as expected but %d", len(e.Data))
			}
		case 0x01:
		case 0x02:
		case 0x03:
		case 0x04:
		case 0x05:
		case 0x06:
		case 0x07:
		case 0x08:
		case 0x09:
		case 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F:
		case 0x2F:
			if c.Time < e.Time {
				c.Time = e.Time
			}
			return c.readTrack(r)
		case 0x20:
			if n != 1 {
				return fmt.Errorf("channel prefix length not 1 as expected but %d", len(e.Data))
			}
		case 0x21:
			if n != 1 {
				return fmt.Errorf("port length not 1 as expected but %d", len(e.Data))
			}
		case 0x51:
			if n != 3 {
				return fmt.Errorf("tempo length not 3 as expected but %d", len(e.Data))
			}
		case 0x54:
			if n != 5 {
				return fmt.Errorf("smpte offset length not 5 as expected but %d", len(e.Data))
			}
		case 0x58:
			if n != 4 {
				return fmt.Errorf("time signature length not 4 as expected but %d", len(e.Data))
			}
		case 0x59:
			if n != 2 {
				return fmt.Errorf("key signature length not 2 as expected but %d", len(e.Data))
			}
		case 0x7F:
		default:
			return fmt.Errorf("unknown meta event type %d", e.Note)
		}
	default:
		return fmt.Errorf("unknown system event type %d", e.Chan)
	}
	return c.readEvent(r)
}
