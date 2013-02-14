package main

import (
	"github.com/BurntSushi/bcbgo/bow"
	"github.com/BurntSushi/bcbgo/cmd/util"
)

func init() {
	util.FlagUse("cpu")
	util.FlagParse("frag-lib-dir fmap-file out-bow", "")
	util.AssertNArg(3)
}

func main() {
	lib := util.FragmentLibrary(util.Arg(0))
	fmap := util.FmapRead(util.Arg(1))

	bow := bow.ComputeBOW(lib, fmap)
	util.BOWWrite(util.CreateFile(util.Arg(2)), bow)
}
