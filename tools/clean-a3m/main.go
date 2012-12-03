// clean-a3m will read in all sequences in a given A3M file and overwrite the
// the given file with all non-empty sequences.
//
// This is necessary to clean up hhsuite's messes. Namely, when adding DSSP
// secondary structure to A3M files, it add an empty ">ss_dssp", which
// of course, hhblits chokes on.
package main

import (
	"flag"
	"log"
	"os"

	"github.com/BurntSushi/bcbgo/io/fasta"
)

func init() {
	log.SetFlags(0)

	flag.Usage = usage
	flag.Parse()
}

func usage() {
	log.Println("Usage: clean-a3m [flags] a3m-file")
	flag.PrintDefaults()
}

func main() {
	if flag.NArg() != 1 {
		usage()
	}

	a3mPath := flag.Arg(0)
	fra3m, err := os.Open(a3mPath)
	assert(err)

	freader := fasta.NewReader(fra3m)
	freader.TrustSequences = true
	seqs, err := freader.ReadAll()
	assert(err)
	assert(fra3m.Close())

	fwa3m, err := os.Create(a3mPath)
	assert(err)
	fwriter := fasta.NewWriter(fwa3m)
	fwriter.Columns = 0
	for _, seq := range seqs {
		if len(seq.Residues) > 0 {
			fwriter.Write(seq)
		}
	}
	assert(fwriter.Flush())
	assert(fwa3m.Close())
}

func assert(err error) {
	if err != nil {
		log.Fatalf("[%s]: %s", flag.Arg(0), err)
	}
}
