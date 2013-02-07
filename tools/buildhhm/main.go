// Example buildhhm shows how to construct an HHM using HHblits and HHmake
// from a single sequence FASTA file.
package main

import (
	"flag"

	"github.com/BurntSushi/bcbgo/apps/hhsuite"
	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/io/hhm"
)

var (
	flagQuiet = false
)

func init() {
	flag.BoolVar(&flagQuiet, "quiet", flagQuiet,
		"When set, hhblits/hhmake output will be hidden.")

	util.FlagUse("seq-db")
	util.FlagParse("in-fasta-file out-hhm-file", "")
	util.AssertNArg(2)
}

func main() {
	inFasta := util.Arg(0)
	outHHM := util.Arg(1)

	hhblits := hhsuite.HHBlitsDefault
	hhmake := hhsuite.HHMakePseudo
	hhblits.Verbose = !flagQuiet
	hhmake.Verbose = !flagQuiet

	HHM, err := hhsuite.BuildHHM(
		hhblits, hhmake, util.FlagSeqDB, inFasta)
	util.Assert(err, "Error building HHM")

	util.Assert(hhm.Write(util.CreateFile(outHHM), HHM),
		"Error writing HHM '%s'", outHHM)
}
