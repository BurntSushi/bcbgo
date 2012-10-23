package bowdb

import (
	"fmt"
	"sort"

	"github.com/BurntSushi/bcbgo/fragbag"
)

type searchFull []searchItem

type searchInverted struct {
	*DB
}

func newFullSearcher(db *DB) (searchFull, error) {
	items, err := db.files.read()
	if err != nil {
		return nil, err
	}
	return searchFull(items), nil
}

func (sf searchFull) search(
	opts SearchOptions, bow fragbag.BOW) (SearchResults, error) {

	return search([]searchItem(sf), opts, bow), nil
}

func newInvertedSearcher(db *DB) (searchInverted, error) {
	// We don't do anything here because we don't know what bows we're
	// searching for yet.
	return searchInverted{db}, nil
}

func (si searchInverted) search(
	opts SearchOptions, bow fragbag.BOW) (SearchResults, error) {

	set := make(map[string]bool, 100)
	allItems := make([]searchItem, 0, 100)
	for i := 0; i < bow.Len(); i++ {
		// Only include search items with a bow fragment in common.
		if bow.Frequency(i) == 0 {
			continue
		}

		items, err := si.DB.files.readInvertedSearchItem(i)
		if err != nil {
			return SearchResults{}, err
		}

		// Make sure we don't search duplicate search items.
		for _, item := range items {
			key := fmt.Sprintf("%s%c", item.IdCode, item.ChainIdent)
			if set[key] {
				continue
			}
			allItems = append(allItems, item)
			set[key] = true
		}
	}
	return search(allItems, opts, bow), nil
}

func search(items []searchItem,
	opts SearchOptions, bow fragbag.BOW) SearchResults {

	srs := newSearchResults(opts)
	for _, item := range items {
		sr := SearchResult{
			item.PDBItem,
			bow.Euclid(item.BOW),
			bow.Cosine(item.BOW),
			item.BOW,
		}
		srs.maybeAdd(sr)
	}
	sort.Sort(srs)
	return srs
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
