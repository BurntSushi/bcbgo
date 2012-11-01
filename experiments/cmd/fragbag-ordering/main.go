// test fragbag-ordering does an all-against-all search of the specified BOW
// database, and outputs the ordering of each search.
package main

import (
	"flag"
	"fmt"
	"os"
	"path"

	"github.com/BurntSushi/bcbgo/bowdb"
)

var (
	flagInverted = false
)

func main() {
	if flag.NArg() < 1 {
		usage()
	}
	dbPath := flag.Arg(0)

	// Open the BOW database.
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

	// Read in all of the entires in the BOW database.
	entries, err := db.ReadAll()
	if err != nil {
		fatalf("%s\n", err)
	}

	// Set our search options.
	bowOpts := bowdb.DefaultSearchOptions
	bowOpts.Limit = len(entries)

	fmt.Println("QueryID\tQueryChain\tResultID\tResultChain" +
		"\tEuclid\tCosine")
	for _, entry := range entries {
		results, err := db.Search(bowOpts, entry.BOW)
		if err != nil {
			errorf("Could not get BOW ordering for %s (chain %c): %s\n",
				entry.IdCode, entry.ChainIdent, err)
			continue
		}

		for _, result := range results.Results {
			fmt.Printf("%s\t%c\t%s\t%c\t%0.4f\t%0.4f\n",
				entry.IdCode, entry.ChainIdent,
				result.IdCode, result.ChainIdent,
				result.Euclid, result.Cosine)
		}
		fmt.Println("")
	}

	if err := db.ReadClose(); err != nil {
		fatalf("There was an error closing the database: %s\n", err)
	}
}

func init() {
	flag.BoolVar(&flagInverted, "inverted", flagInverted,
		"When set, the search will use an inverted index.")
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	errorf("Usage: %s database-path \n", path.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(1)
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
