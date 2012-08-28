package bowdb

import (
	"fmt"

	"github.com/BurntSushi/bcbgo/fragbag"
)

const (
	Euclid = iota
)

const (
	OrderAsc = iota
	OrderDesc
)

type searcher interface {
	search(opts SearchOptions, bow fragbag.BOW) SearchResults
}

type SearchOptions struct {
	Limit     int
	Threshold float64
	SearchBy  int
	SortBy    int
	Order     int
}

var DefaultSearchOptions = SearchOptions{
	Limit:     25,
	Threshold: 0.0,
	SearchBy:  Euclid,
	SortBy:    Euclid,
	Order:     OrderAsc,
}

type SearchResults struct {
	SearchOptions
	Results []SearchResult
}

func (srs SearchResults) Len() int {
	return len(srs.Results)
}

func (srs SearchResults) Less(i, j int) bool {
	var orderCmp func(a, b float64) bool
	switch srs.Order {
	case OrderAsc:
		orderCmp = func(a, b float64) bool { return a < b }
	case OrderDesc:
		orderCmp = func(a, b float64) bool { return a >= b }
	default:
		panic(fmt.Sprintf("Unknown order type: %d.", srs.Order))
	}

	var valCmp func(r SearchResult) float64
	switch srs.SortBy {
	case Euclid:
		valCmp = func(r SearchResult) float64 { return r.Euclid }
	default:
		panic(fmt.Sprintf("Unknown sort type: %d.", srs.SortBy))
	}

	return orderCmp(valCmp(srs.Results[i]), valCmp(srs.Results[j]))
}

func (srs SearchResults) Swap(i, j int) {
	srs.Results[i], srs.Results[j] = srs.Results[j], srs.Results[i]
}

type PDBItem struct {
	IdCode         string
	Classification string
}

type SearchResult struct {
	PDBItem
	Euclid float64
}

// better is satisfied when sr1 is a better search result than sr2 according
// to the search options.
func (sr1 SearchResult) better(opts SearchOptions, sr2 SearchResult) bool {
	switch opts.SearchBy {
	case Euclid:
		return sr1.Euclid < sr2.Euclid
	default:
		panic(fmt.Sprintf("Unknown search type: %d.", opts.SearchBy))
	}
	panic("unreachable")
}

func (sr SearchResult) String() string {
	return fmt.Sprintf("%s\t%s\t%0.4f",
		sr.IdCode, sr.Classification, sr.Euclid)
}

type searchItem struct {
	PDBItem
	fragbag.BOW
}
