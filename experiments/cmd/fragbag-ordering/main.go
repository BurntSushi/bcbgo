// test fragbag-ordering does an all-against-all search of the specified BOW
// database, and outputs the ordering of each search.
package main

import (
	"fmt"

	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/cmd/util"
)

func init() {
	util.FlagParse("frag-lib-path", "")
	util.AssertNArg(1)
}

func main() {
	db := util.OpenBOWDB(util.Arg(0))

	// Read in all of the entires in the BOW database.
	entries, err := db.ReadAll()
	util.Assert(err)

	searcher, err := db.NewFullSearcher()
	util.Assert(err, "Could not initialize searcher")

	// Set our search options.
	bowOpts := bowdb.DefaultSearchOptions
	bowOpts.Limit = len(entries)

	fmt.Println("QueryID\tQueryChain\tResultID\tResultChain" +
		"\tEuclid\tCosine")
	for _, entry := range entries {
		results, err := searcher.Search(bowOpts, entry.BOW)
		if err != nil {
			util.Warnf("Could not get BOW ordering for %s (chain %c): %s\n",
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

	util.Assert(db.ReadClose())
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
