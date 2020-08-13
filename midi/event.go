package midi

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

type Event struct {
	Time  uint64
	Delta uint64
	Type  byte
	Chan  byte
	Note  byte
	Value byte
	Data  []byte
}

const (
	NoteOff byte = 0x8 | iota
	NoteOn
	Polyphonic
	Control
	Program
	Channel
	PitchBend
	Meta
)

const (
	Sequence = 0x00 + iota
	Text
	Copyright
	Name
	Instrument
	Lyric
	Marker
	CuePoint
	ProgramName
	DeviceName
	ChannelPrefix = 0x20
	PortNumber    = 0x21
	EndOfTrack    = 0x2F
	Tempo         = 0x51
	SMPTEOffset   = 0x54
	TimeSignature = 0x58
	KeySignature  = 0x59
	Sequencer     = 0x7F
)

var (
	TypeName = map[byte]string{
		0x8: "NoteOff",
		0x9: "NoteOn",
		0xA: "Polyphonic",
		0xB: "Control",
		0xC: "Program",
		0xD: "Channel",
		0xE: "PitchBend",
		0xF: "#",
	}
	EventName = map[byte]string{
		0x00: "Sequence",
		0x01: "Text",
		0x02: "Copyright",
		0x03: "Name",
		0x04: "Instrument",
		0x05: "Lyric",
		0x06: "Marker",
		0x07: "CuePoint",
		0x08: "ProgramName",
		0x09: "DeviceName",
		0x20: "ChannelPrefix",
		0x21: "PortNumber",
		0x2F: "EndOfTrack",
		0x51: "Tempo",
		0x54: "SMPTEOffset",
		0x58: "TimeSignature",
		0x59: "KeySignature",
		0x7F: "Sequencer",
	}
	ControlName = map[byte]string{
		0:   "Bank Select",
		1:   "Modulation Wheel",
		2:   "Breath Controller",
		4:   "Foot Controller",
		5:   "Portamento Time",
		6:   "Data Entry",
		7:   "Channel Volume",
		8:   "Balance",
		10:  "Pan",
		11:  "Expression Controller",
		12:  "Effect Control 1",
		13:  "Effect Control 2",
		16:  "Gen Purpose Controller 1",
		17:  "Gen Purpose Controller 2",
		18:  "Gen Purpose Controller 3",
		19:  "Gen Purpose Controller 4",
		32:  "Bank Select",
		33:  "Modulation Wheel",
		34:  "Breath Controller",
		36:  "Foot Controller",
		37:  "Portamento Time",
		38:  "Data Entry",
		39:  "Channel Volume",
		40:  "Balance",
		42:  "Pan",
		43:  "Expression Controller",
		44:  "Effect Control 1",
		45:  "Effect Control 2",
		48:  "General Purpose Controller 1",
		49:  "General Purpose Controller 2",
		50:  "General Purpose Controller 3",
		51:  "General Purpose Controller 4",
		64:  "Sustain On/Off",
		65:  "Portamento On/Off",
		66:  "Sostenuto On/Off",
		67:  "Soft Pedal On/Off",
		68:  "Legato On/Off",
		69:  "Hold 2 On/Off",
		70:  "Sound Controller 1",
		71:  "Sound Controller 2",
		72:  "Sound Controller 3",
		73:  "Sound Controller 4",
		74:  "Sound Controller 5",
		75:  "Sound Controller 6",
		76:  "Sound Controller 7",
		77:  "Sound Controller 8",
		78:  "Sound Controller 9",
		79:  "Sound Controller 10",
		80:  "General Purpose Controller 5",
		81:  "General Purpose Controller 6",
		82:  "General Purpose Controller 7",
		83:  "General Purpose Controller 8",
		84:  "Portamento Control",
		88:  "High Resolution Velocity Prefix",
		91:  "Effects 1 Depth",
		92:  "Effects 2 Depth",
		93:  "Effects 3 Depth",
		94:  "Effects 4 Depth",
		95:  "Effects 5 Depth",
		96:  "Data Increment",
		97:  "Data Decrement",
		98:  "Non Registered Parameter Number 1",
		99:  "Non Registered Parameter Number 2",
		100: "Registered Parameter Number 1",
		101: "Registered Parameter Number 2",
		120: "All Sound Off",
		121: "Reset All Controllers",
		122: "Local Control On/Off",
		123: "All Notes Off",
		124: "Omni Mode Off",
		125: "Omni Mode On",
		126: "Mono Mode On",
		127: "Poly Mode On",
	}
	NoteName = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
)

func (e *Event) String() string {
	s := fmt.Sprintf("%d +%d 0x%X %02d %s", e.Time, e.Delta, e.Type, e.Chan, e.TypeName())
	switch e.Type {
	case NoteOn, NoteOff:
		s += fmt.Sprintf(" %s", e.NoteName())
		if e.Value > 0 {
			s += fmt.Sprintf(":%v", e.Value)
		}
	case Polyphonic:
	case Control:
		s += fmt.Sprintf(" %v %v", e.Control(), e.Value)
	case Program:
		s += fmt.Sprintf(" %v", e.Note)
	case Channel:
	case PitchBend:
	case Meta:
		switch e.Chan {
		case 0xF:
			s += fmt.Sprintf(" %s", e.EventName())
			switch e.Note {
			case Sequence:
				s += fmt.Sprintf(" %d", e.Sequence())
			case Text, Copyright, Name, Instrument, Lyric, Marker, CuePoint, ProgramName, DeviceName:
				s += fmt.Sprintf(" <%s>", e.Text())
			case ChannelPrefix:
				s += fmt.Sprintf(" %v", e.Channel())
			case PortNumber:
				s += fmt.Sprintf(" %v", e.Port())
			case Tempo:
				s += fmt.Sprintf(" %v", e.Tempo())
			case SMPTEOffset, TimeSignature, KeySignature, Sequencer:
				s += fmt.Sprintf(" %v", e.Bytes())
			case EndOfTrack:
			}
		}
	}
	return "{" + s + "}"
}

func (e *Event) Fix() {
	if e.Type == NoteOn && e.Value == 0 {
		e.Type = NoteOff
	}
}

func (e *Event) TypeName() string {
	if v, ok := TypeName[e.Type]; ok {
		return v
	}
	return fmt.Sprintf("Type0x%x", e.Note)
}

func (e *Event) NoteName() string {
	key := NoteName[e.Key()]
	return key + fmt.Sprint(e.Octave())
}

func (e *Event) Octave() byte {
	return e.Note/12 - 2
}

func (e *Event) Key() byte {
	return e.Note % 12
}

func (e *Event) EventName() string {
	if v, ok := EventName[e.Note]; ok {
		return v
	}
	return fmt.Sprintf("Event0x%x", e.Note)
}

func (e *Event) Sequence() uint16 {
	return binary.BigEndian.Uint16(e.Data)
}

func (e *Event) Text() []byte {
	return bytes.TrimSpace(e.Data)
}

func (e *Event) Tempo() uint32 {
	return 60000000 / binary.BigEndian.Uint32(append([]byte{0}, e.Data...))
}

func (e *Event) Channel() byte {
	return e.Data[0] & 0xF
}

func (e *Event) Port() byte {
	return e.Data[0] & 0x80
}

func (e *Event) Bytes() []byte {
	return e.Data
}

func (e *Event) Control() string {
	if v, ok := ControlName[e.Note]; ok {
		return v
	}
	return fmt.Sprintf("Control0x%x", e.Note)
}
