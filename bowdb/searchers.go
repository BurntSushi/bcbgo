package bowdb

import (
	"sort"

	"github.com/BurntSushi/bcbgo/fragbag"
)

type searchFull []searchItem

type searchInverted []searchItem

func newFullSearcher(db *DB) (searchFull, error) {
	items, err := db.files.read()
	if err != nil {
		return nil, err
	}
	return searchFull(items), nil
}

func (sf searchFull) search(
	opts SearchOptions, bow fragbag.BOW) SearchResults {

	results := make([]SearchResult, 0, max(1, opts.Limit))
	for _, item := range sf {
		sr := SearchResult{
			item.PDBItem,
			bow.Euclid(item.BOW),
		}

		// If we don't have any results yet, indiscriminately add this result
		// and move on.
		if len(results) == 0 {
			results = append(results, sr)
			continue
		}

		// This search result is better than our current worst search result.
		// Thus, add it to our result set in proper ascending order.
		// Finally, check to see if the results list is bigger than the
		// limit. If so, trim.
		worst := results[len(results)-1]
		if len(results) < opts.Limit || sr.better(opts, worst) {
			added := false
			for i := 0; i < len(results); i++ {
				if sr.better(opts, results[i]) {
					results = append(results[:i],
						append([]SearchResult{sr}, results[i:]...)...)
					added = true
					break
				}
			}
			if !added {
				results = append(results, sr)
			}
			if len(results) > opts.Limit {
				results = results[:opts.Limit]
			}
		}
	}
	srs := SearchResults{opts, results}
	sort.Sort(srs)
	return srs
}

func newInvertedSearcher(db *DB) (searchInverted, error) {
	return nil, nil
}

func (si searchInverted) search(
	opts SearchOptions, bow fragbag.BOW) SearchResults {

	return SearchResults{}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
