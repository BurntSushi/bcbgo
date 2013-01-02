package main

import (
	"encoding/csv"
	"encoding/gob"
	"flag"
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/hhfrag"
)

type results []result

type result struct {
	needle  string
	results bowdb.SearchResults
}

var (
	flagBlits      = false
	flagSeqDB      = "nr20"
	flagPdbDB      = "pdb-select25"
	flagGoMaxProcs = runtime.NumCPU()
	flagQuiet      = false
	flagCsv        = false
	flagLimit      = 25

	seqDB hhsuite.Database
	pdbDB hhfrag.PDBDatabase
)

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

func main() {
	if flag.NArg() < 2 {
		usage()
	}
	dbPath := flag.Arg(0)
	queryFiles := flag.Args()[1:]

	db, err := bowdb.Open(dbPath)
	if err != nil {
		fatalf("%s\n", err)
	}

	opts := bowdb.DefaultSearchOptions
	opts.Limit = flagLimit

	searcher, err := db.NewFullSearcher()
	if err != nil {
		fatalf("Could not initialize full searcher: %s\n", err)
	}

	allResults := make(results, 0, 100)
	for _, queryFile := range queryFiles {
		fmap, err := getFmap(queryFile)
		if err != nil {
			errorf("Could not get fragment map for '%s' because: %s\n",
				queryFile, err)
			continue
		}

		results, err := searcher.Search(opts, fmap.BOW(db.Library))
		if err != nil {
			errorf("Could not get search results for query %s: %s\n",
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
	if err := db.ReadClose(); err != nil {
		fatalf("There was an error closing the database: %s\n", err)
	}
}

func getFmap(qfile string) (hhfrag.FragmentMap, error) {
	if strings.HasSuffix(qfile, ".fmap") {
		f, err := os.Open(qfile)
		if err != nil {
			return nil, err
		}

		var fmap hhfrag.FragmentMap
		r := gob.NewDecoder(f)
		if err := r.Decode(&fmap); err != nil {
			return nil, err
		}
		return fmap, nil
	}

	conf := hhfrag.DefaultConfig
	conf.Blits = flagBlits
	return conf.MapFromFasta(pdbDB, seqDB, qfile)
}

func init() {
	flag.BoolVar(&flagBlits, "blits", flagBlits,
		"When set, hhblits will be used in lieu of hhsearch.")
	flag.StringVar(&flagSeqDB, "seqdb", flagSeqDB,
		"The sequence database used to generate the query HHM.")
	flag.StringVar(&flagPdbDB, "pdbdb", flagPdbDB,
		"The PDB/HHM database used to assign fragments.")
	flag.BoolVar(&flagCsv, "csv", flagCsv,
		"When set, the search results will be printed in a CSV file format\n"+
			"\twith a tab delimiter.")
	flag.IntVar(&flagLimit, "limit", flagLimit,
		"The limit of results returned for each query chain.")
	flag.IntVar(&flagGoMaxProcs, "p", flagGoMaxProcs,
		"The maximum number of CPUs that can be executing simultaneously.")
	flag.BoolVar(&flagQuiet, "quiet", flagQuiet,
		"When set, no progress bar will be shown.\n"+
			"\tErrors will still be printed to stderr.")
	flag.Usage = usage
	flag.Parse()

	seqDB = hhsuite.Database(flagSeqDB)
	pdbDB = hhfrag.PDBDatabase(flagPdbDB)

	runtime.GOMAXPROCS(flagGoMaxProcs)
}

func usage() {
	errorf(
		"Usage: %s database-path fasta-or-fmap-file [fasta-or-fmap-file ...]\n",
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
