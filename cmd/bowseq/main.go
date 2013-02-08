package main

import (
	"encoding/csv"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/hhfrag"
)

var (
	flagCsv   = false
	flagLimit = 25
)

func init() {
	flag.BoolVar(&flagCsv, "csv", flagCsv,
		"When set, the search results will be printed in a CSV file format\n"+
			"\twith a tab delimiter.")
	flag.IntVar(&flagLimit, "limit", flagLimit,
		"The limit of results returned for each query chain.")

	util.FlagUse("cpu", "seq-db", "pdb-hhm-db", "blits", "verbose")
	util.FlagParse("bowdb-path fasta-or-fmap-file [fasta-or-fmap-file ...]", "")
	util.AssertLeastNArg(2)
}

func main() {
	dbPath := util.Arg(0)
	queryFiles := flag.Args()[1:]

	db := util.OpenBOWDB(dbPath)

	opts := bowdb.DefaultSearchOptions
	opts.Limit = flagLimit

	searcher, err := db.NewFullSearcher()
	util.Assert(err)

	allResults := make(results, 0, 100)
	for _, queryFile := range queryFiles {
		fmap, err := getFmap(queryFile)
		if err != nil {
			util.Warnf("Could not get fragment map for '%s' because: %s\n",
				queryFile, err)
			continue
		}

		results, err := searcher.Search(opts, fmap.BOW(db.Library))
		if err != nil {
			util.Warnf("Could not get search results for query %s: %s\n",
				queryFile, err)
			continue
		}

		r := result{
			needle:  queryFile,
			results: results,
		}
		allResults = append(allResults, r)
	}

	outputResults(allResults)
	util.Assert(db.ReadClose())
}

type results []result

type result struct {
	needle  string
	results bowdb.SearchResults
}

func outputResults(results results) {
	switch {
	case flagCsv:
		csvWriter := csv.NewWriter(os.Stdout)
		csvWriter.Comma = '\t'
		csvWriter.UseCRLF = false
		csvWriter.Write([]string{
			"query", "hit_pdb", "hit_chain", "hit_cosine", "hit_euclid",
		})
		for _, query := range results {
			for _, result := range query.results.Results {
				csvWriter.Write([]string{
					query.needle,
					result.IdCode, fmt.Sprintf("%c", result.ChainIdent),
					fmt.Sprintf("%f", result.Cosine),
					fmt.Sprintf("%f", result.Euclid),
				})
			}
		}
		csvWriter.Flush()
	default:
		for _, query := range results {
			fmt.Printf("Search query: %s\n", query.needle)
			for _, result := range query.results.Results {
				fmt.Printf("%s\t%c\t%0.4f\n",
					result.IdCode, result.ChainIdent, result.Cosine)
			}
			fmt.Printf("\n")
		}
	}
}

func getFmap(qfile string) (hhfrag.FragmentMap, error) {
	if strings.HasSuffix(qfile, ".fmap") {
		return util.FmapRead(qfile), nil
	}

	conf := hhfrag.DefaultConfig
	conf.Blits = util.FlagBlits
	return conf.MapFromFasta(util.FlagPdbHhmDB, util.FlagSeqDB, qfile)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
