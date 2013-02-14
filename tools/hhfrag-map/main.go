package main

import (
	"github.com/BurntSushi/bcbgo/cmd/util"
)

func init() {
	util.FlagUse("cpu", "seq-db", "pdb-hhm-db", "blits",
		"hhfrag-min", "hhfrag-max", "hhfrag-inc")
	util.FlagParse("target-fasta out-fmap", "")
	util.AssertNArg(2)
}

func main() {
	fasInp := util.Arg(0)
	fmapOut := util.Arg(1)

	fmap := util.GetFmap(fasInp)
	util.FmapWrite(util.CreateFile(fmapOut), fmap)
}
