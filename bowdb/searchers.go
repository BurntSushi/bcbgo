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

func (db *DB) NewFullSearcher() (Searcher, error) {
	items, err := db.files.read()
	if err != nil {
		return nil, err
	}
	return searchFull(items), nil
}

func (sf searchFull) Search(
	opts SearchOptions, bow fragbag.BOW) (SearchResults, error) {

	return search([]searchItem(sf), opts, bow), nil
}

func (db *DB) NewInvertedSearcher() (Searcher, error) {
	// We don't do anything here because we don't know what bows we're
	// searching for yet.
	return searchInverted{db}, nil
}

func (si searchInverted) Search(
	opts SearchOptions, bow fragbag.BOW) (SearchResults, error) {

	set := make(map[sequenceId]bool, 100)
	added := false
	for i := 0; i < bow.Len(); i++ {
		// Only include search items with a bow fragment in common.
		if bow.Frequency(i) == 0 {
			continue
		}

		seqs, err := si.DB.files.getInvertedList(i)
		if err != nil {
			return SearchResults{}, err
		}
		if len(seqs) == 0 {
			continue
		}

		// Do set intersection.
		// I'm being stupid, I know it. Brain cramp.
		if !added {
			for _, seqId := range seqs {
				set[seqId] = true
			}
		} else {
			for k := range set {
				set[k] = false
			}
			for _, seqId := range seqs {
				if _, ok := set[seqId]; ok {
					set[seqId] = true
				}
			}
			for k := range set {
				if !set[k] {
					delete(set, k)
				}
			}
		}
	}

	items := make([]searchItem, 0, len(set))
	for seqId := range set {
		item, err := si.DB.files.readIndexed(seqId)
		if err != nil {
			return SearchResults{}, err
		}
		items = append(items, item)
	}
	fmt.Printf("Searching %d items\n", len(items))
	return search(items, opts, bow), nil
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
