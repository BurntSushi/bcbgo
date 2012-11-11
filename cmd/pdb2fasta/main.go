package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/BurntSushi/bcbgo/io/fasta"
	"github.com/BurntSushi/bcbgo/io/pdb"
)

var (
	flagChain          = ""
	flagSeparateChains = false
	flagSeqRes         = false
)

func main() {
	if flag.NArg() < 1 || flag.NArg() > 2 {
		usage()
	}

	pdbEntry, err := pdb.New(flag.Arg(0))
	if err != nil {
		fatalf("Could not read PDB file '%s': %s", flag.Arg(0), err)
	}

	var fasOut io.Writer
	if flag.NArg() == 1 {
		fasOut = os.Stdout
	} else {
		fasOut, err = os.Create(flag.Arg(1))
		if err != nil {
			fatalf("Could not create FASTA file '%s': %s", flag.Arg(1), err)
		}
	}

	fasEntries := make([]fasta.Entry, 0, 5)
	if !flagSeparateChains {
		var fasEntry fasta.Entry
		if len(pdbEntry.Chains) == 1 {
			fasEntry.Header = chainHeader(pdbEntry.OneChain())
		} else {
			fasEntry.Header = fmt.Sprintf("%s", pdbEntry.IdCode)
		}

		seq := make([]byte, 0, 100)
		for _, chain := range pdbEntry.Chains {
			if isChainUsable(chain) {
				seq = append(seq, getChainSequence(chain)...)
			}
		}
		fasEntry.Sequence = seq

		if len(fasEntry.Sequence) == 0 {
			fatalf("Could not find any amino acids.")
		}
		fasEntries = append(fasEntries, fasEntry)
	} else {
		for _, chain := range pdbEntry.Chains {
			if !isChainUsable(chain) {
				continue
			}

			fasEntry := fasta.Entry{
				Header:   chainHeader(chain),
				Sequence: getChainSequence(chain),
			}
			fasEntries = append(fasEntries, fasEntry)
		}
	}

	if len(fasEntries) == 0 {
		fatalf("Could not find any chains with amino acids.")
	}
	if err := fasta.NewWriter(fasOut).WriteAll(fasEntries); err != nil {
		fatalf("Could not write FASTA: %s", err)
	}
}

func chainHeader(chain *pdb.Chain) string {
	return fmt.Sprintf("%s%c", strings.ToLower(chain.Entry.IdCode), chain.Ident)
}

func getChainSequence(chain *pdb.Chain) []byte {
	if flagSeqRes {
		return chain.Sequence
	}
	return chain.CaSequence
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

func fatalf(format string, v ...interface{}) {
	fmt.Fprintf(os.Stderr, format, v...)
	fmt.Fprintln(os.Stderr, "")
	os.Exit(1)
}

func init() {
	flag.BoolVar(&flagSeparateChains, "separate-chains", flagSeparateChains,
		"When set, each chain will get its own FASTA entry.")
	flag.StringVar(&flagChain, "chain", flagChain,
		"This may be set to one or more chain identifiers. Only amino acids "+
			"belonging to a chain specified will be included.")
	flag.BoolVar(&flagSeqRes, "seqres", flagSeqRes,
		"When set, sequences will be read from the SEQRES records. Otherwise, "+
			"sequences are read from residues in Ca ATOM records.")
	flag.Usage = usage
	flag.Parse()
}

func usage() {
	fmt.Fprintf(os.Stderr,
		"Usage: %s pdb2fasta [flags] in-pdb-file [out-fasta-file]\n",
		path.Base(os.Args[0]))
	flag.PrintDefaults()
	os.Exit(1)
}
