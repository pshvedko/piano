package midi

type Track struct {
	Size      uint32
	Time      uint64
	Events    []*Event
	Last      *Event
	NumEvents uint
}
