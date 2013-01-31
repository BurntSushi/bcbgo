package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

var (
	flagCpu = runtime.NumCPU()
)

func init() {
	log.SetFlags(0)

	flag.IntVar(&flagCpu, "cpu", flagCpu,
		"The max number of CPUs to use.")

	flag.Usage = usage
	flag.Parse()

	runtime.GOMAXPROCS(flagCpu)
}

func usage() {
	log.Printf("Usage: bow [flags] frag-lib-dir chain pdb-file out-bow\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	if flag.NArg() != 4 {
		usage()
	}
	libPath := flag.Arg(0)
	chain := flag.Arg(1)
	pdbFile := flag.Arg(2)
	bowOut := flag.Arg(3)

	lib, err := fragbag.NewLibrary(libPath)
	if err != nil {
		fatalf("Could not open fragment library '%s': %s\n", lib, err)
	}

	entry, err := pdb.ReadPDB(pdbFile)
	if err != nil {
		fatalf("Could not open PDB file '%s': %s\n", pdbFile, err)
	}

	thechain := entry.Chain(chain[0])
	if thechain == nil || !thechain.IsProtein() {
		fatalf("Could not find chain with identifier '%c'.", chain[0])
	}

	bow := lib.NewBowChain(thechain)

	out, err := os.Create(bowOut)
	if err != nil {
		fatalf("Could not create file '%s': %s", bowOut, err)
	}

	w := gob.NewEncoder(out)
	if err := w.Encode(bow); err != nil {
		fatalf("Could not GOB encode BOW: %s", err)
	}
}

func errorf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

func fatalf(format string, v ...interface{}) {
	errorf(format, v...)
	os.Exit(1)
}
