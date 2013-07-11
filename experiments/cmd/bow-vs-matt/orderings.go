package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/BurntSushi/bcbgo/apps/matt"
	"github.com/BurntSushi/bcbgo/bow"
	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/TuftsBCB/io/pdb"
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
	dist   float64
}

func (c chain) String() string {
	return fmt.Sprintf("%s\t%0.4f", c.idCode, c.dist)
}

func getBowOrdering(db *bow.DB,
	opts bow.SearchOptions, bower bow.Bower) ordering {

	results := db.Search(opts, bower)

	ordered := make(ordering, len(results))
	for i, result := range results {
		ordered[i] = chain{
			idCode: result.Entry.Id,
			dist:   result.Cosine,
		}
	}

	// These are already sorted.
	return ordered
}

func getMattOrdering(
	conf matt.Config, query matt.PDBArg, rest []matt.PDBArg) ordering {

	argsets := make([][]matt.PDBArg, len(rest))
	for i, target := range rest {
		argsets[i] = []matt.PDBArg{query, target}
	}

	results, errs := conf.RunAll(argsets)
	ordered := make(ordering, 0, len(rest))
	for i, result := range results {
		target := argsets[i][1]
		if errs[i] != nil {
			util.Warnf("Could not get Matt RMSD for %s (chain %c) against "+
				"%s (chain %c): %s",
				query.IdCode, query.Chain, target.IdCode, target.Chain, errs[i])
			continue
		}
		ordered = append(ordered, chain{
			idCode: fmt.Sprintf("%s%c", target.IdCode, target.Chain),
			dist:   100 * (result.RMSD / float64(result.CoreLength)), // SAS
		})
	}
	sort.Sort(ordered)
	return ordered
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
		util.Warning(err, "Could not open PDB file '%s'", pdbFile)

		for _, chain := range entry.Chains {
			if !chain.IsProtein() {
				continue
			}
			chains = append(chains, chain)
		}
	}
	return chains
}
