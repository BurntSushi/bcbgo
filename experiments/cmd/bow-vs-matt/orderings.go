package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/bcbgo/apps/matt"
	"github.com/BurntSushi/bcbgo/bowdb"
	"github.com/BurntSushi/bcbgo/fragbag"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

type comparison [2]ordering

func (c comparison) String() string {
	least := min(len(c[0]), len(c[1]))
	lines := make([]string, least)
	for i := 0; i < least; i++ {
		lines[i] = fmt.Sprintf("%s\t%s", c[0][i], c[1][i])
	}
	return strings.Join(lines, "\n")
}

type ordering []chain

func (o ordering) Less(i, j int) bool {
	return o[i].dist < o[j].dist
}

func (o ordering) Len() int {
	return len(o)
}

func (o ordering) Swap(i, j int) {
	o[i], o[j] = o[j], o[i]
}

func (o ordering) String() string {
	lines := make([]string, len(o))
	for i, chain := range o {
		lines[i] = chain.String()
	}
	return strings.Join(lines, "\n")
}

type chain struct {
	idCode string
	ident  byte
	dist   float64
}

func (c chain) String() string {
	return fmt.Sprintf("%s\t%c\t%0.4f", c.idCode, c.ident, c.dist)
}

func getBowOrdering(searcher bowdb.Searcher,
	opts bowdb.SearchOptions, bow fragbag.BOW) (ordering, error) {

	results, err := searcher.Search(opts, bow)
	if err != nil {
		return nil, err
	}

	ordered := make(ordering, len(results.Results))
	for i, result := range results.Results {
		ordered[i] = chain{
			idCode: result.IdCode,
			ident:  result.ChainIdent,
			dist:   result.Cosine,
		}
	}

	// These are already sorted.
	return ordered, nil
}

func getMattOrdering(
	conf matt.Config, query matt.PDBArg, rest []matt.PDBArg) (ordering, error) {

	argsets := make([][]matt.PDBArg, len(rest))
	for i, target := range rest {
		argsets[i] = []matt.PDBArg{query, target}
	}

	results, errs := conf.RunAll(argsets)
	ordered := make(ordering, 0, len(rest))
	for i, result := range results {
		target := argsets[i][1]
		if errs[i] != nil {
			errorf("Could not get Matt RMSD for %s (chain %c) against "+
				"%s (chain %c): %s",
				query.IdCode, query.Chain, target.IdCode, target.Chain, errs[i])
			continue
		}
		ordered = append(ordered, chain{
			idCode: target.IdCode,
			ident:  target.Chain,
			dist:   100 * (result.RMSD / float64(result.CoreLength)), // SAS
		})
	}
	sort.Sort(ordered)
	return ordered, nil
}

func createMattArgs(chains []*pdb.Chain) []matt.PDBArg {
	args := make([]matt.PDBArg, len(chains))
	for i, chain := range chains {
		args[i] = matt.NewChainArg(chain)
	}
	return args
}

func createChains(pdbFiles []string) []*pdb.Chain {
	chains := make([]*pdb.Chain, 0, len(pdbFiles))
	for _, pdbFile := range pdbFiles {
		entry, err := pdb.ReadPDB(pdbFile)
		if err != nil {
			errorf("Could not parse PDB file '%s' because: %s\n", pdbFile, err)
			continue
		}

		for _, chain := range entry.Chains {
			if !chain.IsProtein() {
				continue
			}
			chains = append(chains, chain)
		}
	}
	return chains
}
