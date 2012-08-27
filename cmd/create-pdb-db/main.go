// Example bag-of-words shows how to compute a bag-of-words vector given a
// fragment library and a PDB file.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"runtime/pprof"

	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/pdb"
)

var (
	flagCpuProfile = ""
	flagGoMaxProcs = runtime.NumCPU()
	flagOverwrite  = false
	flagQuiet      = false
)

func writer(db *bowdb.DB,
	pchan chan progressJob, pool pool) (chan struct{}, error) {

	quit := make(chan struct{}, 0)
	go func() {
		for result := range pool.results {
			pchan <- progressJob{"", nil}
			if err := db.Write(result.entry, result.bow); err != nil {
				fatalf("%s\n", err)
			}
		}
		quit <- struct{}{}
	}()
	return quit, nil
}

type progressJob struct {
	pdbFile string
	err     error
}

func progress(total int) chan progressJob {
	pchan := make(chan progressJob, 15)
	successCnt, errCnt := 0, 0
	go func() {
		for pjob := range pchan {
			if pjob.err != nil {
				errorf("\rCould not parse PDB file '%s' because: %s\n",
					pjob.pdbFile, pjob.err)
				errCnt++
			} else {
				successCnt++
			}
			verbosef("\r%d of %d PDB files processed. "+
				"(%0.2f%% done with %d errors.)",
				successCnt+errCnt, total,
				100.0*(float64(successCnt+errCnt)/float64(total)),
				errCnt)
		}
	}()
	return pchan
}

func main() {
	if flag.NArg() < 3 {
		usage()
	}
	dbPath := flag.Arg(0)
	libPath := flag.Arg(1)
	pdbFiles := flag.Args()[2:]

	if flagOverwrite {
		if err := os.RemoveAll(dbPath); err != nil {
			fatalf("Could not remove '%s' directory because: %s.", dbPath, err)
		}
	}

	lib, err := fragbag.NewLibrary(libPath)
	if err != nil {
		fatalf("Could not open fragment library '%s': %s\n", lib, err)
	}
	verbosef("Using library %s.\n", lib)

	db, err := bowdb.Create(lib, dbPath)
	if err != nil {
		fatalf("%s\n", err)
	}

	pool := newBowWorkers(lib, max(1, flagGoMaxProcs))
	progressChan := progress(len(pdbFiles))
	doneWriting, err := writer(db, progressChan, pool)
	if err != nil {
		fatalf("Could not create database file: %s\n", err)
	}

	if len(flagCpuProfile) > 0 {
		f, err := os.Create(flagCpuProfile)
		if err != nil {
			fatalf("%s\n", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}
	for _, pdbFile := range pdbFiles {
		entry, err := pdb.New(pdbFile)
		if err != nil {
			progressChan <- progressJob{pdbFile, err}
			continue
		}
		pool.enqueue(entry)
	}

	pool.done()
	<-doneWriting
	close(progressChan)
	verbosef("\n")

	if err := db.WriteClose(); err != nil {
		fatalf("There was an error closing the database: %s\n", err)
	}
}

func init() {
	flag.StringVar(&flagCpuProfile, "cpuprofile", flagCpuProfile,
		"When set, a CPU profile will be written to the file provided.")
	flag.IntVar(&flagGoMaxProcs, "p", flagGoMaxProcs,
		"The maximum number of CPUs that can be executing simultaneously.")
	flag.BoolVar(&flagOverwrite, "overwrite", flagOverwrite,
		"When set, any existing database will be completely overwritten.")
	flag.BoolVar(&flagQuiet, "quiet", flagQuiet,
		"When set, no progress bar will be shown.\n"+
			"\tErrors will still be printed to stderr.")
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

func verbosef(format string, v ...interface{}) {
	if flagQuiet {
		return
	}
	fmt.Fprintf(os.Stdout, format, v...)
}

func errorf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
}

func fatalf(format string, v ...interface{}) {
	errorf(format, v...)
	os.Exit(1)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
