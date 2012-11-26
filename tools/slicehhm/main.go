package main

import (
	"flag"
	"log"
	"os"
	"strconv"

	"github.com/BurntSushi/bcbgo/io/hhm"
)

func init() {
	log.SetFlags(0)

	flag.Usage = usage
	flag.Parse()
}

func usage() {
	log.Println("Usage: slicehhm hhm-file start end")
	os.Exit(1)
}

func main() {
	if flag.NArg() != 3 {
		usage()
	}
	hhmFile := flag.Arg(0)

	start, err := strconv.Atoi(flag.Arg(1))
	assert(err)

	end, err := strconv.Atoi(flag.Arg(2))
	assert(err)

	fhhm, err := os.Open(hhmFile)
	assert(err)

	qhhm, err := hhm.Read(fhhm)
	assert(err)

	hhm.Write(os.Stdout, qhhm.Slice(start, end))
}

func assert(err error) {
	if err != nil {
		log.Fatalln(err)
	}
}
