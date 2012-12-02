package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/bcbgo/io/pdb"
	"github.com/BurntSushi/bcbgo/seq"
)

var flagChain = ""

func init() {
	log.SetFlags(0)

	flag.StringVar(&flagChain, "chain", flagChain,
		"When set, only this chain will be tested for a correspondence. "+
			"Otherwise, all chains will be tested.")

	flag.Usage = usage
	flag.Parse()
}

func usage() {
	log.Println("Usage: pdb-test [flags] input-file")
	flag.PrintDefaults()
	os.Exit(1)
}

func main() {
	if flag.NArg() != 1 {
		usage()
	}

	entry, err := pdb.ReadPDB(flag.Arg(0))
	if err != nil {
		log.Fatalf("Could not read PBD file '%s': %s", entry.Path, err)
	}

	if len(flagChain) > 0 {
		if len(flagChain) != 1 {
			log.Fatalln("Chain identifiers must be a single character.")
		}
		chain := entry.Chain(flagChain[0])
		if chain == nil {
			log.Fatalf("Could not find chain '%c' in PDB entry '%s'.",
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
