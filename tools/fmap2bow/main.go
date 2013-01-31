package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"os"
	"runtime"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/hhfrag"
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
	log.Printf("Usage: fmap2bow [flags] frag-lib-dir fmap-file out-bow\n")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	if flag.NArg() != 3 {
		usage()
	}
	libPath := flag.Arg(0)
	fmap := loadFmap(flag.Arg(1))
	bowOut := flag.Arg(2)

	lib, err := fragbag.NewLibrary(libPath)
	if err != nil {
		fatalf("Could not open fragment library '%s': %s\n", lib, err)
	}

	bow := fmap.BOW(lib)

	out, err := os.Create(bowOut)
	if err != nil {
		fatalf("Could not create file '%s': %s", bowOut, err)
	}

	w := gob.NewEncoder(out)
	if err := w.Encode(bow); err != nil {
		fatalf("Could not GOB encode BOW: %s", err)
	}
}

func loadFmap(fpath string) hhfrag.FragmentMap {
	f, err := os.Open(fpath)
	if err != nil {
		fatalf("Could not open fmap file '%s': %s", fpath, err)
	}

	var fmap hhfrag.FragmentMap
	r := gob.NewDecoder(f)
	if err := r.Decode(&fmap); err != nil {
		fatalf("Could not GOB decode fmap file '%s': %s", fpath, err)
	}
	return fmap
}

func errorf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

func fatalf(format string, v ...interface{}) {
	errorf(format, v...)
	os.Exit(1)
}
