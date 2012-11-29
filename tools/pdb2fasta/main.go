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
	"github.com/BurntSushi/bcbgo/seq"
)

var (
	flagChain          = ""
	flagSeparateChains = false
	flagSplit          = ""
)

func main() {
	if flag.NArg() < 1 || flag.NArg() > 2 {
		usage()
	}

	pdbEntry, err := pdb.ReadPDB(flag.Arg(0))
	if err != nil {
		fatalf("Could not read PDB file '%s': %s", flag.Arg(0), err)
	}

	var fasOut io.Writer
	if flag.NArg() == 1 {
		fasOut = os.Stdout
	} else {
		if len(flagSplit) > 0 {
			fatalf("The '--split' option is incompatible with a single " +
				"output file.")
		}
		fasOut, err = os.Create(flag.Arg(1))
		if err != nil {
			fatalf("Could not create FASTA file '%s': %s", flag.Arg(1), err)
		}
	}

	fasEntries := make([]seq.Sequence, 0, 5)
	if !flagSeparateChains {
		var fasEntry seq.Sequence
		if len(pdbEntry.Chains) == 1 {
			fasEntry.Name = chainHeader(pdbEntry.OneChain())
		} else {
			fasEntry.Name = fmt.Sprintf("%s", pdbEntry.IdCode)
		}

		seq := make([]seq.Residue, 0, 100)
		for _, chain := range pdbEntry.Chains {
			if isChainUsable(chain) {
				seq = append(seq, chain.Sequence...)
			}
		}
		fasEntry.Residues = seq

		if len(fasEntry.Residues) == 0 {
			fatalf("Could not find any amino acids.")
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
		fatalf("Could not find any chains with amino acids.")
	}
	if len(flagSplit) == 0 {
		if err := fasta.NewWriter(fasOut).WriteAll(fasEntries); err != nil {
			fatalf("Could not write FASTA: %s", err)
		}
	} else {
		for _, entry := range fasEntries {
			fp := path.Join(flagSplit, fmt.Sprintf("%s.fasta", entry.Name))
			out, err := os.Create(fp)
			if err != nil {
				fatalf("Could not create FASTA file: %s", err)
			}

			w := fasta.NewWriter(out)
			if err := w.Write(entry); err != nil {
				fatalf("Could not write to FASTA file: %s", err)
			}
			if err := w.Flush(); err != nil {
				fatalf("Could not write to FASTA file: %s", err)
			}
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
	flag.StringVar(&flagSplit, "split", flagSplit,
		"When set, each FASTA entry produced will be written to a file in the "+
			"specified directory with the PDB id code and chain identifier as "+
			"the name.")
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
