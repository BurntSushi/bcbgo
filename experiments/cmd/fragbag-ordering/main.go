// test fragbag-ordering does an all-against-all search of the specified BOW
// database, and outputs the ordering of each search.
package main

import (
	"fmt"

	"github.com/BurntSushi/bcbgo/bow"
	"github.com/BurntSushi/bcbgo/cmd/util"
)

func init() {
	util.FlagParse("frag-lib-path", "")
	util.AssertNArg(1)
}

func main() {
	db := util.OpenBOWDB(util.Arg(0))

	// Set our search options.
	bowOpts := bow.SearchDefault
	bowOpts.Limit = -1

	fmt.Println("QueryID\tResultID\tCosine\tEuclid")
	for _, entry := range db.Entries {
		results := db.SearchEntry(bowOpts, entry)

		for _, result := range results {
			fmt.Printf("%s\t%c\t%s\t%c\t%0.4f\t%0.4f\n",
				entry.Id, result.Entry.Id, result.Cosine, result.Euclid)
		}
		fmt.Println("")
	}

	util.Assert(db.Close())
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
