// clean-a3m will read in all sequences in a given A3M file and overwrite the
// the given file with all non-empty sequences.
//
// This is necessary to clean up hhsuite's messes. Namely, when adding DSSP
// secondary structure to A3M files, it adds an empty ">ss_dssp", which
// of course, hhblits chokes on.
package main

import (
	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/io/fasta"
)

func init() {
	util.FlagParse("a3m-file", "")
	util.AssertNArg(1)
}

func main() {
	a3mPath := util.Arg(0)
	fa3m := util.OpenFile(a3mPath)

	freader := fasta.NewReader(fa3m)
	freader.TrustSequences = true
	seqs, err := freader.ReadAll()
	util.Assert(err, "Could not read fasta format '%s'", a3mPath)
	util.Assert(fa3m.Close())

	w := util.CreateFile(a3mPath)
	fwriter := fasta.NewWriter(w)
	fwriter.Columns = 0
	for _, seq := range seqs {
		if len(seq.Residues) > 0 {
			util.Assert(fwriter.Write(seq))
		}
	}
	util.Assert(fwriter.Flush())
	util.Assert(w.Close())
}
