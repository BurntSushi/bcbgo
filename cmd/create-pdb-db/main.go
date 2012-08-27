// Example bag-of-words shows how to compute a bag-of-words vector given a
// fragment library and a PDB file.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"

	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/pdb"
	"github.com/BurntSushi/bcbgo/pdbbow"
)

var (
	flagGoMaxProcs = runtime.NumCPU()
)

func writer(db *pdbbow.DB, pool pool) (chan struct{}, error) {
	quit := make(chan struct{}, 0)
	go func() {
		for result := range pool.results {
			if err := db.Write(result.entry, result.bow); err != nil {
				fatalf("%s\n", err)
			}
		}
		quit <- struct{}{}
	}()

	return quit, nil
}

func main() {
	if flag.NArg() < 3 {
		usage()
	}

	lib, err := fragbag.NewLibrary(flag.Arg(1))
	if err != nil {
		fatalf("Could not open fragment library '%s': %s\n", lib, err)
	}
	fmt.Fprintf(os.Stderr, "Using library %s.\n", lib)

	db, err := pdbbow.Create(lib, flag.Arg(0))
	if err != nil {
		fatalf("%s\n", err)
	}

	pool := newBowWorkers(lib, max(1, flagGoMaxProcs))
	doneWriting, err := writer(db, pool)
	if err != nil {
		fatalf("Could not create database file: %s\n", err)
	}

	pdbFiles := flag.Args()[2:]
	total := len(pdbFiles)
	for i, pdbFile := range pdbFiles {
		entry, err := pdb.New(pdbFile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Could not get a BOW vector for PDB file "+
				"'%s': %s.\n", pdbFile, err)
			return
		}
		pool.enqueue(entry)
		fmt.Fprintf(os.Stderr,
			"\r%d of %d PDB files processed. (%0.2f%% done.)",
			i+1, total, 100.0*(float64(i+1)/float64(total)))
	}

	fmt.Fprint(os.Stderr, "\nComputing final BOW vectors...\n")
	pool.done()
	<-doneWriting

	if err := db.WriteClose(); err != nil {
		fatalf("There was an error closing the database: %s\n", err)
	}
}

func init() {
	flag.IntVar(&flagGoMaxProcs, "p", flagGoMaxProcs,
		"The maximum number of CPUs that can be executing simultaneously.")
	flag.Usage = usage
	flag.Parse()

	runtime.GOMAXPROCS(flagGoMaxProcs)
}

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s database-path frag-lib-directory pdb-file [pdb-file ...]\n",
		path.Base(os.Args[0]))
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr,
		"\nex. './%s data/fraglibs/centers400_11 data/samples/*.pdb'\n",
		path.Base(os.Args[0]))
	os.Exit(1)
}

func fatalf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
	os.Exit(1)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
