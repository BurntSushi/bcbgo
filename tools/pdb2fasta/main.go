package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/io/fasta"
	"github.com/BurntSushi/bcbgo/io/pdb"
	"github.com/BurntSushi/bcbgo/seq"
)

var (
	flagChain          = ""
	flagSeparateChains = false
	flagSplit          = ""
)

func init() {
	flag.BoolVar(&flagSeparateChains, "separate-chains", flagSeparateChains,
		"When set, each chain will get its own FASTA entry.")
	flag.StringVar(&flagChain, "chain", flagChain,
		"This may be set to one or more chain identifiers. Only amino acids "+
			"belonging to a chain specified will be included.\n"+
			"If this is set, then 'split' MUST also be set.")
	flag.StringVar(&flagSplit, "split", flagSplit,
		"When set, each FASTA entry produced will be written to a file in the "+
			"specified directory with the PDB id code and chain identifier as "+
			"the name.")

	util.FlagParse("in-pdb-file [out-fasta-file]", "")

	if util.NArg() != 1 && util.NArg() != 2 {
		util.Usage()
	}
	if len(flagChain) > 0 && len(flagSplit) == 0 {
		util.Fatalf("The '-chain' option must be accompanied by the " +
			"'-split' option.")
	}
}

func main() {
	pdbEntry := util.PDBRead(flag.Arg(0))

	fasEntries := make([]seq.Sequence, 0, 5)
	if !flagSeparateChains {
		var fasEntry seq.Sequence
		if len(pdbEntry.Chains) == 1 {
			fasEntry.Name = chainHeader(pdbEntry.OneChain())
		} else {
			fasEntry.Name = fmt.Sprintf("%s", strings.ToLower(pdbEntry.IdCode))
		}

		seq := make([]seq.Residue, 0, 100)
		for _, chain := range pdbEntry.Chains {
			if isChainUsable(chain) {
				seq = append(seq, chain.Sequence...)
			}
		}
		fasEntry.Residues = seq

		if len(fasEntry.Residues) == 0 {
			util.Fatalf("Could not find any amino acids.")
		}
		fasEntries = append(fasEntries, fasEntry)
	} else {
		for _, chain := range pdbEntry.Chains {
			if !isChainUsable(chain) {
				continue
			}

			fasEntry := seq.Sequence{
				Name:     chainHeader(chain),
				Residues: chain.Sequence,
			}
			fasEntries = append(fasEntries, fasEntry)
		}
	}
	if len(fasEntries) == 0 {
		util.Fatalf("Could not find any chains with amino acids.")
	}

	var fasOut io.Writer
	if flag.NArg() == 1 {
		fasOut = os.Stdout
	} else {
		if len(flagSplit) > 0 {
			util.Fatalf("The '--split' option is incompatible with a single " +
				"output file.")
		}
		fasOut = util.CreateFile(util.Arg(1))
	}

	if len(flagSplit) == 0 {
		util.Assert(fasta.NewWriter(fasOut).WriteAll(fasEntries),
			"Could not write FASTA file '%s'", fasOut)
	} else {
		for _, entry := range fasEntries {
			fp := path.Join(flagSplit, fmt.Sprintf("%s.fasta", entry.Name))
			out := util.CreateFile(fp)

			w := fasta.NewWriter(out)
			util.Assert(w.Write(entry), "Could not write to '%s'", fp)
			util.Assert(w.Flush(), "Could not write to '%s'", fp)
		}
	}
}

func chainHeader(chain *pdb.Chain) string {
	return fmt.Sprintf("%s%c", strings.ToLower(chain.Entry.IdCode), chain.Ident)
}

func isChainUsable(chain *pdb.Chain) bool {
	if len(flagChain) == 0 {
		return true
	}
	for i := 0; i < len(flagChain); i++ {
		if chain.Ident == flagChain[i] {
			return true
		}
	}
	return false
}
