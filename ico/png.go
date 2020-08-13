package main

import (
	"fmt"
	"image"
	"image/png"
	"log"
	"os"
)

func main() {
	var err error
	var f *os.File
	f, err = os.Open("ico/piano.png")
	if err != nil {
		log.Fatal(err)
	}
	var p image.Image
	p, err = png.Decode(f)
	if err != nil {
		log.Fatal(err)
	}
	b := p.Bounds()
	q := image.NewRGBA(b)
	for x := 0; x < b.Dx(); x++ {
		for y := 0; y < b.Dy(); y++ {
			q.Set(x, y, p.At(x, y))
		}
	}
	fmt.Printf("\n%#v\n", q)
}
