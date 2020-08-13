package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/pshvedko/piano/midi"
	"github.com/pshvedko/piano/piano"
)

const midiFile = "http://bitmidi.com/uploads/28051.mid"

func main() {
	file := midiFile
	var v bool
	flag.BoolVar(&v, "verbose", false, "log notes during playing")
	flag.Parse()
	if flag.NArg() > 0 {
		file = flag.Arg(0)
	}
	u, err := url.Parse(file)
	if err != nil {
		log.Fatal(err)
	}
	var f io.ReadCloser
	switch u.Scheme {
	case "file", "":
		f, err = os.Open(u.Path)
		if err != nil {
			log.Fatal(err)
		}
	case "http", "https":
		c := http.Client{}
		var r *http.Response
		r, err = c.Get(file)
		if err != nil {
			log.Fatal(err)
		}
		f = r.Body
	default:
		flag.PrintDefaults()
		return
	}
	m := &midi.Context{}
	err = m.Read(f)
	if err != nil {
		log.Fatal(err)
	}
	p := &piano.Piano{}
	err = p.Run(800, 600, 44100, m, v)
	if err != nil {
		log.Fatal(err)
	}
}
