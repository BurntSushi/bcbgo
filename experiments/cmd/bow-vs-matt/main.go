// test bow-vs-matt compares the structural neighbors detected by a search
// on BOW vectors verus the structural neighbors detected by the Matt
// structural aligner.
//
// The test only produces the output of the comparison, but does not say how
// similar they are.
//
// The comparison is that of ordering. Namely, given some protein chain A,
// find the closest structural neighbors returned by two different methods:
// structural alignment and BOW searching.
//
// This is done by first creating a BOW database from the given list of PDB
// files. (This requires that 'create-pdb-db' is in your PATH.) Then, output
// a file for every protein chain that contains a ordering of all structural
// neighbors using both search techniques.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
	"runtime/pprof"
	"strings"
	"text/tabwriter"

	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/matt"
)

type results []result

type result struct {
	entry   string
	chain   byte
	results bowdb.SearchResults
}

var (
	flagCpuProfile = ""
	flagGoMaxProcs = runtime.NumCPU()
	flagQuiet      = false
	flagInverted   = false
)

func main() {
	if flag.NArg() < 2 {
		usage()
	}
	dbPath := flag.Arg(0)
	fragLibDir := flag.Arg(1)
	pdbFiles := flag.Args()[2:]

	if err := createBowDb(dbPath, fragLibDir, pdbFiles); err != nil {
		fatalf("%s\n", err)
	}

	var searchType int
	if flagInverted {
		searchType = bowdb.SearchInverted
	} else {
		searchType = bowdb.SearchFull
	}
	db, err := bowdb.Open(dbPath, searchType)
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

	bowOpts := bowdb.DefaultSearchOptions
	bowOpts.Limit = 200
	mattOpts := matt.DefaultConfig
	mattOpts.Verbose = false

	chains := createChains(pdbFiles)
	// mattArgs := createMattArgs(chains) 

	tabw := tabwriter.NewWriter(os.Stdout, 0, 4, 4, ' ', 0)
	header := []byte(
		"BOW entry\t" +
			"BOW chain\t" +
			"BOW dist\n")
	// "Matt entry\t" + 
	// "Matt chain\t" + 
	// "Matt dist\n") 
	for _, chain := range chains {
		// marg := mattArgs[i] 
		bow := db.Library.NewBowChain(chain)

		bowOrdered, err := getBowOrdering(db, bowOpts, bow)
		if err != nil {
			errorf("Could not get BOW ordering for %s (chain %c): %s\n",
				chain.Entry.IdCode, chain.Ident, err)
			continue
		}

		// mattOrdered, err := getMattOrdering(mattOpts, marg, mattArgs) 
		// if err != nil { 
		// errorf("Could not get Matt ordering for %s (chain %c): %s\n", 
		// chain.Entry.IdCode, chain.Ident, err) 
		// continue 
		// } 

		fmt.Printf("Ordering for %s (chain %c)\n",
			chain.Entry.IdCode, chain.Ident)

		// compared := comparison([2]ordering{bowOrdered, bowOrdered}) 
		tabw.Write(header)
		tabw.Write([]byte(bowOrdered.String()))
		tabw.Flush()
		fmt.Println("\n")
	}

	if err := db.ReadClose(); err != nil {
		fatalf("There was an error closing the database: %s\n", err)
	}
}

func createBowDb(dbPath string, fragLibDir string, pdbFiles []string) error {
	if _, err := os.Stat(dbPath); err == nil || os.IsExist(err) {
		return nil
	}

	args := []string{dbPath, fragLibDir}
	args = append(args, pdbFiles...)
	cmd := exec.Command("create-pdb-db", args...)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("Error running '%s': %s.",
			strings.Join(cmd.Args, " "), err)
	}
	return nil
}

func init() {
	flag.BoolVar(&flagInverted, "inverted", flagInverted,
		"When set, the search will use an inverted index.")
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
	errorf("Usage: %s database-path frag-lib-directory "+
		"query-pdb-file [query-pdb-file ...]\n",
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

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
