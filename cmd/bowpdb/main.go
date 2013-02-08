package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"math"
	"os"
	"runtime/pprof"

	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

var (
	flagChain      = ""
	flagCpuProfile = ""
	flagCsv        = false
	flagInverted   = false
	flagLimit      = 25
)

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

	util.FlagUse("cpu", "verbose")
	util.FlagParse("bowdb-path query-pdb-file [query-pdb-file ...]", "")
	util.AssertLeastNArg(2)
}

func main() {
	dbPath := flag.Arg(0)
	pdbFiles := flag.Args()[1:]

	if len(flagCpuProfile) > 0 {
		f := util.CreateFile(flagCpuProfile)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	db := util.OpenBOWDB(dbPath)

	opts := bowdb.DefaultSearchOptions
	opts.Limit = flagLimit

	var searcher bowdb.Searcher
	var err error
	if flagInverted {
		searcher, err = db.NewInvertedSearcher()
		util.Assert(err)
	} else {
		searcher, err = db.NewFullSearcher()
		util.Assert(err)
	}

	allResults := make(results, 0, 100)
	for _, pdbFile := range pdbFiles {
		entry, err := pdb.ReadPDB(pdbFile)
		if err != nil {
			util.Warnf("Could not parse PDB file '%s' because: %s\n",
				pdbFile, err)
			continue
		}

		for _, chain := range entry.Chains {
			if !chain.IsProtein() || !isChainUsable(chain) {
				continue
			}

			bow := db.Library.NewBowChain(chain)
			results, err := searcher.Search(opts, bow)
			if err != nil {
				util.Warnf("Could not get search results for PDB entry %s "+
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
	util.Assert(db.ReadClose())
}

type results []result

type result struct {
	entry   string
	chain   byte
	results bowdb.SearchResults
}

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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
