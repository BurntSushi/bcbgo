package main

import (
	"flag"
	"fmt"

	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/TuftsBCB/io/pdb"
	"github.com/TuftsBCB/seq"
)

var flagChain = ""

func init() {
	flag.StringVar(&flagChain, "chain", flagChain,
		"When set, only this chain will be tested for a correspondence. "+
			"Otherwise, all chains will be tested.")

	util.FlagParse("pdb-file", "")
	util.AssertNArg(1)
}

func main() {
	entry := util.PDBRead(flag.Arg(0))

	if len(flagChain) > 0 {
		if len(flagChain) != 1 {
			util.Fatalf("Chain identifiers must be a single character.")
		}
		chain := entry.Chain(flagChain[0])
		if chain == nil {
			util.Fatalf("Could not find chain '%c' in PDB entry '%s'.",
				chain.Ident, entry.Path)
		}
		showMapping(chain, chain.SequenceAtoms())
	} else {
		for _, chain := range entry.Chains {
			showMapping(chain, chain.SequenceAtoms())
		}
	}
}

func showMapping(chain *pdb.Chain, mapped []*pdb.Residue) {
	id := fmt.Sprintf("%s%c", chain.Entry.IdCode, chain.Ident)
	fmt.Printf("%s SEQRES sequence:\n", id)
	fmt.Printf("%s\n\n", chain.Sequence)
	fmt.Printf("%s ATOM sequence:\n", id)
	fmt.Printf("%s\n\n", mappedSequence(mapped))
	fmt.Println("=========================================================")
}

func mappedSequence(mapped []*pdb.Residue) []seq.Residue {
	guess := make([]seq.Residue, len(mapped))
	for i, residue := range mapped {
		if residue == nil {
			guess[i] = '-'
		} else {
			guess[i] = residue.Name
		}
	}
	return guess
}
