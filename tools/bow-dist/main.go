package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"math"
	"os"

	"github.com/BurntSushi/bcbgo/fragbag"
)

func init() {
	log.SetFlags(0)

	flag.Usage = usage
	flag.Parse()
}

func usage() {
	log.Printf("Usage: bow-dist [flags] bow1 bow2\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func loadBow(fpath string) fragbag.BOW {
	f, err := os.Open(fpath)
	if err != nil {
		fatalf("Could not open '%s': %s", fpath, err)
	}

	var bow fragbag.BOW
	r := gob.NewDecoder(f)
	if err := r.Decode(&bow); err != nil {
		fatalf("Could not decode BOW '%s': %s", fpath, err)
	}
	return bow
}

func main() {
	if flag.NArg() != 2 {
		usage()
	}
	bow1 := loadBow(flag.Arg(0))
	bow2 := loadBow(flag.Arg(1))

	fmt.Printf("%0.4f\n", math.Abs(bow1.Cosine(bow2)))
}

func errorf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

func fatalf(format string, v ...interface{}) {
	errorf(format, v...)
	os.Exit(1)
}
