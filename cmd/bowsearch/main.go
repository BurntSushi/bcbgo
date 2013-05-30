package main

import (
	"flag"
	"fmt"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"text/tabwriter"

	"github.com/BurntSushi/bcbgo/bow"
	"github.com/BurntSushi/bcbgo/cmd/util"
)

var (
	flagOutput = "plain"
	flagChain  = ""

	searchOpts = bow.SearchDefault
)

func init() {
	flag.StringVar(&flagOutput, "output", flagOutput,
		"The output format of the search results. Valid values are\n"+
			"'plain' and 'csv'.")
	flag.StringVar(&flagChain, "chain", flagChain,
		"Can be set to one or more single character chain identifiers.\n"+
			"This will restrict all PDB queries to only chains specified.\n"+
			"Has no effect on FASTA or fmap protein files.")

	// Search options.
	flag.IntVar(&searchOpts.Limit, "limit", searchOpts.Limit,
		"The maximum number of search results to return.\n"+
			"To specify no limit, set this to -1.")
	flag.Float64Var(&searchOpts.Min, "min", searchOpts.Min,
		"All search results will have at least this distance with the query.")
	flag.Float64Var(&searchOpts.Max, "max", searchOpts.Max,
		"All search results will have at most this distance with the query.")

	flagDesc := false
	flag.BoolVar(&flagDesc, "desc", flagDesc,
		"When set, results will be shown in descending order.")

	flagSort := "cosine"
	flag.StringVar(&flagSort, "sort", flagSort,
		"The field to sort search results by.\n"+
			"Valid values are 'cosine' and 'euclid'.")

	util.FlagUse("cpu", "cpuprof", "verbose", "seq-db", "pdb-hhm-db", "blits",
		"hhfrag-min", "hhfrag-max", "hhfrag-inc")
	util.FlagParse(
		"bowdb-path protein-files",
		"Where protein files can be files or directories that will be\n"+
			"searched recursively for FASTA, fmap or PDB files.")
	util.AssertLeastNArg(2)

	// Convert command line flag values to search option values.
	if flagDesc {
		searchOpts.Order = bow.OrderDesc
	}
	switch flagSort {
	case "cosine":
		searchOpts.SortBy = bow.Cosine
	case "euclid":
		searchOpts.SortBy = bow.Euclid
	default:
		util.Fatalf("Unknown sort field '%s'.", flagSort)
	}
}

func main() {
	if len(util.FlagCpuProf) > 0 {
		f := util.CreateFile(util.FlagCpuProf)
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	db, err := bow.OpenDB(util.Arg(0))
	util.Assert(err)

	out, outDone := outputter()
	bowerChan := make(chan bow.StructureBower)
	wg := new(sync.WaitGroup)
	for i := 0; i < max(1, runtime.GOMAXPROCS(0)); i++ {
		go func() {
			wg.Add(1)
			for bower := range bowerChan {
				out <- searchResult{bower, db.Search(searchOpts, bower)}
			}
			wg.Done()
		}()
	}

	files := util.AllFilesFromArgs(util.Args()[1:])
	for _, file := range files {
		if util.IsFasta(file) || util.IsFmap(file) {
			bowerChan <- util.GetFmap(file)
		} else if util.IsPDB(file) {
			entry := util.PDBRead(file)
			for _, chain := range entry.Chains {
				if chain.IsProtein() {
					if len(flagChain) == 0 ||
						strings.Contains(flagChain, string(chain.Ident)) {

						bowerChan <- chain
					}
				}
			}
		} else {
			util.Warnf("Unrecognized protein file: '%s'.", file)
		}
	}

	close(bowerChan)
	wg.Wait()
	close(out)
	<-outDone
	util.Assert(db.Close())
}

type searchResult struct {
	query   bow.StructureBower
	results []bow.SearchResult
}

func outputter() (chan searchResult, chan struct{}) {
	out := make(chan searchResult)
	done := make(chan struct{})
	go func() {
		if flagOutput == "csv" {
			fmt.Printf("QueryID\tHitID\tCosine\tEuclid\n")
		}

		first := true
		for sr := range out {
			switch flagOutput {
			case "plain":
				outputPlain(sr, first)
			case "csv":
				outputCsv(sr, first)
			default:
				util.Fatalf("Invalid output format '%s'.", flagOutput)
			}
			first = false
		}
		done <- struct{}{}
	}()
	return out, done
}

func outputPlain(sr searchResult, first bool) {
	w := tabwriter.NewWriter(os.Stdout, 5, 0, 4, ' ', 0)
	wf := func(format string, v ...interface{}) {
		fmt.Fprintf(w, format, v...)
	}

	if !first {
		fmt.Println(strings.Repeat("-", 80))
	}
	header := fmt.Sprintf("%s (%d hits)", sr.query.Id(), len(sr.results))

	fmt.Println(header)
	fmt.Println(strings.Repeat("-", len(header)))
	wf("Hit\tCosine\tEuclid\n")
	for _, result := range sr.results {
		wf("%s\t%0.4f\t%0.4f\n", result.Entry.Id, result.Cosine, result.Euclid)
	}
	w.Flush()
}

func outputCsv(sr searchResult, first bool) {
	for _, result := range sr.results {
		fmt.Printf("%s\t%s\t%0.4f\t%0.4f\n",
			sr.query.Id(), result.Entry.Id, result.Cosine, result.Euclid)
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
