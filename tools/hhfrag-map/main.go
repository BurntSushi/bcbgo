package main

import (
	"flag"

	"github.com/BurntSushi/bcbgo/cmd/util"
	"github.com/BurntSushi/bcbgo/hhfrag"
)

var (
	hhfragConf = hhfrag.DefaultConfig
)

func init() {
	flag.IntVar(&hhfragConf.WindowMin, "win-min", hhfragConf.WindowMin,
		"The minimum HMM window size for HHfrag.")
	flag.IntVar(&hhfragConf.WindowMax, "win-max", hhfragConf.WindowMax,
		"The maximum HMM window size for HHfrag.")
	flag.IntVar(&hhfragConf.WindowIncrement, "win-inc",
		hhfragConf.WindowIncrement,
		"The sliding window increment for HHfrag.")

	util.FlagUse("cpu", "seq-db", "pdb-hhm-db", "blits")
	util.FlagParse("target-fasta out-fmap", "")
	util.AssertNArg(2)
}

func main() {
	fasInp := util.Arg(0)
	fmapOut := util.Arg(1)

	hhfragConf.Blits = util.FlagBlits
	fmap, err := hhfragConf.MapFromFasta(
		util.FlagPdbHhmDB, util.FlagSeqDB, fasInp)
	util.Assert(err, "Could not generate map from '%s'", fasInp)

	util.FmapWrite(util.CreateFile(fmapOut), fmap)
}
