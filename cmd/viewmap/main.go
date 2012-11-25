package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/bcbgo/hhfrag"
)

func init() {
	log.SetFlags(0)

	flag.Usage = usage
	flag.Parse()
}

func usage() {
	log.Printf("Usage: viewmap fmap-file\n")
	flag.PrintDefaults()
}

func assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}

func main() {
	if flag.NArg() != 1 {
		log.Printf("One input file required.")
		usage()
	}
	ffmap := flag.Arg(0)

	f, err := os.Open(ffmap)
	assert(err)
	r := gob.NewDecoder(f)

	var fmap hhfrag.FragmentMap
	assert(r.Decode(&fmap))

	for _, frags := range fmap {
		fmt.Printf("\nSEGMENT: %d %d (%d)\n",
			frags.Start, frags.End, len(frags.Frags))
		frags.Write(os.Stdout)
	}
}
