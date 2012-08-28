package main

import (
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"runtime/pprof"

	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/pdb"
)

var (
	flagCpuProfile = ""
	flagGoMaxProcs = runtime.NumCPU()
	flagQuiet      = false
)

func main() {
	if flag.NArg() < 2 {
		usage()
	}
	dbPath := flag.Arg(0)
	pdbFiles := flag.Args()[1:]

	db, err := bowdb.Open(dbPath, bowdb.SearchFull)
	if err != nil {
		fatalf("%s\n", err)
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
			errorf("Could not parse PDB file '%s' because: %s", pdbFile, err)
			continue
		}

		results := db.SearchPDB(bowdb.DefaultSearchOptions, entry)
		for _, result := range results.Results {
			fmt.Println(result)
		}
	}

	if err := db.ReadClose(); err != nil {
		fatalf("There was an error closing the database: %s\n", err)
	}
}

func init() {
	flag.StringVar(&flagCpuProfile, "cpuprofile", flagCpuProfile,
		"When set, a CPU profile will be written to the file provided.")
	flag.IntVar(&flagGoMaxProcs, "p", flagGoMaxProcs,
		"The maximum number of CPUs that can be executing simultaneously.")
	flag.BoolVar(&flagQuiet, "quiet", flagQuiet,
		"When set, no progress bar will be shown.\n"+
			"\tErrors will still be printed to stderr.")
	flag.Usage = usage
	flag.Parse()

	runtime.GOMAXPROCS(flagGoMaxProcs)
}

func usage() {
	errorf("Usage: %s database-path query-pdb-file [query-pdb-file ...]\n",
		path.Base(os.Args[0]))
	flag.PrintDefaults()
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
