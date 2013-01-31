package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"os"
	"path"
	"runtime"
	"runtime/pprof"

	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

type results []result

type result struct {
	entry   string
	chain   byte
	results bowdb.SearchResults
}

var (
	flagChain      = ""
	flagCpuProfile = ""
	flagGoMaxProcs = runtime.NumCPU()
	flagQuiet      = false
	flagCsv        = false
	flagInverted   = false
	flagLimit      = 25
)

func outputResults(results results) {
	switch {
	case flagCsv:
		csvWriter := csv.NewWriter(os.Stdout)
		csvWriter.Comma = '\t'
		csvWriter.UseCRLF = false
		csvWriter.Write([]string{
			"query_pdb", "query_chain",
			"hit_pdb", "hit_chain",
			"hit_cosine", "hit_euclid",
		})
		for _, query := range results {
			for _, result := range query.results.Results {
				csvWriter.Write([]string{
					query.entry, fmt.Sprintf("%c", query.chain),
					result.IdCode, fmt.Sprintf("%c", result.ChainIdent),
					fmt.Sprintf("%f", math.Abs(result.Cosine)),
					fmt.Sprintf("%f", math.Abs(result.Euclid)),
				})
			}
		}
		csvWriter.Flush()
	default:
		for _, query := range results {
			fmt.Printf("Search query: %s (chain: %c)\n",
				query.entry, query.chain)
			for _, result := range query.results.Results {
				fmt.Printf("%s\t%c\t%0.4f\n",
					result.IdCode, result.ChainIdent,
					math.Abs(result.Cosine))
			}
			fmt.Printf("\n")
		}
	}
}

func main() {
	if flag.NArg() < 2 {
		usage()
	}
	dbPath := flag.Arg(0)
	pdbFiles := flag.Args()[1:]

	if len(flagCpuProfile) > 0 {
		f, err := os.Create(flagCpuProfile)
		if err != nil {
			fatalf("%s\n", err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	db, err := bowdb.Open(dbPath)
	if err != nil {
		fatalf("%s\n", err)
	}

	opts := bowdb.DefaultSearchOptions
	opts.Limit = flagLimit

	var searcher bowdb.Searcher
	if flagInverted {
		searcher, err = db.NewInvertedSearcher()
		if err != nil {
			fatalf("Could not initialize inverted searcher: %s\n", err)
		}
	} else {
		searcher, err = db.NewFullSearcher()
		if err != nil {
			fatalf("Could not initialize full searcher: %s\n", err)
		}
	}

	allResults := make(results, 0, 100)
	for _, pdbFile := range pdbFiles {
		entry, err := pdb.ReadPDB(pdbFile)
		if err != nil {
			errorf("Could not parse PDB file '%s' because: %s\n", pdbFile, err)
			continue
		}

		for _, chain := range entry.Chains {
			if !chain.IsProtein() || !isChainUsable(chain) {
				continue
			}

			bow := db.Library.NewBowChain(chain)
			results, err := searcher.Search(opts, bow)
			if err != nil {
				errorf("Could not get search results for PDB entry %s "+
					"(chain %c): %s\n", entry.IdCode, chain.Ident, err)
				continue
			}

			chainResult := result{
				entry:   entry.IdCode,
				chain:   chain.Ident,
				results: results,
			}
			allResults = append(allResults, chainResult)
		}
	}

	outputResults(allResults)
	if err := db.ReadClose(); err != nil {
		fatalf("There was an error closing the database: %s\n", err)
	}
}

func init() {
	flag.StringVar(&flagChain, "chain", flagChain,
		"This may be set to one or more chain identifiers. Only chains "+
			"belonging to a chain specified will be used as a query.")
	flag.BoolVar(&flagCsv, "csv", flagCsv,
		"When set, the search results will be printed in a CSV file format\n"+
			"\twith a tab delimiter.")
	flag.BoolVar(&flagInverted, "inverted", flagInverted,
		"When set, the search will use an inverted index.")
	flag.IntVar(&flagLimit, "limit", flagLimit,
		"The limit of results returned for each query chain.")
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

func isChainUsable(chain *pdb.Chain) bool {
	if len(flagChain) == 0 {
		return true
	}
	for i := 0; i < len(flagChain); i++ {
		if chain.Ident == flagChain[i] {
			return true
		}
	}
	return false
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
